package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort string
	DefaultProcessorURL string
	FallbackProcessorURL string
	HealthCheckInterval time.Duration
	RequestTimeout time.Duration
	
	// Redis Configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	
	// PostgreSQL Configuration
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string
}

func Load() *Config {
	serverPort := getEnv("SERVER_PORT", "8080")
	defaultProcessorURL := getEnv("DEFAULT_PROCESSOR_URL", "http://payment-processor-default:8080")
	fallbackProcessorURL := getEnv("FALLBACK_PROCESSOR_URL", "http://payment-processor-fallback:8080")
	
	healthCheckInterval := getEnvAsDuration("HEALTH_CHECK_INTERVAL", 5*time.Second)
	requestTimeout := getEnvAsDuration("REQUEST_TIMEOUT", 2*time.Second)

	// Redis configuration
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvAsInt("REDIS_DB", 0)

	// PostgreSQL configuration
	postgresHost := getEnv("POSTGRES_HOST", "localhost")
	postgresPort := getEnv("POSTGRES_PORT", "5432")
	postgresUser := getEnv("POSTGRES_USER", "postgres")
	postgresPassword := getEnv("POSTGRES_PASSWORD", "")
	postgresDB := getEnv("POSTGRES_DB", "payments")
	postgresSSLMode := getEnv("POSTGRES_SSLMODE", "disable")

	return &Config{
		ServerPort: serverPort,
		DefaultProcessorURL: defaultProcessorURL,
		FallbackProcessorURL: fallbackProcessorURL,
		HealthCheckInterval: healthCheckInterval,
		RequestTimeout: requestTimeout,
		RedisAddr: redisAddr,
		RedisPassword: redisPassword,
		RedisDB: redisDB,
		PostgresHost: postgresHost,
		PostgresPort: postgresPort,
		PostgresUser: postgresUser,
		PostgresPassword: postgresPassword,
		PostgresDB: postgresDB,
		PostgresSSLMode: postgresSSLMode,
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

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
} 