package logger

import (
	"fmt"
	"ggcode/internal/config"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *logrus.Logger

// InitLogger 初始化日志系统
func InitLogger(cfg *config.Config) error {
	log = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("无效的日志级别 %s: %v", cfg.Log.Level, err)
	}
	log.SetLevel(level)

	// 设置日志格式
	switch cfg.Log.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	default:
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	}

	// 设置日志输出
	switch cfg.Log.Output {
	case "stdout":
		log.SetOutput(os.Stdout)
	case "stderr":
		log.SetOutput(os.Stderr)
	default:
		// 文件输出
		if err := setupFileOutput(cfg.Log); err != nil {
			return fmt.Errorf("设置文件输出失败: %v", err)
		}
	}

	// 添加默认字段
	log.AddHook(&DefaultFieldsHook{})

	return nil
}

// setupFileOutput 设置文件输出
func setupFileOutput(cfg config.LogConfig) error {
	// 确保日志目录存在
	logDir := filepath.Dir(cfg.Output)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 配置日志轮转
	writer := &lumberjack.Logger{
		Filename:   cfg.Output,
		MaxSize:    cfg.MaxSize, // MB
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge, // 天
		Compress:   cfg.Compress,
	}

	log.SetOutput(writer)
	return nil
}

// DefaultFieldsHook 默认字段钩子
type DefaultFieldsHook struct{}

func (h *DefaultFieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *DefaultFieldsHook) Fire(entry *logrus.Entry) error {
	entry.Data["service"] = "ggcode"
	entry.Data["version"] = "2.3.0"
	return nil
}

// GetLogger 获取日志实例
func GetLogger() *logrus.Logger {
	if log == nil {
		// 如果未初始化，使用默认配置
		log = logrus.New()
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
		log.SetOutput(os.Stdout)
		log.SetLevel(logrus.InfoLevel)
	}
	return log
}

// 便捷方法
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// WithField 添加字段
func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

// WithFields 添加多个字段
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// WithError 添加错误
func WithError(err error) *logrus.Entry {
	return GetLogger().WithError(err)
}
