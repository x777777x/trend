package config

import (
	"strings"

	"github.com/spf13/viper"

	"trend/pkg/etcd"
	"trend/pkg/logger"
	"trend/pkg/storage"
)

// Config 全局配置结构体
type Config struct {
	App    AppConfig    `mapstructure:"app"`
	Etcd   etcd.Config  `mapstructure:"etcd"`
	Logger logger.Config `mapstructure:"logger"`
	Master MasterConfig `mapstructure:"master"`
	Worker WorkerConfig `mapstructure:"worker"`
}

// AppConfig 应用基本配置
type AppConfig struct {
	ClusterName string `mapstructure:"cluster_name"`
}

// MasterConfig Master节点配置
type MasterConfig struct {
	MySQL            storage.MySQLConfig `mapstructure:"mysql"`
	APIAddr          string              `mapstructure:"api_addr"`           // HTTP API 监听地址
	CronExpr         string              `mapstructure:"cron_expr"`
	TaskPath         string              `mapstructure:"task_path"`
	BacklogThreshold int                 `mapstructure:"backlog_threshold"`
}

// WorkerConfig Worker节点配置
type WorkerConfig struct {
	MasterAPI   string `mapstructure:"master_api"`    // Master API 地址
	Concurrency int    `mapstructure:"concurrency"`   // 任务并发数
	MetricsAddr string `mapstructure:"metrics_addr"`  // Prometheus metrics HTTP 地址
}

// ClusterConfig Master API 返回给 Worker 的集群级配置
type ClusterConfig struct {
	Etcd        etcd.Config             `json:"etcd"`
	TaskPath    string                  `json:"task_path"`
	DataSources []DataSourceConfig      `json:"data_sources"`
	Storages    []StorageConfig         `json:"storages"`
}

// DataSourceConfig 数据源配置（从 API 获取）
type DataSourceConfig struct {
	Name       string                 `json:"name"`
	SourceType string                 `json:"source_type"`
	Config     map[string]interface{} `json:"config"`
}

// StorageConfig 存储配置（从 API 获取）
type StorageConfig struct {
	Name       string                 `json:"name"`
	SourceType string                 `json:"source_type"`
	Config     map[string]interface{} `json:"config"`
}

var Conf *Config

// Task type constants
const (
	TaskTypeOrzdba = "orzdba"
)

// InitConfig 初始化全局配置，从指定文件加载
func InitConfig(cfgFile string) error {
	v := viper.New()
	v.SetConfigFile(cfgFile)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvPrefix("TREND")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	Conf = new(Config)
	if err := v.Unmarshal(Conf); err != nil {
		return err
	}

	return nil
}
