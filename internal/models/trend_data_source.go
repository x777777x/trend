package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONMap 是 GORM 中存储 JSON 配置的类型
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, j)
}

// TrendDataSource 数据源注册表（ES/MySQL/Kafka 等）
type TrendDataSource struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	Name       string    `gorm:"type:varchar(64);uniqueIndex;not null;comment:数据源名称"`
	SourceType string    `gorm:"type:varchar(32);not null;comment:数据源类型 elasticsearch/mysql/kafka"`
	Config     JSONMap   `gorm:"type:json;not null;comment:连接配置(JSON)"`
	Status     int8      `gorm:"type:tinyint;not null;default:1;comment:0-禁用 1-启用"`
	CreateTime time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP"`
}

func (TrendDataSource) TableName() string {
	return "trend_data_source"
}

// TrendStorageConfig 存储配置表（Worker 写分位值结果的目标）
type TrendStorageConfig struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	Name       string    `gorm:"type:varchar(64);uniqueIndex;not null;comment:存储名称"`
	SourceType string    `gorm:"type:varchar(32);not null;comment:存储类型 mysql/elasticsearch"`
	Config     JSONMap   `gorm:"type:json;not null;comment:连接配置(JSON)"`
	Status     int8      `gorm:"type:tinyint;not null;default:1;comment:0-禁用 1-启用"`
	CreateTime time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP"`
}

func (TrendStorageConfig) TableName() string {
	return "trend_storage_config"
}
