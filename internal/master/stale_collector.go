package master

import (
	"context"
	"fmt"
	"time"

	"trend/internal/config"
	"trend/internal/models"
	"trend/pkg/logger"
	"trend/pkg/metrics"
	"trend/pkg/storage"
)

// StaleCollector 周期性查询 calc_instance 表，统计超时的任务实例
type StaleCollector struct {
	interval time.Duration
}

// NewStaleCollector 创建采集器
func NewStaleCollector(interval time.Duration) *StaleCollector {
	return &StaleCollector{interval: interval}
}

// Start 在后台周期采集，返回的 Stop 函数用于退出
func (c *StaleCollector) Start(ctx context.Context) {
	// 启动时标记 master 存活
	metrics.MasterUp.Set(1)

	go c.run(ctx)
}

// Stop 停止采集（通过取消 ctx 实现）
func (c *StaleCollector) Stop() {}

func (c *StaleCollector) run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

func (c *StaleCollector) collect() {
	db := storage.GetDB()
	if db == nil {
		logger.Error("StaleCollector: database connection is nil")
		return
	}

	clusterName := config.Conf.App.ClusterName

	// 查询当前集群启用的任务类型
	var tasksConfig []models.TrendClusterTask
	if err := db.Where("cluster_name = ? AND status = 1", clusterName).Find(&tasksConfig).Error; err != nil {
		logger.Error("StaleCollector: failed to query cluster tasks", logger.Err(err))
		return
	}

	for _, tc := range tasksConfig {
		if tc.StaleThresholdMinutes <= 0 {
			continue
		}

		tableName := fmt.Sprintf("trend_%s_calc_instance", tc.TaskName)

		var count int64
		err := db.Table(tableName).
			Where("status = ? AND last_time < DATE_SUB(NOW(), INTERVAL ? MINUTE)", 1, tc.StaleThresholdMinutes).
			Count(&count).Error

		if err != nil {
			logger.Error("StaleCollector: failed to count stale instances",
				logger.String("task_type", tc.TaskName),
				logger.String("table", tableName),
				logger.Err(err))
			continue
		}

		// 无论是否有超时任务都设置 gauge，防止无超时任务的 label 组合在 Prometheus 中残留
		metrics.MasterStaleTasks.WithLabelValues(clusterName, tc.TaskName).Set(float64(count))
	}
}
