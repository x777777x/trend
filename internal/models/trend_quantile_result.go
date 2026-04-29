package models

import "time"

// TrendQuantileResult 存储分位值计算结果，通过 CRC32 分表存储
type TrendQuantileResult struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ClusterName string    `gorm:"type:varchar(128);index;not null"`
	TaskID      string    `gorm:"type:varchar(128);not null"`
	Host        string    `gorm:"type:varchar(128);not null"`
	MetricName  string    `gorm:"type:varchar(64);not null"`
	P99         float64   `gorm:"type:double;not null"`
	P95         float64   `gorm:"type:double;not null"`
	P90         float64   `gorm:"type:double;not null"`
	P70         float64   `gorm:"type:double;not null"`
	P50         float64   `gorm:"type:double;not null"`
	P30         float64   `gorm:"type:double;not null"`
	SampleCount int       `gorm:"type:int;not null"`
	WindowStart time.Time `gorm:"type:datetime;not null"`
	WindowEnd   time.Time `gorm:"type:datetime;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// TableName 指定基础表名，实际存储时通过分表逻辑动态指定
func (TrendQuantileResult) TableName() string {
	return "trend_quantile_result"
}
