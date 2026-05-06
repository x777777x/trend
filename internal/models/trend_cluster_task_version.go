package models

import (
	"encoding/json"
	"time"
)

// TrendClusterTaskVersion 对应 trend_cluster_task_version 表
// 定义每个任务类型的版本化指标属性列表
type TrendClusterTaskVersion struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;column:id;comment:主键ID"`
	ClusterName string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_cluster_task_version;column:cluster_name;comment:集群名称"`
	TaskName    string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_cluster_task_version;column:task_name;comment:任务类型名称"`
	Version     uint      `gorm:"type:int unsigned;not null;uniqueIndex:uk_cluster_task_version;column:version;comment:版本号"`
	Attributes  string    `gorm:"type:json;not null;column:attributes;comment:指标属性列表 []map[string]string, key=name/type, type=int|float"`
	Description string    `gorm:"type:varchar(256);default:'';column:description;comment:版本描述"`
	CreateTime  time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;column:create_time;comment:创建时间"`
}

// TableName 指定表名
func (TrendClusterTaskVersion) TableName() string {
	return "trend_cluster_task_version"
}

// ParseAttributes 解析 Attributes JSON 为 MetricAttribute 列表
func (v *TrendClusterTaskVersion) ParseAttributes() ([]MetricAttribute, error) {
	var attrs []MetricAttribute
	if err := json.Unmarshal([]byte(v.Attributes), &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

// MetricAttribute 单个指标属性定义
type MetricAttribute struct {
	Name string `json:"name"` // 指标名称: "dml", "cpu_usage"
	Type string `json:"type"` // "int" 或 "float"
}

// MetricValueIndexes 分位值在 metrics_data 内层数组中的位置
// metrics_data[i] = [p99, p95, p90, p70, p50, p30, sample_count]
const (
	IdxP99         = 0
	IdxP95         = 1
	IdxP90         = 2
	IdxP70         = 3
	IdxP50         = 4
	IdxP30         = 5
	IdxSampleCount = 6
)
