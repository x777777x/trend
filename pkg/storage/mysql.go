package storage

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trend/internal/models"
)

// MySQLConfig 定义 MySQL 数据库配置
type MySQLConfig struct {
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	DBName          string `mapstructure:"dbname"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	Debug           bool   `mapstructure:"debug"`
}

// MasterDB 用于 Master 端读取任务配置
var MasterDB *gorm.DB

// WorkerDB 用于 Worker 端写入分位值结果（可能与 MasterDB 指向不同实例）
var WorkerDB *gorm.DB

// ResultsDB 只读连接，用于 Master 端查询趋势分位值结果
var ResultsDB *gorm.DB

func openMySQL(dsn string, logLevel logger.LogLevel) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func configurePool(db *gorm.DB, cfg *MySQLConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Hour)
	return nil
}

// InitMasterMySQL 初始化 Master 端 MySQL 连接（读取任务配置）
func InitMasterMySQL(cfg *MySQLConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	db, err := openMySQL(dsn, logLevel)
	if err != nil {
		return err
	}
	if err := configurePool(db, cfg); err != nil {
		return err
	}
	MasterDB = db

	if err := db.AutoMigrate(
		&models.TrendClusterTask{},
		&models.TrendOrzdbaCalcInstance{},
		&models.TrendDataSource{},
		&models.TrendStorageConfig{},
	); err != nil {
		return fmt.Errorf("failed to auto migrate master tables: %w", err)
	}
	return nil
}

// InitWorkerMySQL 初始化 Worker 端 MySQL 连接（写入分位值结果）
func InitWorkerMySQL(cfg *MySQLConfig) error {
	if WorkerDB != nil {
		sqlDB, _ := WorkerDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	db, err := openMySQL(dsn, logLevel)
	if err != nil {
		return err
	}
	if err := configurePool(db, cfg); err != nil {
		return err
	}
	WorkerDB = db

	if err := db.AutoMigrate(&models.TrendQuantileResult{}); err != nil {
		return fmt.Errorf("failed to auto migrate worker tables: %w", err)
	}
	return nil
}

// GetDB 获取 Master 端数据库连接（向后兼容）
func GetDB() *gorm.DB {
	return MasterDB
}

// GetWorkerDB 获取 Worker 端数据库连接
func GetWorkerDB() *gorm.DB {
	return WorkerDB
}

// InitResultsMySQL 初始化 Results 端 MySQL 只读连接（查询趋势分位值结果）
func InitResultsMySQL(cfg *MySQLConfig) error {
	if ResultsDB != nil {
		sqlDB, _ := ResultsDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	db, err := openMySQL(dsn, logLevel)
	if err != nil {
		return err
	}
	if err := configurePool(db, cfg); err != nil {
		return err
	}
	ResultsDB = db
	// 不执行 AutoMigrate，只读连接
	return nil
}

// GetResultsDB 获取 Results 端只读数据库连接
func GetResultsDB() *gorm.DB {
	return ResultsDB
}

// InitMySQL 兼容旧调用：同时初始化 MasterDB 和 WorkerDB（指向同一实例）
func InitMySQL(cfg *MySQLConfig) error {
	if err := InitMasterMySQL(cfg); err != nil {
		return err
	}
	if err := InitWorkerMySQL(cfg); err != nil {
		return err
	}
	return nil
}

// EnsureTaskCalcInstanceTables 检查并创建各任务类型对应的实例配置表
// 表名规则：trend_{task_type}_calc_instance
func EnsureTaskCalcInstanceTables() error {
	if MasterDB == nil {
		return fmt.Errorf("master database not initialized")
	}

	// 查询所有已启用的任务类型
	var taskTypes []string
	err := MasterDB.Table("trend_cluster_task").
		Distinct("task_name").
		Where("status = ?", 1).
		Pluck("task_name", &taskTypes).Error
	if err != nil {
		return fmt.Errorf("failed to query task types: %w", err)
	}

	for _, taskType := range taskTypes {
		tableName := fmt.Sprintf("trend_%s_calc_instance", taskType)
		if MasterDB.Migrator().HasTable(tableName) {
			continue
		}

		calcInst := models.NewTaskCalcInstance(taskType)
		if err := MasterDB.AutoMigrate(calcInst); err != nil {
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
	}

	return nil
}
