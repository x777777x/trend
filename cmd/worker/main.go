package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trend/internal/config"
	"trend/internal/worker"
	"trend/pkg/etcd"
	"trend/pkg/logger"
	"trend/pkg/metrics"
)

var cfgFile = flag.String("config", "configs/config.yaml", "Path to config file")

func main() {
	flag.Parse()

	// 1. 初始化本地配置（仅 cluster_name + master_api + concurrency + logger）
	if err := config.InitConfig(*cfgFile); err != nil {
		panic("Failed to init config: " + err.Error())
	}

	// 2. 初始化日志
	if err := logger.Init(&config.Conf.Logger); err != nil {
		panic("Failed to init logger: " + err.Error())
	}
	logger.Info("Starting Worker Node...")

	// 3. 注册 Prometheus 指标
	metrics.RegisterAll()

	// 4. 从 Master API 拉取集群级配置（带重试）
	clusterName := config.Conf.App.ClusterName
	masterAPI := config.Conf.Worker.MasterAPI

	logger.Info("Fetching cluster config from Master",
		logger.String("cluster", clusterName),
		logger.String("master_api", masterAPI))

	clusterCfg, err := worker.RetryFetchConfig(masterAPI, clusterName, 3, 5*time.Second)
	if err != nil {
		logger.Fatal("Failed to fetch cluster config after retries", logger.Err(err))
	}

	// 4. 根据集群配置初始化数据源和存储客户端
	if err := worker.InitClientsFromConfig(clusterCfg); err != nil {
		logger.Fatal("Failed to init clients from cluster config", logger.Err(err))
	}

	// 5. 初始化 etcd 客户端（从集群配置获取）
	etcdCli, err := etcd.NewClient(&clusterCfg.Etcd)
	if err != nil {
		logger.Fatal("Failed to connect to etcd", logger.Err(err))
	}
	defer etcdCli.Close()

	// 6. 初始化 Worker 组件
	// a. Executor (并发限制)
	executor := worker.NewExecutor()

	// a.1 注册任务失败回调，递增失败计数
	executor.SetOnFailure(func(taskType, clusterName string) {
		metrics.WorkerTaskFailuresTotal.WithLabelValues(clusterName, taskType).Inc()
	})

	// b. Consumer (监听 etcd 任务推送)
	consumer := worker.NewConsumer(etcdCli, executor, clusterCfg.TaskPath, clusterName)

	// 7. 后台启动消费者
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)

	// 标记 worker 存活
	metrics.WorkerUp.Set(1)

	// 启动 Metrics HTTP 服务
	metricsAddr := config.Conf.Worker.MetricsAddr
	if metricsAddr == "" {
		metricsAddr = ":9090"
	}
	go func() {
		logger.Info("Starting metrics server", logger.String("addr", metricsAddr))
		if err := http.ListenAndServe(metricsAddr, metrics.Handler()); err != nil {
			logger.Error("Metrics server failed", logger.Err(err))
		}
	}()

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Worker Node...")
	cancel()                 // 通知消费者退出
	consumer.Wait()          // 等待消费者 goroutine 完全退出
	executor.WaitForCompletion() // 等待所有执行中的任务完成
	logger.Info("Worker Node shutdown complete.")
}
