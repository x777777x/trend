package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"trend/internal/config"
	"trend/internal/task"
	"trend/pkg/etcd"
	"trend/pkg/logger"
	"trend/pkg/storage"
)

// Consumer 监听 etcd 中的待处理任务并提交给 Executor
type Consumer struct {
	client      *etcd.Client
	executor    *Executor
	taskPath    string
	clusterName string
	done        chan struct{}
}

// NewConsumer 初始化消费者
func NewConsumer(client *etcd.Client, executor *Executor, taskPath, clusterName string) *Consumer {
	return &Consumer{
		client:      client,
		executor:    executor,
		taskPath:    taskPath,
		clusterName: clusterName,
		done:        make(chan struct{}),
	}
}

// Start 开始监听任务队列
func (c *Consumer) Start(ctx context.Context) {
	defer close(c.done)
	basePrefix := c.taskPath
	if basePrefix == "" {
		basePrefix = "/trend"
	}

	watchPath := fmt.Sprintf("%s/%s/pending/", basePrefix, c.clusterName)
	logger.Info("Starting Task Consumer", logger.String("watch_path", watchPath))

	ch := c.client.Watch(ctx, watchPath, clientv3.WithPrefix())
	for resp := range ch {
		for _, ev := range resp.Events {
			if ev.Type == clientv3.EventTypePut {
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
				case config.TaskTypeOrzdba:
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

				// 传入完成回调，任务执行完成后删除 etcd 任务并更新水位线
				c.executor.Submit(ctx, t, func(err error) {
					if delErr := c.deleteTask(ctx, key); delErr != nil {
						logger.Error("Failed to delete pending task after execution", logger.String("key", key), logger.Err(delErr))
					}
					if err == nil {
						if updateErr := c.updateLastTime(t); updateErr != nil {
							logger.Error("Failed to update last_time after task completion",
								logger.String("task_id", t.GetID()), logger.Err(updateErr))
						}
					}
				})
			}
		}
	}
	logger.Info("Task Consumer stopped")
}

func (c *Consumer) deleteTask(ctx context.Context, key string) error {
	_, err := c.client.Cli.Delete(ctx, key)
	return err
}

// updateLastTime 任务完成后将窗口终点写回数据库，推进下一次查询的水位线
func (c *Consumer) updateLastTime(t task.Task) error {
	db := storage.GetWorkerDB()
	if db == nil {
		return fmt.Errorf("worker database not initialized")
	}

	// 根据任务类型动态路由到对应的 calc_instance 表
	tableName := fmt.Sprintf("trend_%s_calc_instance", t.GetType())

	// 任务 Run() 中 endTime = LastTime + 1 min，这是实际处理到的窗口终点
	endTime := t.GetLastTime().Add(1 * time.Minute)

	result := db.Table(tableName).
		Where("cluster_name = ? AND instance_name = ?", t.GetClusterName(), t.GetHost()).
		Update("last_time", endTime)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		logger.Debug("No calc_instance row found to update last_time",
			logger.String("table", tableName),
			logger.String("cluster", t.GetClusterName()),
			logger.String("host", t.GetHost()))
	}
	return nil
}

// Wait 等待 Consumer goroutine 退出
func (c *Consumer) Wait() {
	<-c.done
}
