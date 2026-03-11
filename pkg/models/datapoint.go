package models

// DataPoint 代表一个监控数据节点
type DataPoint struct {
	Timestamp int64
	Value     float64
}
