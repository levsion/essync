package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"essync/conf"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"log"
)

type DocQuery map[string]interface{}
type EsQuery map[string]interface{}
type MatchQuery map[string]interface{}

type ResLists []conf.EsDoc
type SearchResponseHitsHits struct {
	Index  string          `json:"_index"`
	Type   string          `json:"_type"`
	ID     string          `json:"_id"`
	Score  float64         `json:"_score"`
	Source json.RawMessage `json:"_source"`
}
type SearchResponseHits struct {
	Total struct {
		Value    uint64 `json:"value"`
		Relation string `json:"relation"`
	} `json:"total"`
	MaxScore float64                   `json:"max_score"`
	Hits     []*SearchResponseHitsHits `json:"hits"`
}
type SearchResponse struct {
	Took     uint64 `json:"took"`
	TimedOut bool   `json:"timed_out"`
	Shards   struct {
		Total      uint64 `json:"total"`
		Successful uint64 `json:"successful"`
		Skipped    uint64 `json:"skipped"`
		Failed     uint64 `json:"failed"`
	} `json:"_shards"`
	Hits         *SearchResponseHits        `json:"hits"`
	Aggregations map[string]json.RawMessage `json:"aggregations"`
}
type MySearchResponse struct {
	Took     uint64 `json:"took"`
	TimedOut bool   `json:"timed_out"`
	Shards   struct {
		Total      uint64 `json:"total"`
		Successful uint64 `json:"successful"`
		Skipped    uint64 `json:"skipped"`
		Failed     uint64 `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total struct {
			Value    uint64 `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string     `json:"_index"`
			Type   string     `json:"_type"`
			ID     string     `json:"_id"`
			Score  float64    `json:"_score"`
			Source conf.EsDoc `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
	Aggregations map[string]json.RawMessage `json:"aggregations"`
}

type IdList []string
type resData struct {
	List   ResLists `json:"list"`
	Total  uint64   `json:"total"`
	IdList IdList   `json:"idList"`
}
type resCreate struct {
	IndexName string `json:"_index"`
	Type      string `json:"_type"`
	Id        string `json:"_id"`
	Result    string `json:"result"`
	Shards    struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	}
}
type resInfo struct {
	IndexName string          `json:"_index"`
	Type      string          `json:"_type"`
	Id        string          `json:"_id"`
	Found     bool            `json:"found"`
	Source    json.RawMessage `json:"_source"`
}

var MatchQueryDemo = map[string]interface{}{
	"query": map[string]interface{}{
		"match": map[string]interface{}{},
	},
}
var EsDocDemo = `{
	"parameter":"{\"businessMode\":\"GXZF_BM_ZWZS_LGQJ\",\"currentPage\":1,\"handler\":\"u8m44hn4xefqav8k11e6g2\",\"pageSize\":1,\"taskType\":0}",
	"result":"{\"code\":0,\"data\":\"PageVO(totalPage=0, total=0, currentPage=1, pageSize=1, list=[])\",\"msg\":\"执行成功\"}",
	"appId":"1414434062843641856",
	"appName":"统一门户",
	"callDate":1630688809937,
	"interfaceName":"查询用户单据",
	"interfaceUrl":"//task/external/user/query",
	"requestMethod":"GET",
	"status":1
}
_id:xEWgrHsBnlCsnW5ak8_W 
_index:interface_call_log_qa 
_score:0 
_type:log
`
var EsQueryDemo = `{
	"query": {
		"match": {
			"id": "1",
		},
	},
}`
var EsQueryDemo2 = `{
	"query": {
		"range": {
			"callDate": {
				"gte": 0
			}
		}
	},
	"sort": {
		"callDate": {
			"order": "desc"
		}
	},
	"from": 0,
	"size": 1
}`

func Search(es *elasticsearch.Client, indexName string, query EsQuery) (resData, error) {
	resTmp := resData{}
	// search
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return resTmp, err
	}
	// Perform the search request.
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(indexName),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		return resTmp, err
	}
	defer res.Body.Close()
	var lists ResLists
	var idList IdList
	total, idList, err := DecodeSearch(res, &lists)
	if err != nil {
		return resTmp, err
	}
	return resData{
		List:   lists,
		Total:  total,
		IdList: idList,
	}, nil
}

func PageSort(es *elasticsearch.Client, indexName string, matchQuery MatchQuery, sortField string, sortType string, from int, size int) (resData, error) {
	resTmp := resData{}
	// search
	var buf bytes.Buffer
	var query = matchQuery
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return resTmp, err
	}
	// Perform the search request.
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(indexName),
		es.Search.WithBody(&buf),
		es.Search.WithSort(sortField+":"+sortType),
		es.Search.WithFrom(from),
		es.Search.WithSize(size),
		es.Search.WithTrackTotalHits(true), //默认只能查询10000条
		es.Search.WithPretty(),
	)
	if err != nil {
		return resTmp, err
	}

	/*var rp MySearchResponse
	err1 := json.NewDecoder(res.Body).Decode(&rp)
	if err1 != nil {
		fmt.Println(err1.Error())
		return resTmp, err
	}
	fmt.Println(rp.Hits.Hits[0].Source)
	*/
	defer res.Body.Close()
	var lists ResLists
	var idList IdList
	total, idList, err := DecodeSearch(res, &lists)
	if err != nil {
		return resTmp, err
	}
	return resData{
		List:   lists,
		Total:  total,
		IdList: idList,
	}, nil
}

func Create(es *elasticsearch.Client, indexName string, doc conf.EsDoc, docId string, docType string) (string, error) {
	docType = "_doc"
	// Create creates a new document in the index.
	// Returns a 409 response when a document with a same ID already exists in the index.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		return "", err
	}
	res, err := es.Create(indexName, docId, &buf, es.Create.WithDocumentType(docType))
	//fmt.Println(res)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var r resCreate
	err1 := json.NewDecoder(res.Body).Decode(&r)
	if err1 != nil {
		return "id", err1
	}
	return r.Id, nil
}

func DeleteByQuery(es *elasticsearch.Client, indexName string, query EsQuery) (*esapi.Response, error) {
	// DeleteByQuery deletes documents matching the provided query
	resTmp := &esapi.Response{}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return resTmp, err
	}
	index := []string{indexName}
	res, err := es.DeleteByQuery(index, &buf)
	if err != nil {
		return resTmp, err
	}
	defer res.Body.Close()
	/*var r resData
	err1 := json.NewDecoder(res.Body).Decode(&r)
	if err1 != nil {
		return resTmp, err1
	}*/
	return res, nil
}

func Delete(es *elasticsearch.Client, indexName string, id string) (resData, error) {
	resTmp := resData{}
	res, err := es.Delete(indexName, id)
	if err != nil {
		return resTmp, err
	}
	defer res.Body.Close()
	return resTmp, nil
}

func Get(es *elasticsearch.Client, indexName string, id string) (resInfo, error) {
	resTmp := resInfo{}
	res, err := es.Get(indexName, id)
	if err != nil {
		return resTmp, err
	}
	defer res.Body.Close()
	var rp resInfo
	log.Println(res.Body)
	err1 := json.NewDecoder(res.Body).Decode(&rp)
	if err1 != nil {
		return resTmp, err
	}
	return rp, nil
}

func Update(es *elasticsearch.Client) (resData, error) {
	resTmp := resData{}
	// Update updates a document with a script or partial document.
	var buf bytes.Buffer
	doc := map[string]interface{}{
		"doc": map[string]interface{}{
			"title":   "更新你看到外面的世界是什么样的？",
			"content": "更新外面的世界真的很精彩",
		},
	}
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		return resTmp, err
	}
	res, err := es.Update("demo", "esd", &buf, es.Update.WithDocumentType("doc"))
	if err != nil {
		return resTmp, err
	}
	defer res.Body.Close()
	return resTmp, nil
}

func UpdateByQuery(es *elasticsearch.Client) (resData, error) {
	resTmp := resData{}
	// UpdateByQuery performs an update on every document in the index without changing the source,
	// for example to pick up a mapping change.
	index := []string{"demo"}
	var buf bytes.Buffer
	doc := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"title": "外面",
			},
		},
		// 根据搜索条件更新title、content
		"script": map[string]interface{}{
			"source": "ctx._source.title=params.title;ctx._source.content=params.content;",
			"params": map[string]interface{}{
				"title":   "看看外面的世界真的很精彩",
				"content": "他们和你看到外面的世界是什么样的？",
			},
			"lang": "painless",
		},
	}
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		return resTmp, err
	}
	res, err := es.UpdateByQuery(
		index,
		es.UpdateByQuery.WithDocumentType("doc"),
		es.UpdateByQuery.WithBody(&buf),
		es.UpdateByQuery.WithContext(context.Background()),
		es.UpdateByQuery.WithPretty(),
	)
	if err != nil {
		return resTmp, err
	}
	defer res.Body.Close()
	return resTmp, nil
}

func DecodeSearch(resp *esapi.Response, v interface{}) (uint64, []string, error) {
	var listTmp IdList
	if resp.StatusCode == 404 {
		return 0, listTmp, nil
	}
	if resp.StatusCode != 200 {
		return 0, listTmp, errors.New("1")
	}
	var r SearchResponse
	//var r MySearchResponse
	err := json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return 0, listTmp, err
	}
	ids := make([]string, 0, len(r.Hits.Hits))
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, doc := range r.Hits.Hits {
		ids = append(ids, doc.ID)
		buf.Write(doc.Source)
		if i != len(r.Hits.Hits)-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte(']')
	err = json.NewDecoder(&buf).Decode(v)
	if err != nil {
		return 0, listTmp, err
	}
	return r.Hits.Total.Value, ids, nil
}
