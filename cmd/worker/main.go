package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"trend/internal/config"
	"trend/internal/worker"
	"trend/pkg/etcd"
	"trend/pkg/logger"
	"trend/pkg/storage"
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
	logger.Info("Starting Worker Node...")

	// 3. 初始化外部存储和配置 (etcd, mysql, elasticsearch)
	etcdCli, err := etcd.NewClient(&config.Conf.Etcd)
	if err != nil {
		logger.Fatal("Failed to connect to etcd", logger.Err(err))
	}
	defer etcdCli.Close()

	if err := storage.InitMySQL(&config.Conf.MySQL); err != nil {
		logger.Fatal("Failed to connect to MySQL", logger.Err(err))
	}
	if err := storage.InitES(&config.Conf.ES); err != nil {
		logger.Fatal("Failed to connect to Elasticsearch", logger.Err(err))
	}

	// 4. 初始化 Worker 组件
	// a. DataFetcher
	fetcher := worker.NewDataFetcher()

	// b. Executor (并发限制)
	executor := worker.NewExecutor(fetcher)

	// c. Consumer (监听推送)
	consumer := worker.NewConsumer(etcdCli, executor)

	// 后台启动消费者
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Worker Node...")
	// 阻止新的消费，并等待队列中已取出的任务执行完
	executor.WaitForCompletion()
	logger.Info("Worker Node shutdown complete.")
}
