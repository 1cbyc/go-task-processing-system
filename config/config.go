package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Server   ServerConfig
	Worker   WorkerConfig
	Database DatabaseConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port         int
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type WorkerConfig struct {
	NumWorkers     int
	MaxConcurrency int
	JobTimeout     time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	SSLMode  string
}

type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			Host:         getEnvAsString("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Worker: WorkerConfig{
			NumWorkers:     getEnvAsInt("WORKER_NUM_WORKERS", 10),
			MaxConcurrency: getEnvAsInt("WORKER_MAX_CONCURRENCY", 100),
			JobTimeout:     getEnvAsDuration("WORKER_JOB_TIMEOUT", 5*time.Minute),
			RetryAttempts:  getEnvAsInt("WORKER_RETRY_ATTEMPTS", 3),
			RetryDelay:     getEnvAsDuration("WORKER_RETRY_DELAY", 1*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnvAsString("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			Username: getEnvAsString("DB_USERNAME", "postgres"),
			Password: getEnvAsString("DB_PASSWORD", ""),
			Database: getEnvAsString("DB_DATABASE", "task_processor"),
			SSLMode:  getEnvAsString("DB_SSL_MODE", "disable"),
		},
		Logging: LoggingConfig{
			Level:  getEnvAsString("LOG_LEVEL", "info"),
			Format: getEnvAsString("LOG_FORMAT", "json"),
			Output: getEnvAsString("LOG_OUTPUT", "stdout"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Worker.NumWorkers <= 0 {
		return fmt.Errorf("worker count must be greater than 0")
	}
	if c.Worker.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be greater than 0")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	return nil
}

func getEnvAsString(name, defaultVal string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	if env, exists := os.LookupEnv(name); exists {
		if val, err := strconv.Atoi(env); err == nil {
			return val
		}
		logrus.Warnf("invalid value for %s, using default: %d", name, defaultVal)
	}
	return defaultVal
}

func getEnvAsDuration(name string, defaultVal time.Duration) time.Duration {
	if env, exists := os.LookupEnv(name); exists {
		if val, err := time.ParseDuration(env); err == nil {
			return val
		}
		logrus.Warnf("invalid duration for %s, using default: %v", name, defaultVal)
	}
	return defaultVal
}
