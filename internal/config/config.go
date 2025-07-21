package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Judge    JudgeConfig    `yaml:"judge"`
	Security SecurityConfig `yaml:"security"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         string        `yaml:"port" env:"SERVER_PORT"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string        `yaml:"host" env:"DB_HOST"`
	Port            string        `yaml:"port" env:"DB_PORT"`
	User            string        `yaml:"user" env:"DB_USER"`
	Password        string        `yaml:"password" env:"DB_PASSWORD"`
	Name            string        `yaml:"name" env:"DB_NAME"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string        `yaml:"secret" env:"JWT_SECRET"`
	Expiration time.Duration `yaml:"expiration" env:"JWT_EXPIRATION"`
	Issuer     string        `yaml:"issuer" env:"JWT_ISSUER"`
}

// JudgeConfig 评测系统配置
type JudgeConfig struct {
	DataRoot           string        `yaml:"data_root" env:"JUDGE_DATA_ROOT"`
	WorkspaceRoot      string        `yaml:"workspace_root" env:"JUDGE_WORKSPACE_ROOT"`
	MaxWorkers         int           `yaml:"max_workers" env:"JUDGE_MAX_WORKERS"`
	DefaultTimeLimit   int           `yaml:"default_time_limit" env:"JUDGE_DEFAULT_TIME_LIMIT"`
	DefaultMemoryLimit int           `yaml:"default_memory_limit" env:"JUDGE_DEFAULT_MEMORY_LIMIT"`
	CleanupInterval    time.Duration `yaml:"cleanup_interval" env:"JUDGE_CLEANUP_INTERVAL"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RateLimitRequests int           `yaml:"rate_limit_requests" env:"RATE_LIMIT_REQUESTS"`
	RateLimitWindow   time.Duration `yaml:"rate_limit_window" env:"RATE_LIMIT_WINDOW"`
	MaxRequestSize    int64         `yaml:"max_request_size" env:"MAX_REQUEST_SIZE"`
	AllowedOrigins    []string      `yaml:"allowed_origins" env:"ALLOWED_ORIGINS"`
	EnableCORS        bool          `yaml:"enable_cors" env:"ENABLE_CORS"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level" env:"LOG_LEVEL"`
	Format     string `yaml:"format" env:"LOG_FORMAT"`
	Output     string `yaml:"output" env:"LOG_OUTPUT"`
	MaxSize    int    `yaml:"max_size" env:"LOG_MAX_SIZE"`
	MaxBackups int    `yaml:"max_backups" env:"LOG_MAX_BACKUPS"`
	MaxAge     int    `yaml:"max_age" env:"LOG_MAX_AGE"`
	Compress   bool   `yaml:"compress" env:"LOG_COMPRESS"`
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	// 加载环境变量文件
	if err := godotenv.Load(); err != nil {
		// 非致命错误，继续使用环境变量
		fmt.Println("加载环境变量文件失败", err)
	}

	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "3306"),
			User:            getEnv("DB_USER", "root"),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", "ggcode"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 1*time.Hour),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
			Expiration: getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
			Issuer:     getEnv("JWT_ISSUER", "ggcode"),
		},
		Judge: JudgeConfig{
			DataRoot:           getEnv("JUDGE_DATA_ROOT", "/judge/data"),
			WorkspaceRoot:      getEnv("JUDGE_WORKSPACE_ROOT", "/tmp/ggcode_workspace"),
			MaxWorkers:         getEnvInt("JUDGE_MAX_WORKERS", 10),
			DefaultTimeLimit:   getEnvInt("JUDGE_DEFAULT_TIME_LIMIT", 5),
			DefaultMemoryLimit: getEnvInt("JUDGE_DEFAULT_MEMORY_LIMIT", 128),
			CleanupInterval:    getEnvDuration("JUDGE_CLEANUP_INTERVAL", 1*time.Hour),
		},
		Security: SecurityConfig{
			RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:   getEnvDuration("RATE_LIMIT_WINDOW", 1*time.Minute),
			MaxRequestSize:    getEnvInt64("MAX_REQUEST_SIZE", 10*1024*1024), // 10MB
			AllowedOrigins:    getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),
			EnableCORS:        getEnvBool("ENABLE_CORS", true),
		},
		Log: LogConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			MaxSize:    getEnvInt("LOG_MAX_SIZE", 100),
			MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 10),
			MaxAge:     getEnvInt("LOG_MAX_AGE", 7),
			Compress:   getEnvBool("LOG_COMPRESS", true),
		},
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return config, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("服务器端口不能为空")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}

	if c.JWT.Secret == "your-secret-key-change-this-in-production" {
		fmt.Println("[警告] 生产环境请设置自定义 JWT 密钥！")
	}

	return nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

// 辅助函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// 简单的逗号分隔解析
		return []string{value}
	}
	return defaultValue
}
