package storage

import (
	"github.com/elastic/go-elasticsearch/v8"
)

// ESConfig 定义 Elasticsearch 配置
type ESConfig struct {
	Addresses []string `yaml:"addresses"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
}

// ESClient 封装 elasticsearch 客户端
var ESClient *elasticsearch.Client

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
