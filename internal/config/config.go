package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `envconfig:"SERVER"`
	Database DatabaseConfig `envconfig:"DATABASE"`
	Redis    RedisConfig    `envconfig:"REDIS"`
	App      AppConfig      `envconfig:"APP"`
	Log      LogConfig      `envconfig:"LOG"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port            string        `envconfig:"PORT" default:"8080"`
	GracefulTimeout time.Duration `envconfig:"GRACEFUL_TIMEOUT" default:"30s"`
	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"10s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
	MaxRecvMsgSize  int           `envconfig:"MAX_RECV_MSG_SIZE" default:"4194304"` // 4MB
	MaxSendMsgSize  int           `envconfig:"MAX_SEND_MSG_SIZE" default:"4194304"` // 4MB
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	URL             string        `envconfig:"POSTGRES_URL" required:"true"`
	MaxOpenConns    int           `envconfig:"MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `envconfig:"CONN_MAX_IDLE_TIME" default:"5m"`
}

// RedisConfig holds Redis-related configuration
type RedisConfig struct {
	URL         string        `envconfig:"REDIS_URL" required:"true"`
	PoolSize    int           `envconfig:"POOL_SIZE" default:"10"`
	MinIdleConn int           `envconfig:"MIN_IDLE_CONN" default:"5"`
	DialTimeout time.Duration `envconfig:"DIAL_TIMEOUT" default:"5s"`
	ReadTimeout time.Duration `envconfig:"READ_TIMEOUT" default:"3s"`
	WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" default:"3s"`
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	BaseURL         string        `envconfig:"BASE_URL" default:"http://localhost:8080"`
	ShortCodeLength int           `envconfig:"SHORT_CODE_LENGTH" default:"7"`
	DefaultTTL      time.Duration `envconfig:"DEFAULT_TTL" default:"8760h"` // 1 year
	MaxURLLength    int           `envconfig:"MAX_URL_LENGTH" default:"2048"`
	RateLimit       int           `envconfig:"RATE_LIMIT" default:"100"`
	RateWindow      time.Duration `envconfig:"RATE_WINDOW" default:"1m"`
	CacheTTL        time.Duration `envconfig:"CACHE_TTL" default:"1h"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `envconfig:"LEVEL" default:"info"`
	Format string `envconfig:"FORMAT" default:"json"` // json or console
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.App.ShortCodeLength < 4 || c.App.ShortCodeLength > 10 {
		c.App.ShortCodeLength = 7
	}
	
	if c.App.MaxURLLength < 100 || c.App.MaxURLLength > 4096 {
		c.App.MaxURLLength = 2048
	}
	
	if c.App.RateLimit < 1 {
		c.App.RateLimit = 100
	}
	
	return nil
}
