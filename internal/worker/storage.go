package worker

import (
	"context"
	"fmt"
	"hash/crc32"
	"time"

	"trend/pkg/logger"
	"trend/pkg/storage"
)

// AnomalyResult 异常检测存储结果数据模型
type AnomalyResult struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ClusterName string    `gorm:"type:varchar(128);index"`
	TaskID      string    `gorm:"type:varchar(128)"`
	CurrentVal  float64   `gorm:"type:double"`
	LowerBound  float64   `gorm:"type:double"`
	UpperBound  float64   `gorm:"type:double"`
	IsAnomaly   bool      `gorm:"type:tinyint(1)"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// ResultStorage 处理向数据库的打点及分表存储逻辑
type ResultStorage struct{}

// NewResultStorage 初始化结果存储处理器
func NewResultStorage() *ResultStorage {
	return &ResultStorage{}
}

// SaveResult 存储结果，根据 ClusterName 自动路由表名
func (r *ResultStorage) SaveResult(ctx context.Context, result *AnomalyResult) error {
	db := storage.GetDB()
	if db == nil {
		logger.Warn("Database not initialized, skipping save result")
		return nil
	}

	tableName := r.getTableName(result.ClusterName)

	// 使用 gorm 的 Table 方法指定动态表名执行插入
	if err := db.Table(tableName).Create(result).Error; err != nil {
		logger.Error("Failed to save anomaly result", logger.String("table", tableName), logger.Err(err))
		return err
	}

	logger.Info("Saved anomaly result", logger.String("table", tableName), logger.String("cluster", result.ClusterName))
	return nil
}

// getTableName 根据 cluster_name 获取分表名
func (r *ResultStorage) getTableName(clusterName string) string {
	hash := crc32.ChecksumIEEE([]byte(clusterName))
	// 按10张表进行 sharding： 00 到 09
	tableIndex := hash % 10
	return fmt.Sprintf("metric_features_%02d", tableIndex)
}
