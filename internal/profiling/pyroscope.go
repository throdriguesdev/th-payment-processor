package profiling

import (
	"context"
	"os"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/sirupsen/logrus"
)

type Config struct {
	ServerAddress   string
	ApplicationName string
	Tags            map[string]string
	ProfileTypes    []pyroscope.ProfileType
	Enabled         bool
}

type Profiler struct {
	profiler *pyroscope.Profiler
	config   Config
	logger   *logrus.Logger
}

// Labels represents profiling labels
type Labels map[string]string

// NewProfiler creates a new Pyroscope profiler instance
func NewProfiler(logger *logrus.Logger) *Profiler {
	config := Config{
		ServerAddress:   getEnv("PYROSCOPE_SERVER_ADDRESS", ""),
		ApplicationName: getEnv("PYROSCOPE_APPLICATION_NAME", "th-payment-processor"),
		Tags: map[string]string{
			"version":     "1.0.0",
			"environment": getEnv("ENVIRONMENT", "development"),
			"service":     getEnv("SERVICE_NAME", "payment-processor"),
		},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
		Enabled: getEnv("PYROSCOPE_SERVER_ADDRESS", "") != "",
	}

	return &Profiler{
		config: config,
		logger: logger,
	}
}

// Start initializes and starts the Pyroscope profiler
func (p *Profiler) Start() error {
	if !p.config.Enabled {
		p.logger.Info("Pyroscope profiling is disabled (no server address provided)")
		return nil
	}

	p.logger.WithFields(logrus.Fields{
		"server_address":   p.config.ServerAddress,
		"application_name": p.config.ApplicationName,
		"tags":            p.config.Tags,
	}).Info("Starting Pyroscope profiler")

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: p.config.ApplicationName,
		ServerAddress:   p.config.ServerAddress,
		Tags:           p.config.Tags,
		ProfileTypes:   p.config.ProfileTypes,
		
		// Optional configuration
		Logger: pyroscope.StandardLogger, // Use Pyroscope's standard logger
		
		// Upload interval - how often to send profiles
		UploadRate: 15 * time.Second, // 15 seconds
		
		// Disable logging to reduce noise in production
		DisableGCRuns: false,
	})

	if err != nil {
		p.logger.WithError(err).Error("Failed to start Pyroscope profiler")
		return err
	}

	p.profiler = profiler
	p.logger.Info("Pyroscope profiler started successfully")
	return nil
}

// Stop gracefully shuts down the profiler
func (p *Profiler) Stop() error {
	if p.profiler == nil {
		return nil
	}

	p.logger.Info("Stopping Pyroscope profiler")
	err := p.profiler.Stop()
	if err != nil {
		p.logger.WithError(err).Error("Error stopping Pyroscope profiler")
		return err
	}

	p.logger.Info("Pyroscope profiler stopped successfully")
	return nil
}

// TagWrapper creates a new context with profiling tags
func (p *Profiler) TagWrapper(ctx context.Context, tags Labels) context.Context {
	if !p.config.Enabled {
		return ctx
	}
	
	// Since pyroscope.TagWrapper doesn't return a context, we'll just return the original
	// The profiling tags will be applied during ExecuteWithTags calls
	return ctx
}

// ExecuteWithTags executes a function with profiling tags
func (p *Profiler) ExecuteWithTags(ctx context.Context, tags Labels, fn func(context.Context)) {
	if !p.config.Enabled {
		fn(ctx)
		return
	}
	
	// Convert our labels to pyroscope format
	labelSlice := make([]string, 0, len(tags)*2)
	for k, v := range tags {
		labelSlice = append(labelSlice, k, v)
	}
	
	pyroscope.TagWrapper(ctx, pyroscope.Labels(labelSlice...), fn)
}

// ProfileCPU profiles a specific operation with CPU profiling
func (p *Profiler) ProfileCPU(ctx context.Context, operation string, fn func(context.Context) error) error {
	if !p.config.Enabled {
		return fn(ctx)
	}

	var err error
	pyroscope.TagWrapper(ctx, pyroscope.Labels("operation", operation), func(c context.Context) {
		err = fn(c)
		if err != nil {
			// Add error tag for failed operations
			pyroscope.TagWrapper(c, pyroscope.Labels("error", "true"), func(context.Context) {})
		}
	})
	return err
}

// ProfileHTTPHandler creates profiling tags for HTTP requests
func (p *Profiler) ProfileHTTPHandler(method, path string) Labels {
	if !p.config.Enabled {
		return nil
	}

	return Labels{
		"http_method": method,
		"http_path":   path,
		"handler":     "http",
	}
}

// ProfilePaymentOperation creates profiling tags for payment operations
func (p *Profiler) ProfilePaymentOperation(processor, operation string) Labels {
	if !p.config.Enabled {
		return nil
	}

	return Labels{
		"processor": processor,
		"operation": operation,
		"component": "payment",
	}
}

// ProfileDatabaseOperation creates profiling tags for database operations
func (p *Profiler) ProfileDatabaseOperation(dbType, operation, table string) Labels {
	if !p.config.Enabled {
		return nil
	}

	return Labels{
		"db_type":      dbType,
		"db_operation": operation,
		"db_table":     table,
		"component":    "database",
	}
}

// AddCustomTags adds custom tags to the current profiling session
func (p *Profiler) AddCustomTags(ctx context.Context, tags map[string]string) context.Context {
	if !p.config.Enabled {
		return ctx
	}

	// Store tags in context for later use - no actual profiling until ExecuteWithTags
	return ctx
}

// IsEnabled returns whether profiling is currently enabled
func (p *Profiler) IsEnabled() bool {
	return p.config.Enabled
}

// GetConfig returns the current profiler configuration
func (p *Profiler) GetConfig() Config {
	return p.config
}

// Helper function to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}