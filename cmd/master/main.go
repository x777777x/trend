package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"trend/internal/config"
	"trend/internal/master"
	"trend/pkg/etcd"
	"trend/pkg/logger"
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

	// 3. 初始化 etcd 客户端
	etcdCli, err := etcd.NewClient(&config.Conf.Etcd)
	if err != nil {
		logger.Fatal("Failed to connect to etcd", logger.Err(err))
	}
	defer etcdCli.Close()

	// 4. 初始化 Master 组件
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

	// 优雅退出监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Master Node...")
	scheduler.Stop()
	election.Stop()
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
