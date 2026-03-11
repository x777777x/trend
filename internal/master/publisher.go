package master

import (
	"context"
	"fmt"
)

// TaskPublisher 定义了各类型任务在 Master 端的发布接口
type TaskPublisher interface {
	// Initialize 做一些初始化准备
	Initialize(slideInterval uint) error
	// Publish 将获取到的任务清单发布/分发下去
	Publish(ctx context.Context, dispatcher *Dispatcher) error
}

// NewTaskPublisher 根据数据库配置的任务名称及集群名实例化具体的 Publisher
func NewTaskPublisher(taskName string, clusterName string) (TaskPublisher, error) {
	switch taskName {
	case "orzdba":
		return &OrzdbaPublisher{
			ClusterName: clusterName,
		}, nil
	// 未来可在此补充针对其他类型任务的 publisher 实例化逻辑
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskName)
	}
}
