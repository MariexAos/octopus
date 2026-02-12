package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Bloom    BloomConfig    `mapstructure:"bloom"`
	RocketMQ RocketMQConfig `mapstructure:"rocketmq"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	MySQL MySQLConfig `mapstructure:"mysql"`
	Redis RedisConfig `mapstructure:"redis"`
}

// MySQLConfig represents MySQL configuration
type MySQLConfig struct {
	DSN string `mapstructure:"dsn"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// BloomConfig represents Bloom Filter configuration
type BloomConfig struct {
	Capacity  int64   `mapstructure:"capacity"`
	ErrorRate float64 `mapstructure:"error_rate"`
}

// RocketMQConfig represents RocketMQ configuration
type RocketMQConfig struct {
	NameServer string `mapstructure:"nameserver"`
	Topic      string `mapstructure:"topic"`
	Group      string `mapstructure:"group"`
}

// Global config instance
var cfg *Config

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set defaults
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand environment variables
	cfg.Database.Redis.Password = expandEnv(cfg.Database.Redis.Password)
	cfg.Database.MySQL.DSN = expandEnv(cfg.Database.MySQL.DSN)

	return cfg, nil
}

// Get returns the global config instance
func Get() *Config {
	return cfg
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("bloom.capacity", 1000000000)
	v.SetDefault("bloom.error_rate", 0.01)
	v.SetDefault("rocketmq.topic", "access_log")
	v.SetDefault("rocketmq.group", "shortlink_consumer_group")
}

// expandEnv expands environment variables in the string
func expandEnv(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envKey := s[2 : len(s)-1]
		return viper.GetString(envKey)
	}
	return s
}
