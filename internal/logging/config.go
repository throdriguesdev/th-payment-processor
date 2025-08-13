package logging

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string
	Format     string
	Output     string
	EnableFile bool
	FilePath   string
}

// LoadConfigFromEnv loads logging configuration from environment variables
func LoadConfigFromEnv() LogConfig {
	return LogConfig{
		Level:      getEnvWithDefault("LOG_LEVEL", "info"),
		Format:     getEnvWithDefault("LOG_FORMAT", "json"),
		Output:     getEnvWithDefault("LOG_OUTPUT", "stdout"),
		EnableFile: getEnvBool("LOG_ENABLE_FILE", false),
		FilePath:   getEnvWithDefault("LOG_FILE_PATH", "logs/app.log"),
	}
}

// ConfigureGlobalLogger configures the global logrus logger with enhanced settings
func ConfigureGlobalLogger(config LogConfig) {
	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', defaulting to info", config.Level)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set formatter
	switch strings.ToLower(config.Format) {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:   "2006-01-02T15:04:05.000Z07:00",
			DisableTimestamp:  false,
			DisableHTMLEscape: true,
			PrettyPrint:       false,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "@timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "caller",
			},
		})
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			DisableColors:   false,
		})
	default:
		logrus.Warnf("Unknown log format '%s', defaulting to JSON", config.Format)
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	// Set output
	switch strings.ToLower(config.Output) {
	case "stdout":
		logrus.SetOutput(os.Stdout)
	case "stderr":
		logrus.SetOutput(os.Stderr)
	default:
		logrus.SetOutput(os.Stdout)
	}

	// Enable caller reporting in debug mode
	if level == logrus.DebugLevel || level == logrus.TraceLevel {
		logrus.SetReportCaller(true)
	}
}

// getEnvWithDefault gets environment variable with default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool gets boolean environment variable
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}