package models

import (
	"time"
)

// TrendOrzdbaCalcInstance 对应 trend_orzdba_calc_instances 表
type TrendOrzdbaCalcInstance struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement;column:id;comment:主键ID，自增"`
	ClusterName  string    `gorm:"type:varchar(64);uniqueIndex:uk_cluster_instance;column:cluster_name;comment:集群名称"`
	InstanceName string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_cluster_instance;column:instance_name;comment:数据库实例名称"`
	Status       int8      `gorm:"type:tinyint;not null;default:0;column:status;comment:任务状态：0-禁用，1-启用"`
	LastTime     time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;column:last_time;comment:滑动窗口结束时间"`
	CreateTime   time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;column:create_time;comment:创建时间"`
	UpdateTime   time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP;column:update_time;comment:修改时间"`
}

// TableName 指定表名
func (TrendOrzdbaCalcInstance) TableName() string {
	return "trend_orzdba_calc_instance"
}

// TaskCalcInstance 通用的任务实例配置模型，支持动态表名
// 表名规则：trend_{task_type}_calc_instance
type TaskCalcInstance struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement;column:id;comment:主键ID，自增"`
	ClusterName  string    `gorm:"type:varchar(64);uniqueIndex:uk_cluster_instance;column:cluster_name;comment:集群名称"`
	InstanceName string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_cluster_instance;column:instance_name;comment:数据库实例名称"`
	Status       int8      `gorm:"type:tinyint;not null;default:0;column:status;comment:任务状态：0-禁用，1-启用"`
	LastTime     time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;column:last_time;comment:滑动窗口结束时间"`
	CreateTime   time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;column:create_time;comment:创建时间"`
	UpdateTime   time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP;column:update_time;comment:修改时间"`
	tableName    string
}

// TableName 返回动态表名
func (t *TaskCalcInstance) TableName() string {
	return t.tableName
}

// NewTaskCalcInstance 为指定任务类型创建动态表名的 CalcInstance 实例
func NewTaskCalcInstance(taskType string) *TaskCalcInstance {
	return &TaskCalcInstance{
		tableName: "trend_" + taskType + "_calc_instance",
	}
}
