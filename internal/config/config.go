package config

import (
	"os"
	"time"
)

type Config struct {
	ServerPort string
	DefaultProcessorURL string
	FallbackProcessorURL string
	HealthCheckInterval time.Duration
	RequestTimeout time.Duration
}

func Load() *Config {
	serverPort := getEnv("SERVER_PORT", "8080")
	defaultProcessorURL := getEnv("DEFAULT_PROCESSOR_URL", "http://payment-processor-default:8080")
	fallbackProcessorURL := getEnv("FALLBACK_PROCESSOR_URL", "http://payment-processor-fallback:8080")
	
	healthCheckInterval := getEnvAsDuration("HEALTH_CHECK_INTERVAL", 5*time.Second)
	requestTimeout := getEnvAsDuration("REQUEST_TIMEOUT", 10*time.Second)

	return &Config{
		ServerPort: serverPort,
		DefaultProcessorURL: defaultProcessorURL,
		FallbackProcessorURL: fallbackProcessorURL,
		HealthCheckInterval: healthCheckInterval,
		RequestTimeout: requestTimeout,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
} 