package models

import (
	"time"
)

// TrendClusterTask 对应 trend_cluster_task 表
type TrendClusterTask struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement;column:id;comment:主键ID，自增"`
	ClusterName   string    `gorm:"type:varchar(64);not null;column:cluster_name;comment:集群名称"`
	TaskName      string    `gorm:"type:varchar(64);not null;column:task_name;comment:任务类型名称"`
	Status        int8      `gorm:"type:tinyint;not null;default:0;column:status;comment:任务状态：0-禁用，1-启用"`
	SlideInterval uint      `gorm:"type:int unsigned;not null;default:0;column:slide_interval;comment:滑动间隔"`
	CreateTime    time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;column:create_time;comment:创建时间"`
	UpdateTime    time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP;column:update_time;comment:修改时间"`
}

// TableName 指定表名
func (TrendClusterTask) TableName() string {
	return "trend_cluster_task"
}
