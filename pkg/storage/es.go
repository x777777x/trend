package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
)

// ESConfig 定义 Elasticsearch 配置
type ESConfig struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

// ESClient 封装 elasticsearch 客户端
var ESClient *elasticsearch.Client

// tsFormat 存储 ES 中 timestamp 字段的格式（秒/毫秒），-1 表示未检测
var (
	tsFormat int64 // 1=秒, 1000=毫秒
	tsMu     sync.Once
)

// InitES 初始化 Elasticsearch 客户端
func InitES(cfg *ESConfig) error {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return err
	}

	// 测试连接
	if _, err := es.Info(); err != nil {
		return err
	}

	ESClient = es
	return nil
}

// GetES 获取 Elasticsearch 客户端实例
func GetES() *elasticsearch.Client {
	return ESClient
}

// DetectTimestampFormat 检测 ES 中 timestamp 字段的单位（秒或毫秒）
// 返回 1 表示秒，1000 表示毫秒
func DetectTimestampFormat() (int64, error) {
	if ESClient == nil {
		return 0, fmt.Errorf("elasticsearch client not initialized")
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 1,
		"sort": []interface{}{
			map[string]interface{}{"timestamp": "desc"},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, err
	}

	res, err := ESClient.Search(
		ESClient.Search.WithIndex("metrics-*", "trend-orzdba-*"),
		ESClient.Search.WithBody(&buf),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to query ES for timestamp detection: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("ES search error: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return 0, err
	}

	hits, _ := result["hits"].(map[string]interface{})
	hitsList, _ := hits["hits"].([]interface{})
	if len(hitsList) == 0 {
		return 0, fmt.Errorf("no documents found in ES to detect timestamp format")
	}

	hit, _ := hitsList[0].(map[string]interface{})
	source, _ := hit["_source"].(map[string]interface{})
	tsRaw, ok := source["timestamp"]
	if !ok {
		return 0, fmt.Errorf("timestamp field not found in ES document")
	}

	var ts float64
	switch v := tsRaw.(type) {
	case float64:
		ts = v
	case string:
		// ISO 8601 等字符串格式，按毫秒处理（ES 通常用 epoch_millis）
		return 1000, nil
	default:
		return 0, fmt.Errorf("unexpected timestamp type: %T", tsRaw)
	}

	if ts > 1e12 {
		return 1000, nil // 毫秒
	}
	return 1, nil // 秒
}

// GetTimestampFormat 获取并缓存 timestamp 格式（线程安全，仅检测一次）
func GetTimestampFormat() (int64, error) {
	var err error
	tsMu.Do(func() {
		tsFormat, err = DetectTimestampFormat()
	})
	if err != nil {
		// 检测失败时默认使用秒
		return 1, nil
	}
	return tsFormat, nil
}
