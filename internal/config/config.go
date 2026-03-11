package config

import (
	"strings"

	"github.com/spf13/viper"

	"trend/pkg/etcd"
	"trend/pkg/logger"
	"trend/pkg/storage"
)

// GlobalConfig 全局配置结构体
type GlobalConfig struct {
	App    AppConfig           `mapstructure:"app"`
	Logger logger.Config       `mapstructure:"logger"`
	Etcd   etcd.Config         `mapstructure:"etcd"`
	MySQL  storage.MySQLConfig `mapstructure:"mysql"`
	ES     storage.ESConfig    `mapstructure:"elasticsearch"`
	Master MasterConfig        `mapstructure:"master"`
	Worker WorkerConfig        `mapstructure:"worker"`
}

// AppConfig 应用基本配置
type AppConfig struct {
	Mode        string `mapstructure:"mode"` // master or worker
	ClusterName string `mapstructure:"cluster_name"`
}

// MasterConfig Master节点特定配置
type MasterConfig struct {
	CronExpr         string `mapstructure:"cron_expr"`         // 任务调度 Cron 表达式
	TaskPath         string `mapstructure:"task_path"`         // Etcd 任务写入路径前缀
	BacklogThreshold int    `mapstructure:"backlog_threshold"` // 调度器背压阈值
}

// WorkerConfig Worker节点特定配置
type WorkerConfig struct {
	Concurrency int    `mapstructure:"concurrency"` // 任务并发数 (Semaphore 大小)
	TaskPath    string `mapstructure:"task_path"`   // 监听的 Etcd 任务路径前缀
}

var Conf *GlobalConfig

// InitConfig 初始化全局配置，从指定文件加载
func InitConfig(cfgFile string) error {
	v := viper.New()
	v.SetConfigFile(cfgFile)
	v.SetConfigType("yaml")
	// 支持环境变量替换，前缀为 TREND_
	v.AutomaticEnv()
	v.SetEnvPrefix("TREND")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	Conf = new(GlobalConfig)
	if err := v.Unmarshal(Conf); err != nil {
		return err
	}

	return nil
}
