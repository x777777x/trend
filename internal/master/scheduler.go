package master

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-co-op/gocron/v2"

	"trend/internal/config"
	"trend/internal/models"
	"trend/pkg/logger"
	"trend/pkg/storage"
)

// Scheduler 管理 Master 节点的定时任务调度
type Scheduler struct {
	scheduler  gocron.Scheduler
	job        gocron.Job
	dispatcher *Dispatcher
	election   *LeaderElection
	mutex      sync.Mutex
	isRunning  bool
}

// NewScheduler 初始化 Scheduler
func NewScheduler(dispatcher *Dispatcher, election *LeaderElection) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create gocron scheduler: %w", err)
	}

	return &Scheduler{
		scheduler:  s,
		dispatcher: dispatcher,
		election:   election,
	}, nil
}

// Start 开始调度任务
func (s *Scheduler) Start() error {
	s.mutex.Lock()
	defer s.mutex.Lock()
	if s.isRunning {
		return nil
	}

	cronExpr := config.Conf.Master.CronExpr
	job, err := s.scheduler.NewJob(
		gocron.CronJob(cronExpr, false),
		gocron.NewTask(s.executeTask),
	)
	if err != nil {
		return fmt.Errorf("failed to create cron job: %w", err)
	}
	s.job = job

	s.scheduler.Start()
	s.isRunning = true
	logger.Info("Scheduler started", logger.String("cron", cronExpr))
	return nil
}

// Stop 停止调度
func (s *Scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.isRunning {
		err := s.scheduler.Shutdown()
		if err != nil {
			logger.Error("Failed to shutdown scheduler", logger.Err(err))
		}
		s.isRunning = false
		logger.Info("Scheduler stopped")
	}
}

// executeTask 定时任务执行的逻辑
func (s *Scheduler) executeTask() {
	// 只有 Leader 才负责生成和分发任务
	if !s.election.IsLeader() {
		return
	}

	clusterName := config.Conf.App.ClusterName
	logger.Info("Executing scheduled tasks as Leader", logger.String("cluster_name", clusterName))

	db := storage.GetDB()
	if db == nil {
		logger.Error("Database connection is nil, skipping task execution")
		return
	}

	ctx := context.Background()
	var tasksConfig []models.TrendClusterTask

	// 查询该集群启用的任务类型配置
	err := db.Where("cluster_name = ? AND status = 1", clusterName).Find(&tasksConfig).Error
	if err != nil {
		logger.Error("Failed to query trend_cluster_task", logger.String("cluster_name", clusterName), logger.Err(err))
		return
	}

	if len(tasksConfig) == 0 {
		logger.Debug("No active task config found for cluster", logger.String("cluster_name", clusterName))
		return
	}

	for _, tc := range tasksConfig {
		publisher, err := NewTaskPublisher(tc.TaskName, clusterName)
		if err != nil {
			logger.Error("Failed to create task publisher", logger.String("task_name", tc.TaskName), logger.Err(err))
			continue
		}

		if err := publisher.Initialize(); err != nil {
			logger.Error("Failed to initialize task publisher", logger.String("task_name", tc.TaskName), logger.Err(err))
			continue
		}

		// 发布并分发执行
		if err := publisher.Publish(ctx, s.dispatcher); err != nil {
			logger.Error("Failed to publish tasks", logger.String("task_name", tc.TaskName), logger.Err(err))
		}
	}
}
