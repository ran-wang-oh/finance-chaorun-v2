package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

type ServerConfig struct {
	HTTPAddr     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	DSN string
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPAddr:     env("HTTP_ADDR", ":9000"),
			ReadTimeout:  durationEnv("READ_TIMEOUT_SECONDS", 10*time.Second),
			WriteTimeout: durationEnv("WRITE_TIMEOUT_SECONDS", 30*time.Second),
			IdleTimeout:  durationEnv("IDLE_TIMEOUT_SECONDS", 60*time.Second),
		},
		Database: DatabaseConfig{
			DSN: env("DATABASE_DSN", "postgres://chaorun:chaorun_dev@localhost:5432/chaorun_finance?sslmode=disable"),
		},
	}
}

func (c *Config) Validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}
	return nil
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
