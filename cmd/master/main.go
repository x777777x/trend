package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trend/internal/config"
	"trend/internal/master"
	"trend/internal/models"
	"trend/pkg/etcd"
	"trend/pkg/logger"
	"trend/pkg/metrics"
	"trend/pkg/storage"
)

const (
	DefaultBacklogThreshold = 100
)

var cfgFile = flag.String("config", "configs/config.yaml", "Path to config file")

func main() {
	flag.Parse()

	// 1. 初始化配置
	if err := config.InitConfig(*cfgFile); err != nil {
		panic("Failed to init config: " + err.Error())
	}

	// 2. 初始化日志
	if err := logger.Init(&config.Conf.Logger); err != nil {
		panic("Failed to init logger: " + err.Error())
	}
	logger.Info("Starting Master Node...")

	// 3. 注册 Prometheus 指标
	metrics.RegisterAll()

	// 4. 初始化 etcd 客户端
	etcdCli, err := etcd.NewClient(&config.Conf.Etcd)
	if err != nil {
		logger.Fatal("Failed to connect to etcd", logger.Err(err))
	}
	defer etcdCli.Close()

	// 4. 初始化 Master 端 MySQL（读取任务配置用）
	if err := storage.InitMasterMySQL(&config.Conf.Master.MySQL); err != nil {
		logger.Fatal("Failed to connect to MySQL", logger.Err(err))
	}

	// 4.1 检查各任务类型实例表是否已创建
	if err := storage.EnsureTaskCalcInstanceTables(); err != nil {
		logger.Fatal("Failed to ensure task instance tables", logger.Err(err))
	}

	// 4.2 初始化 ResultsDB 只读连接（查询趋势分位值结果）
	if err := initResultsDB(); err != nil {
		logger.Fatal("Failed to init results database", logger.Err(err))
	}

	// 5. 初始化 Master 组件
	// a. Dispatcher (负责将任务下发到队列，含背压)
	threshold := config.Conf.Master.BacklogThreshold
	if threshold <= 0 {
		threshold = DefaultBacklogThreshold // 默认值
	}
	dispatcher := master.NewDispatcher(etcdCli, threshold)

	// b. Leader Election
	nodeID := config.Conf.App.ClusterName + "_" + getLocalIP()
	election := master.NewLeaderElection(etcdCli, nodeID, config.Conf.App.ClusterName)

	// c. Scheduler (调度器依赖 Dispatcher 和 Election)
	scheduler, err := master.NewScheduler(dispatcher, election)
	if err != nil {
		logger.Fatal("Failed to init scheduler", logger.Err(err))
	}

	// 启动核心服务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动 Master API 服务
	apiServer := master.NewAPIServer(config.Conf.Master.APIAddr)
	apiServer.Start()

	// 启动 Leader 选举参与
	if err := election.Start(ctx); err != nil {
		logger.Fatal("Failed to start leader election", logger.Err(err))
	}

	// 启动负载监控
	dispatcher.MonitorWorkers(ctx)

	// 启动调度器
	if err := scheduler.Start(); err != nil {
		logger.Fatal("Failed to start scheduler", logger.Err(err))
	}

	// 启动超时任务采集器
	staleCollector := master.NewStaleCollector(5 * time.Minute)
	staleCollector.Start(ctx)

	// 优雅退出监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Master Node...")
	staleCollector.Stop()
	scheduler.Stop()
	election.Stop()
	apiServer.Stop()
	logger.Info("Master Node shutdown complete.")
}

// getLocalIP 获取本地非回环 IPv4 地址
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// initResultsDB 从 trend_storage_config 查询 MySQL 类型的存储配置并初始化只读连接
func initResultsDB() error {
	db := storage.GetDB()
	if db == nil {
		return fmt.Errorf("master database not initialized")
	}

	var cfg models.TrendStorageConfig
	err := db.Where("source_type = ? AND status = ?", "mysql", 1).
		Order("id ASC").First(&cfg).Error
	if err != nil {
		return fmt.Errorf("no MySQL storage config found: %w", err)
	}

	mysqlCfg, err := parseMySQLConfig(cfg.Config)
	if err != nil {
		return fmt.Errorf("invalid MySQL config: %w", err)
	}

	return storage.InitResultsMySQL(mysqlCfg)
}

func parseMySQLConfig(configMap models.JSONMap) (*storage.MySQLConfig, error) {
	cfg := &storage.MySQLConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 1,
	}

	if v, ok := configMap["user"].(string); ok {
		cfg.User = v
	}
	if v, ok := configMap["password"].(string); ok {
		cfg.Password = v
	}
	if v, ok := configMap["host"].(string); ok {
		cfg.Host = v
	}
	if v, ok := configMap["port"].(float64); ok {
		cfg.Port = int(v)
	}
	if v, ok := configMap["dbname"].(string); ok {
		cfg.DBName = v
	}

	if cfg.User == "" || cfg.Host == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("missing required MySQL config fields (user, host, dbname)")
	}
	if cfg.Port == 0 {
		cfg.Port = 3306
	}

	return cfg, nil
}
