package main

import (
	"bytes"
	"essync/conf"
	"essync/lib"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/gin-gonic/gin"
	"github.com/phachon/go-logger"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var yaml_conf = conf.EsConfig{}
var logger = go_logger.NewLogger()

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Cli param config file is missing !")
	}
	configFile := os.Args[1]
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &yaml_conf)
	if err != nil {
		log.Fatalf(err.Error())
	}
	r := gin.New()
	r.Use(gin.Recovery())

	logger.Detach("console")
	consoleConfig := &go_logger.ConsoleConfig{
		Color:      true,                                           // Does the text display the color
		JsonFormat: false,                                          // Whether or not formatted into a JSON string
		Format:     "%millisecond_format% [%level_string%] %body%", // JsonFormat is false, logger message output to console format string
	}
	logger.Attach("console", go_logger.LOGGER_LEVEL_DEBUG, consoleConfig)

	fileConfig := &go_logger.FileConfig{
		//Filename: yaml_conf.LogDir + "essync.log", // The file name of the logger output, does not exist automatically
		// If you want to separate separate logs into files, configure LevelFileName parameters.
		LevelFileName: map[int]string{
			logger.LoggerLevel("error"): yaml_conf.LogDir + "essync_error.log", // The error level log is written to the error.log file.
			logger.LoggerLevel("info"):  yaml_conf.LogDir + "essync_info.log",  // The info level log is written to the info.log file.
			logger.LoggerLevel("debug"): yaml_conf.LogDir + "essync_debug.log", // The debug level log is written to the debug.log file.
		},
		MaxSize:    1000 * 1024 * 1024,                             // File maximum (KB), default 0 is not limited
		MaxLine:    0,                                              // The maximum number of lines in the file, the default 0 is not limited
		DateSlice:  "d",                                            // Cut the document by date, support "Y" (year), "m" (month), "d" (day), "H" (hour), default "no".
		JsonFormat: false,                                          // Whether the file data is written to JSON formatting
		Format:     "%millisecond_format% [%level_string%] %body%", // JsonFormat is false, logger message written to file format string
	}
	logger.Attach("file", go_logger.LOGGER_LEVEL_DEBUG, fileConfig)

	go getData()
	go clearData()
	go SavePid()

	r.GET("/_healthy", func(c *gin.Context) {
		c.String(200, "I am very healthy")
	})
	r.GET("/exporter", func(c *gin.Context) {
		var errorNew int
		errorLogFile := yaml_conf.LogDir + "essync_error.log"
		errorNew = getLogNew(errorLogFile)
		c.String(200, "essync_error_new{} "+strconv.Itoa(errorNew))
	})
	httpPort := strconv.Itoa(yaml_conf.HttpPort)
	httpAddress := ":" + httpPort
	server := &http.Server{
		Addr:    httpAddress,
		Handler: r,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen httpPort error: " + httpPort)
			log.Fatalf("listen: %s\n error: ", err)
		}
	}()
	logger.Info("Server Start ...")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Server Shutdown ...")
}

func getData() {
	sourceField := yaml_conf.SortField
	syncCount := yaml_conf.SyncCount
	for {
		sourceClient, _ := getSourceClient()
		targetClient, _ := getTargetClient()
		/*
			var query1 lib.EsQuery
			query1 = map[string]interface{}{
				"query": map[string]interface{}{
					"range": map[string]interface{}{
						source_field: map[string]interface{}{
							"gte": 0,
						},
					},
				},
				"sort": map[string]interface{}{
					source_field: map[string]string{
						"order": "desc",
					},
				},
				"from": 0,
				"size": 1,
			}*/
		//res, err := lib.Search(target_client, yaml_conf.TargetEs.IndexName, query1)
		matchQuery := lib.MatchQuery{}
		res, err := lib.PageSort(targetClient, yaml_conf.TargetEs.IndexName, matchQuery, sourceField, "desc", 0, 1)
		if err != nil {
			logger.Error("lib.PageSort: " + err.Error())
		}
		//log.Println(res.List[0])
		sort_field_type := yaml_conf.SortFieldType
		var begin_sort interface{}
		if sort_field_type == "int64" {
			begin_sort = 0
		} else {
			begin_sort = time.Date(1970, 1, 1, 1, 1, 1, 20, time.Local)
		}

		if len(res.List) > 0 {
			//levsion需要配置calldate
			begin_sort = res.List[0].CallDate
		} else {
			logKeepDay := yaml_conf.LogKeepDay
			clearDate := time.Now().AddDate(0, 0, -logKeepDay)
			if logKeepDay > 0 {
				if sort_field_type == "int64" {
					begin_sort = clearDate.Unix()
				} else {
					begin_sort = clearDate
				}

			}
		}

		matchQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"range": map[string]interface{}{
					sourceField: map[string]interface{}{
						"gt": begin_sort,
					},
				},
			},
		}
		res_source, err := lib.PageSort(sourceClient, yaml_conf.SourceEs.IndexName, matchQuery, sourceField, "desc", 0, syncCount)
		if err != nil {
			logger.Error("lib.PageSort: " + err.Error())
		}
		//log.Println(res_source.List)
		if len(res_source.List) > 0 {
			var wg sync.WaitGroup
			for i, data := range res_source.List {
				wg.Add(1)
				docId := res_source.IdList[i]
				go func() {
					_, err = lib.Create(targetClient, yaml_conf.TargetEs.IndexName, data, docId, "_doc")
					if err != nil {
						logger.Error("lib.Create: " + err.Error())
					}
					defer wg.Done()
					//log.Println(i, res_id)
				}()

			}
			wg.Wait()
		}
		time.Sleep(time.Second * yaml_conf.SyncInterval)
	}
}

func clearData() {
	dateField := yaml_conf.DateField
	dateFieldType := yaml_conf.DateFieldType
	logKeepDay := yaml_conf.LogKeepDay
	for {
		if yaml_conf.LogKeepDay <= 0 {
			break
		}
		targetClient, err := getTargetClient()
		var dateSort interface{}
		nowTime := time.Now()
		clearDate := nowTime.AddDate(0, 0, -logKeepDay)
		if dateFieldType == "int64" {
			dateSort = clearDate.Unix()
		} else {
			dateSort = clearDate
		}
		deleteQuery := map[string]interface{}{
			"query": map[string]interface{}{
				"range": map[string]interface{}{
					dateField: map[string]interface{}{
						"lt": dateSort,
					},
				},
			},
		}
		_, err = lib.DeleteByQuery(targetClient, yaml_conf.TargetEs.IndexName, deleteQuery)
		if err != nil {
			logger.Error("DeleteByQuery: " + err.Error())
		}
		time.Sleep(time.Second * yaml_conf.ClearInterval)
	}
}

func getSourceClient() (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: yaml_conf.SourceEs.Hosts,
		Transport: &http.Transport{
			MaxIdleConns:          yaml_conf.SourceEs.HttpConfig.MaxIdleConns,                  //所有host的连接池缓存最大连接数量，默认无穷大
			MaxIdleConnsPerHost:   yaml_conf.SourceEs.HttpConfig.MaxIdleConnsPerHost,           //每个host的连接池缓存最大空闲连接数
			MaxConnsPerHost:       yaml_conf.SourceEs.HttpConfig.MaxConnsPerHost,               //对每个host的最大连接数量，0表示不限制
			IdleConnTimeout:       time.Second * yaml_conf.SourceEs.HttpConfig.IdleConnTimeout, //how long an idle connection is kept in the connection pool.
			ResponseHeaderTimeout: time.Second * yaml_conf.SourceEs.HttpConfig.ResponseHeaderTimeout,
			DialContext: (&net.Dialer{
				Timeout:   time.Second * yaml_conf.SourceEs.HttpConfig.DialTimeout, //限制建立TCP连接的时间
				KeepAlive: time.Second * yaml_conf.SourceEs.HttpConfig.DialKeepAlive,
			}).DialContext,
		},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		logger.Error("getSourceClient: " + err.Error())
		return es, err
	} else {
		//log.Println(es.Info())
		return es, nil
	}
}

func getTargetClient() (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: yaml_conf.TargetEs.Hosts,
		Username:  "elastic",
		Password:  "elastic2022",
		Transport: &http.Transport{
			MaxIdleConns:          yaml_conf.TargetEs.HttpConfig.MaxIdleConns,                  //所有host的连接池缓存最大连接数量，默认无穷大
			MaxIdleConnsPerHost:   yaml_conf.TargetEs.HttpConfig.MaxIdleConnsPerHost,           //每个host的连接池缓存最大空闲连接数
			MaxConnsPerHost:       yaml_conf.TargetEs.HttpConfig.MaxConnsPerHost,               //对每个host的最大连接数量，0表示不限制
			IdleConnTimeout:       time.Second * yaml_conf.TargetEs.HttpConfig.IdleConnTimeout, //how long an idle connection is kept in the connection pool.
			ResponseHeaderTimeout: time.Second * yaml_conf.TargetEs.HttpConfig.ResponseHeaderTimeout,
			DialContext: (&net.Dialer{
				Timeout:   time.Second * yaml_conf.TargetEs.HttpConfig.DialTimeout, //限制建立TCP连接的时间
				KeepAlive: time.Second * yaml_conf.TargetEs.HttpConfig.DialKeepAlive,
			}).DialContext,
		},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		logger.Error("getTargetClient: " + err.Error())
		return es, err
	} else {
		//log.Println(es.Info())
		return es, nil
	}
}

func getLogNew(logFile string) int {
	errorNew := 0
	var cmdOut bytes.Buffer
	var cmdStderr bytes.Buffer
	command := exec.Command("tail", "-n", "2", logFile)
	command.Stdout = &cmdOut
	command.Stderr = &cmdStderr
	err := command.Run()
	if err != nil {
		logger.Error("getLogNew: " + err.Error())
		return errorNew
	}
	outString := cmdOut.String()
	outList := strings.Split(outString, "\n")

	if len(outList) >= 2 {
		firstLine := outList[0]
		timeString := firstLine[0:19]
		lastT, _ := time.ParseInLocation("2006-01-02 15:04:05", timeString, time.Local)
		lastInt := lastT.Unix()
		nowInt := time.Now().Unix()
		if nowInt-lastInt < 300 {
			return 1
		}
	}
	return errorNew
}

func SavePid() bool {
	pidFile := yaml_conf.PidFile
	var f *os.File
	var err error
	var err1 error
	if _, err = os.Stat(pidFile); os.IsNotExist(err) {
		f, err1 = os.Create(pidFile) //创建文件
		defer f.Close()
		if err1 != nil {
			logger.Error("SavePid-create: " + err1.Error())
			return false
		}
	} else {
		f, err1 = os.OpenFile(pidFile, os.O_CREATE, 0666)
		defer f.Close()
		if err1 != nil {
			logger.Error("SavePid-open: " + err1.Error())
			return false
		}
	}
	pid := os.Getpid()
	pidString := strconv.Itoa(pid)
	if _, err = io.WriteString(f, pidString); err != nil {
		logger.Error("SavePid-write: " + err.Error())
		return false
	}
	/*
		byteString := []byte(pidString)
		if err = ioutil.WriteFile(pidFile, byteString, 0666); err != nil {
			logger.Error("SavePid: " + err.Error())
		}
	*/
	return true
}
