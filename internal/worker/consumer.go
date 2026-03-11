package worker

import (
	"context"
	"encoding/json"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"

	"trend/internal/config"
	"trend/internal/task"
	"trend/pkg/etcd"
	"trend/pkg/logger"
)

// Consumer 监听 etcd 中的待处理任务并提交给 Executor
type Consumer struct {
	client   *etcd.Client
	executor *Executor
}

// NewConsumer 初始化消费者
func NewConsumer(client *etcd.Client, executor *Executor) *Consumer {
	return &Consumer{
		client:   client,
		executor: executor,
	}
}

// Start 开始监听任务队列
func (c *Consumer) Start(ctx context.Context) {
	// 针对 Worker 的监听路径：/trend/<cluster_name>/pending/
	basePrefix := config.Conf.Worker.TaskPath
	if basePrefix == "" {
		basePrefix = "/trend"
	}
	clusterName := config.Conf.App.ClusterName

	watchPath := fmt.Sprintf("%s/%s/pending/", basePrefix, clusterName)
	logger.Info("Starting Task Consumer", logger.String("watch_path", watchPath))

	ch := c.client.Watch(ctx, watchPath, clientv3.WithPrefix())
	for resp := range ch {
		for _, ev := range resp.Events {
			if ev.Type == clientv3.EventTypePut {
				// 获取到新任务类型
				var base struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(ev.Kv.Value, &base); err != nil {
					logger.Error("Failed to unmarshal task base", logger.Err(err))
					continue
				}

				var t task.Task
				var err error

				switch base.Type {
				case "orzdba":
					t, err = task.DeserializeOrzdbaTask(ev.Kv.Value)
				default:
					err = fmt.Errorf("unknown task type: %s", base.Type)
				}

				if err != nil {
					logger.Error("Failed to instantiate specific task", logger.Err(err), logger.String("type", base.Type))
					continue
				}

				key := string(ev.Kv.Key)
				logger.Info("Received new task", logger.String("task_id", t.GetID()))

				// 提交给 Executor 并发执行
				// 可以通过并发控制或立即取回处理
				c.executor.Submit(ctx, t)

				// 任务一旦提交给自身的 Executor，即可从 pending 队列中删除
				if err := c.deleteTask(context.Background(), key); err != nil {
					logger.Error("Failed to delete pending task after submission", logger.String("key", key), logger.Err(err))
				}
			}
		}
	}
	logger.Info("Task Consumer stopped")
}

func (c *Consumer) deleteTask(ctx context.Context, key string) error {
	_, err := c.client.Cli.Delete(ctx, key)
	return err
}
