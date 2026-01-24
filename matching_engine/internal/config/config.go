package config

import (
	"os"
	"time"
)

type Config struct {
	GRPCAddr        string
	Environment     string
	ShutdownTimeout time.Duration
}

func Load() *Config {
	return &Config{
		GRPCAddr:        getEnv("GRPC_ADDR", ":50051"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
