package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig

	// Database configuration
	Storage StorageConfig

	// Processor configuration
	Processor ProcessorConfig

	// Logging configuration
	Logging LoggingConfig
}

// ServerConfig holds configuration for the HTTP server
type ServerConfig struct {
	// Port is the HTTP server port
	Port int `mapstructure:"port"`

	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration `mapstructure:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes of the response
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the next request
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

// StorageConfig holds configuration for storage repositories
type StorageConfig struct {
	// Redis configuration
	Redis RedisConfig `mapstructure:"redis"`

	// Persistence type (memory, redis, postgres, etc.)
	PersistenceType string `mapstructure:"persistence_type"`

	// Cache type (memory, redis, etc.)
	CacheType string `mapstructure:"cache_type"`

	// TTL for cache entries in seconds
	CacheTTL int `mapstructure:"cache_ttl"`
}

// RedisConfig holds configuration for Redis
type RedisConfig struct {
	// Address is the Redis server address
	Address string `mapstructure:"address"`

	// Password is the Redis password
	Password string `mapstructure:"password"`

	// DB is the Redis database number
	DB int `mapstructure:"db"`
}

// ProcessorConfig holds configuration for the processing pipeline
type ProcessorConfig struct {
	// QueueCapacity is the maximum number of items in the queue
	QueueCapacity int `mapstructure:"queue_capacity"`

	// MaxAttempts is the maximum number of processing attempts
	MaxAttempts int `mapstructure:"max_attempts"`

	// ProcessingTimeout is the maximum duration for processing an ad
	ProcessingTimeout time.Duration `mapstructure:"processing_timeout"`

	// LockTTL is the TTL for distributed locks in seconds
	LockTTL int `mapstructure:"lock_ttl"`

	// Worker configuration
	Worker WorkerConfig `mapstructure:"worker"`
}

// WorkerConfig holds configuration for workers
type WorkerConfig struct {
	// MinWorkers is the minimum number of workers
	MinWorkers int `mapstructure:"min_workers"`

	// MaxWorkers is the maximum number of workers
	MaxWorkers int `mapstructure:"max_workers"`

	// MaxConcurrentJobs is the maximum number of concurrent jobs per worker
	MaxConcurrentJobs int `mapstructure:"max_concurrent_jobs"`

	// EnableAutoScale determines if auto-scaling is enabled
	EnableAutoScale bool `mapstructure:"enable_auto_scale"`

	// TargetQueueSize is the target queue size for auto-scaling
	TargetQueueSize int `mapstructure:"target_queue_size"`
}

// LoggingConfig holds configuration for logging
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `mapstructure:"level"`

	// Format is the log format (json, text)
	Format string `mapstructure:"format"`
}

var AppConfig *Config

// LoadConfig loads configuration from a file and environment variables
func LoadConfig(configPath string) error {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set up environment variables
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Load configuration file if specified
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	AppConfig = &config

	return nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)

	// Storage defaults
	v.SetDefault("storage.persistence_type", "memory")
	v.SetDefault("storage.cache_type", "memory")
	v.SetDefault("storage.cache_ttl", 3600)
	v.SetDefault("storage.redis.db", 0)

	// Processor defaults
	v.SetDefault("processor.queue_capacity", 10000)
	v.SetDefault("processor.max_attempts", 3)
	v.SetDefault("processor.processing_timeout", 5*time.Minute)
	v.SetDefault("processor.lock_ttl", 30)
	v.SetDefault("processor.worker.min_workers", 2)
	v.SetDefault("processor.worker.max_workers", 10)
	v.SetDefault("processor.worker.max_concurrent_jobs", 0) // 0 means use CPU count
	v.SetDefault("processor.worker.enable_auto_scale", true)
	v.SetDefault("processor.worker.target_queue_size", 100)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Processor.QueueCapacity < 0 {
		return fmt.Errorf("invalid queue capacity: %d", c.Processor.QueueCapacity)
	}

	if c.Processor.MaxAttempts <= 0 {
		return fmt.Errorf("invalid max attempts: %d", c.Processor.MaxAttempts)
	}

	if c.Processor.Worker.MinWorkers <= 0 {
		return fmt.Errorf("invalid min workers: %d", c.Processor.Worker.MinWorkers)
	}

	if c.Processor.Worker.MaxWorkers < c.Processor.Worker.MinWorkers {
		return fmt.Errorf("max workers (%d) must be >= min workers (%d)",
			c.Processor.Worker.MaxWorkers, c.Processor.Worker.MinWorkers)
	}

	// Validate storage configuration
	if c.Storage.CacheType == "redis" && c.Storage.Redis.Address == "" {
		return fmt.Errorf("redis address is required when cache type is redis")
	}

	return nil
}
