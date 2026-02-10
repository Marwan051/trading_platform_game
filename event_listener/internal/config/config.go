package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	GRPCAddr         string
	Environment      string
	ShutdownTimeout  time.Duration
	ValkeyHost       string
	ValkeyPort       int
	ValkeyStreamName string
}

func Load() *Config {
	return &Config{
		GRPCAddr:         getEnv("GRPC_ADDR", ":50051"),
		Environment:      getEnv("ENVIRONMENT", "development"),
		ShutdownTimeout:  getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		ValkeyHost:       getEnv("VALKEY_HOST", "localhost"),
		ValkeyPort:       getIntEnv("VALKEY_PORT", 6379),
		ValkeyStreamName: getEnv("VALKEY_STREAM_NAME", "matching_engine_stream"),
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

func getIntEnv(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
