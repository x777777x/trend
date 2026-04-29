package storage

import (
	"fmt"

	"trend/internal/models"
)

// SaveQuantileResult 将分位值结果存入 Worker 端 MySQL 对应的分表
func SaveQuantileResult(result *models.TrendQuantileResult, calcInstanceID uint64) error {
	if WorkerDB == nil {
		return fmt.Errorf("worker database not initialized")
	}

	tableName := getQuantileTableName(calcInstanceID)
	return WorkerDB.Table(tableName).Create(result).Error
}

// getQuantileTableName 根据 calc_instance_id 哈希路由到分表
func getQuantileTableName(calcInstanceID uint64) string {
	tableIndex := calcInstanceID % 10
	return fmt.Sprintf("metric_features_%02d", tableIndex)
}
