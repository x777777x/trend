package models

import "time"

// TrendQuantileResult 存储分位值计算结果
// 每个窗口一行，多个指标的分位值以 JSON 数组存储在 MetricsData 中
// MetricsData 格式：[][]float64
//   外层数组索引 = MetricAttribute 在版本配置中的顺序
//   内层数组 = [p99, p95, p90, p70, p50, p30, sample_count]
type TrendQuantileResult struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ClusterName string    `gorm:"type:varchar(128);index;not null"`
	TaskID      string    `gorm:"type:varchar(128);not null"`
	Host        string    `gorm:"type:varchar(128);not null"`
	Version     uint      `gorm:"type:int unsigned;not null;column:version"`
	MetricsData string    `gorm:"type:json;not null;column:metrics_data"`
	WindowStart time.Time `gorm:"type:datetime;not null"`
	WindowEnd   time.Time `gorm:"type:datetime;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// TableName 指定基础表名，实际存储时通过分表逻辑动态指定
func (TrendQuantileResult) TableName() string {
	return "trend_quantile_result"
}
