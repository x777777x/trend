package worker

import (
	"context"

	"trend/pkg/logger"
	"trend/pkg/models"
	"trend/pkg/storage"
)

// DataFetcher 负责从存储引擎拉取数据
type DataFetcher struct{}

// NewDataFetcher 初始化一个数据拉取器
func NewDataFetcher() *DataFetcher {
	return &DataFetcher{}
}

// FetchRealtimeData 获取最近的实时数据
func (d *DataFetcher) FetchRealtimeData(ctx context.Context, cluster string) ([]models.DataPoint, error) {
	// 实际场景：从 ES client 查询最近1分钟或5分钟的数据
	_ = storage.GetES() // 使用 ES client
	logger.Debug("Fetching Realtime data from ES", logger.String("cluster", cluster))

	// 模拟返回数据
	return []models.DataPoint{
		{Timestamp: 1690000000, Value: 85.5},
	}, nil
}

// FetchHistoryData 获取历史同环比数据以供算法分析
func (d *DataFetcher) FetchHistoryData(ctx context.Context, cluster string) ([]models.DataPoint, error) {
	// 实际场景：通过 CRC32 哈希将 Cluster 定位到 metric_features_00..09 中，然后用 gorm 查询数据
	_ = storage.GetDB() // 使用 MySQL client
	logger.Debug("Fetching History data from MySQL", logger.String("cluster", cluster))

	// 模拟返回数据
	return []models.DataPoint{
		{Timestamp: 1689000000, Value: 80.0},
		{Timestamp: 1688000000, Value: 82.5},
		{Timestamp: 1687000000, Value: 83.1},
		{Timestamp: 1686000000, Value: 79.5},
	}, nil
}
