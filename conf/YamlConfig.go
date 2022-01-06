package conf

import "time"

type HttpConfig struct {
	MaxIdleConns          int           `yaml:"MaxIdleConns"`
	MaxIdleConnsPerHost   int           `yaml:"MaxIdleConnsPerHost"`
	MaxConnsPerHost       int           `yaml:"MaxConnsPerHost"`
	IdleConnTimeout       time.Duration `yaml:"IdleConnTimeout"`
	ResponseHeaderTimeout time.Duration `yaml:"ResponseHeaderTimeout"`
	DialTimeout           time.Duration `yaml:"DialTimeout"`
	DialKeepAlive         time.Duration `yaml:"DialKeepAlive"`
}

type SourceEs struct {
	Hosts      []string   `yaml:"hosts,flow"`
	User       string     `yaml:"user"`
	Password   string     `yaml:"password"`
	IndexName  string     `yaml:"indexName"`
	DocType    string     `yaml:"docType"`
	HttpConfig HttpConfig `yaml:"http_config"`
}

type TargetEs struct {
	Hosts      []string   `yaml:"hosts,flow"`
	User       string     `yaml:"user"`
	Password   string     `yaml:"password"`
	IndexName  string     `yaml:"indexName"`
	DocType    string     `yaml:"docType"`
	HttpConfig HttpConfig `yaml:"http_config"`
}

type EsConfig struct {
	SourceEs      SourceEs      `yaml:"source_es"`
	TargetEs      TargetEs      `yaml:"target_es"`
	SortField     string        `yaml:"sort_field"`
	SortFieldType string        `yaml:"sort_field_type"`
	DateField     string        `yaml:"date_field"`
	DateFieldType string        `yaml:"date_field_type"`
	SyncInterval  time.Duration `yaml:"sync_interval"`
	ClearInterval time.Duration `yaml:"clear_interval"`
	SyncCount     int           `yaml:"sync_count"`
	LogKeepDay    int           `yaml:"log_keep_day"`
	HttpPort      int           `yaml:"http_port"`
	TcpPort       int           `yaml:"tcp_port"`
	LogDir        string        `yaml:"log_dir"`
	Daemon        bool          `yaml:"daemon"`
	PidFile       string        `yaml:"pid_file"`
}
