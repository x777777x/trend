package master

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"trend/internal/config"
	"trend/internal/task"
	"trend/pkg/etcd"
	"trend/pkg/logger"
)

// Dispatcher 负责将任务投递给 Worker，并对整体负载进行监控（背压）
type Dispatcher struct {
	client    *etcd.Client
	threshold int // 全局任务阈值，超过则暂停分发
}

// NewDispatcher 初始化分发器
func NewDispatcher(client *etcd.Client, threshold int) *Dispatcher {
	return &Dispatcher{
		client:    client,
		threshold: threshold, // 如果任务堆积超过此数量，则触发背压
	}
}

// Dispatch 触发一次任务分发
func (d *Dispatcher) Dispatch(ctx context.Context, t task.Task) error {
	basePath := config.Conf.Master.TaskPath

	taskType := t.GetType()
	clusterName := t.GetClusterName()
	taskID := t.GetID()

	// 生成任务特定的前缀路径用于背压检查
	prefixPath := etcd.GenerateTaskPrefixPath(basePath, clusterName, taskType)

	// 背压检查 (只查当前类型)
	pendingCount, err := d.getPendingTaskCount(ctx, prefixPath)
	if err != nil {
		return fmt.Errorf("failed to get pending task count: %w", err)
	}

	if pendingCount >= d.threshold {
		logger.Warn("Backpressure triggered: too many pending tasks",
			logger.Int("pending", pendingCount),
			logger.Int("threshold", d.threshold),
			logger.String("task_type", taskType),
			logger.String("cluster", clusterName),
		)
		return fmt.Errorf("backpressure activated for %s, pending tasks count: %d", taskType, pendingCount)
	}

	taskBytes, err := t.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	// 将任务推入 etcd
	key := etcd.GenerateTaskPath(basePath, clusterName, taskType, taskID)
	_, err = d.client.Put(ctx, key, string(taskBytes))
	if err != nil {
		return fmt.Errorf("failed to push task to etcd: %w", err)
	}

	logger.Info("Task dispatched successfully", logger.String("task_id", taskID), logger.String("path", key))
	return nil
}

// getPendingTaskCount 获取指定前缀下的待处理任务数
func (d *Dispatcher) getPendingTaskCount(ctx context.Context, prefixPath string) (int, error) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithCountOnly(),
	}
	resp, err := d.client.Get(ctx, prefixPath, opts...)
	if err != nil {
		return 0, err
	}
	return int(resp.Count), nil
}

// MonitorWorkers 实时监听当前系统中运行中的任务数，动态调整分发策略。为简化实现，这里通过 getPendingTaskCount 替代了单独的 Worker 心跳负载采集监控。
func (d *Dispatcher) MonitorWorkers(ctx context.Context) {
	// 根据具体需求，这里可以注册 Watch 机制来订阅 Worker 的执行状态上报数据，
	// 动态计算集群的计算容量并调整 d.threshold。
	logger.Info("Worker Load Monitoring started", logger.Int("threshold", d.threshold))
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 监控全局 load，使用基础路径
	basePrefix := config.Conf.Master.TaskPath
	if basePrefix == "" {
		basePrefix = "/trend/"
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Worker Load Monitoring stopped")
				return
			case <-ticker.C:
				count, err := d.getPendingTaskCount(ctx, basePrefix)
				if err != nil {
					logger.Error("Failed to fetch pending tasks during monitor", logger.Err(err))
				} else {
					logger.Info("Current cluster load", logger.Int("total_pending_tasks", count))
				}
			}
		}
	}()
}
