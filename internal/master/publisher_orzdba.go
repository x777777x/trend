package master

import (
	"context"
	"fmt"
	"time"

	"trend/internal/models"
	"trend/internal/task"
	"trend/pkg/logger"
	"trend/pkg/storage"
)

// OrzdbaPublisher 实现了针对 orzdba 类型任务的抽象
type OrzdbaPublisher struct {
	ClusterName   string
	SlideInterval uint
	tasks         []*task.OrzdbaTask
}

func (p *OrzdbaPublisher) Initialize(slideInterval uint) error {
	db := storage.GetDB()
	if db == nil {
		logger.Error("Database not initialized, cannot fetch orzdba tasks", logger.String("cluster_name", p.ClusterName))
		return fmt.Errorf("database not initialized")
	}

	var instances []models.TrendOrzdbaCalcInstance
	// 查询属于该集群且状态为启用的具体需要执行的子任务实例
	err := db.Where("cluster_name = ? AND status = ?", p.ClusterName, 1).Find(&instances).Error
	if err != nil {
		logger.Error("Failed to fetch orzdba instances from DB", logger.String("cluster_name", p.ClusterName), logger.Err(err))
		return err
	}

	p.tasks = make([]*task.OrzdbaTask, 0, len(instances))
	for _, inst := range instances {
		p.tasks = append(p.tasks, &task.OrzdbaTask{
			ID:            fmt.Sprintf("orzdba-task-%s-%s-%d", p.ClusterName, inst.InstanceName, time.Now().UnixNano()),
			ClusterName:   p.ClusterName,
			Host:          inst.InstanceName,
			LastTime:      inst.LastTime,
			SlideInterval: slideInterval,
			Type:          "orzdba",
			CreatedAt:     time.Now(),
			CalcInstanceID: inst.ID,
		})
	}

	logger.Info("OrzdbaPublisher initialized tasks", logger.String("cluster_name", p.ClusterName), logger.Int("task_count", len(p.tasks)))
	return nil
}

func (p *OrzdbaPublisher) Publish(ctx context.Context, dispatcher *Dispatcher) error {
	for _, t := range p.tasks {
		if err := dispatcher.Dispatch(ctx, t); err != nil {
			logger.Error("OrzdbaPublisher failed to dispatch task", logger.String("task_id", t.ID), logger.Err(err))
			// 可以选择 continue 或者直接 return err，这里选择 continue 尝试发送完
		} else {
			logger.Debug("OrzdbaPublisher successfully dispatched task", logger.String("task_id", t.ID))
		}
	}
	return nil
}
