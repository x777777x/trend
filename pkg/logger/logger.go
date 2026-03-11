package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Log *zap.Logger
)

// Config 定义日志配置
type Config struct {
	Level      string `yaml:"level"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
	Console    bool   `yaml:"console"`
}

// Init 初始化全局日志实例
func Init(cfg *Config) error {
	var core zapcore.Core

	// 解析日志级别
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// JSON 编码器
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	var cores []zapcore.Core

	if cfg.Console {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	if cfg.Filename != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level))
	}

	core = zapcore.NewTee(cores...)
	Log = zap.New(core, zap.AddCaller())

	// 替换内建的全局 Logger
	zap.ReplaceGlobals(Log)
	return nil
}

// Debug 封装 zap.Debug
func Debug(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Debug(msg, fields...)
	}
}

// Info 封装 zap.Info
func Info(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Info(msg, fields...)
	}
}

// Warn 封装 zap.Warn
func Warn(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Warn(msg, fields...)
	}
}

// Error 封装 zap.Error
func Error(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Error(msg, fields...)
	}
}

// Fatal 封装 zap.Fatal
func Fatal(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Fatal(msg, fields...)
	}
}

// Err 封装 zap.Error
func Err(err error) zap.Field {
	return zap.Error(err)
}

// String 封装 zap.String
func String(key string, val string) zap.Field {
	return zap.String(key, val)
}

// Float64 封装 zap.Float64
func Float64(key string, val float64) zap.Field {
	return zap.Float64(key, val)
}

// Duration 封装 zap.Duration
func Duration(key string, val time.Duration) zap.Field {
	return zap.Duration(key, val)
}

// Int 封装 zap.Int
func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}
