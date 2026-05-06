package task

import (
	"time"

	"trend/internal/models"
)

// Task 定义了所有任务类型都需要实现的基础接口
type Task interface {
	GetID() string
	GetClusterName() string
	GetHost() string
	GetLastTime() time.Time
	GetType() string
	GetSlideInterval() uint
	GetCalcInstanceID() uint64
	GetVersion() uint
	GetAttributes() []models.MetricAttribute
	Serialize() ([]byte, error)
	Run() error
}
