package logging

import (
	"context"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

const (
	ServiceNameKey    = "service_name"
	CorrelationIDKey  = "correlation_id"
	TraceIDKey        = "trace_id"
	SpanIDKey         = "span_id"
	ComponentKey      = "component"
	OperationKey      = "operation"
	UserIDKey         = "user_id"
	PaymentIDKey      = "payment_id"
	ProcessorKey      = "processor"
	AmountKey         = "amount"
	StatusKey         = "status"
	LatencyKey        = "latency_ms"
	ErrorCodeKey      = "error_code"
	ErrorMessageKey   = "error_message"
	HTTPMethodKey     = "http_method"
	HTTPStatusKey     = "http_status"
	HTTPURLKey        = "http_url"
	RemoteAddrKey     = "remote_addr"
)

// LogLevel constants
const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	FatalLevel = "fatal"
)

// PaymentFields contains payment-specific log fields
type PaymentFields struct {
	PaymentID     string  `json:"payment_id,omitempty"`
	CorrelationID string  `json:"correlation_id,omitempty"`
	Amount        float64 `json:"amount,omitempty"`
	Processor     string  `json:"processor,omitempty"`
	Status        string  `json:"status,omitempty"`
	UserID        string  `json:"user_id,omitempty"`
}

// HTTPFields contains HTTP-specific log fields
type HTTPFields struct {
	Method     string `json:"method,omitempty"`
	URL        string `json:"url,omitempty"`
	Status     int    `json:"status,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

// TraceFields contains tracing-specific log fields
type TraceFields struct {
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`
}

// StructuredLogger wraps logrus with enhanced context and trace information
type StructuredLogger struct {
	logger    *logrus.Logger
	component string
	ctx       context.Context
}

// NewStructuredLogger creates a new structured logger with default configuration
func NewStructuredLogger(component string) *StructuredLogger {
	logger := logrus.New()
	
	// Configure JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat:   "2006-01-02T15:04:05.000Z07:00",
		DisableTimestamp:  false,
		DisableHTMLEscape: true,
		PrettyPrint:       false,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "@timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	
	// Set log level from environment or default to Info
	level := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(level) {
	case DebugLevel:
		logger.SetLevel(logrus.DebugLevel)
	case InfoLevel:
		logger.SetLevel(logrus.InfoLevel)
	case WarnLevel:
		logger.SetLevel(logrus.WarnLevel)
	case ErrorLevel:
		logger.SetLevel(logrus.ErrorLevel)
	case FatalLevel:
		logger.SetLevel(logrus.FatalLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
	
	logger.SetOutput(os.Stdout)
	
	return &StructuredLogger{
		logger:    logger,
		component: component,
		ctx:       context.Background(),
	}
}

// WithContext creates a new logger with the given context
func (sl *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	return &StructuredLogger{
		logger:    sl.logger,
		component: sl.component,
		ctx:       ctx,
	}
}

// WithPaymentFields adds payment-specific fields to the logger
func (sl *StructuredLogger) WithPaymentFields(fields PaymentFields) *logrus.Entry {
	baseEntry := sl.getBaseEntry()
	
	if fields.PaymentID != "" {
		baseEntry = baseEntry.WithField(PaymentIDKey, fields.PaymentID)
	}
	if fields.CorrelationID != "" {
		baseEntry = baseEntry.WithField(CorrelationIDKey, fields.CorrelationID)
	}
	if fields.Amount > 0 {
		baseEntry = baseEntry.WithField(AmountKey, fields.Amount)
	}
	if fields.Processor != "" {
		baseEntry = baseEntry.WithField(ProcessorKey, fields.Processor)
	}
	if fields.Status != "" {
		baseEntry = baseEntry.WithField(StatusKey, fields.Status)
	}
	if fields.UserID != "" {
		baseEntry = baseEntry.WithField(UserIDKey, fields.UserID)
	}
	
	return baseEntry
}

// WithHTTPFields adds HTTP-specific fields to the logger
func (sl *StructuredLogger) WithHTTPFields(fields HTTPFields) *logrus.Entry {
	baseEntry := sl.getBaseEntry()
	
	if fields.Method != "" {
		baseEntry = baseEntry.WithField(HTTPMethodKey, fields.Method)
	}
	if fields.URL != "" {
		baseEntry = baseEntry.WithField(HTTPURLKey, fields.URL)
	}
	if fields.Status > 0 {
		baseEntry = baseEntry.WithField(HTTPStatusKey, fields.Status)
	}
	if fields.RemoteAddr != "" {
		baseEntry = baseEntry.WithField(RemoteAddrKey, fields.RemoteAddr)
	}
	if fields.UserAgent != "" {
		baseEntry = baseEntry.WithField("user_agent", fields.UserAgent)
	}
	
	return baseEntry
}

// WithOperation adds operation-specific context
func (sl *StructuredLogger) WithOperation(operation string) *logrus.Entry {
	return sl.getBaseEntry().WithField(OperationKey, operation)
}

// WithField adds a single field to the logger
func (sl *StructuredLogger) WithField(key string, value interface{}) *logrus.Entry {
	return sl.getBaseEntry().WithField(key, value)
}

// WithFields adds multiple fields to the logger
func (sl *StructuredLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	return sl.getBaseEntry().WithFields(fields)
}

// WithError adds error information with stack trace if available
func (sl *StructuredLogger) WithError(err error) *logrus.Entry {
	entry := sl.getBaseEntry().WithError(err)
	
	// Add caller information for errors
	if pc, file, line, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			entry = entry.WithFields(logrus.Fields{
				"caller_function": fn.Name(),
				"caller_file":     file,
				"caller_line":     line,
			})
		}
	}
	
	return entry
}

// getBaseEntry creates a base log entry with context, trace, and component information
func (sl *StructuredLogger) getBaseEntry() *logrus.Entry {
	entry := sl.logger.WithField(ServiceNameKey, "th-payment-processor")
	
	if sl.component != "" {
		entry = entry.WithField(ComponentKey, sl.component)
	}
	
	// Add trace context if available
	if sl.ctx != nil {
		if span := trace.SpanFromContext(sl.ctx); span.SpanContext().IsValid() {
			entry = entry.WithFields(logrus.Fields{
				TraceIDKey: span.SpanContext().TraceID().String(),
				SpanIDKey:  span.SpanContext().SpanID().String(),
			})
		}
		
		// Add correlation ID from context
		if correlationID := sl.ctx.Value("correlation_id"); correlationID != nil {
			if corrID, ok := correlationID.(string); ok && corrID != "" {
				entry = entry.WithField(CorrelationIDKey, corrID)
			}
		}
		
		// Add request ID from context if available
		if requestID := sl.ctx.Value("request_id"); requestID != nil {
			if reqID, ok := requestID.(string); ok && reqID != "" {
				entry = entry.WithField("request_id", reqID)
			}
		}
	}
	
	return entry
}

// Info logs an info message
func (sl *StructuredLogger) Info(message string) {
	sl.getBaseEntry().Info(message)
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(message string) {
	sl.getBaseEntry().Debug(message)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(message string) {
	sl.getBaseEntry().Warn(message)
}

// Error logs an error message
func (sl *StructuredLogger) Error(message string) {
	sl.getBaseEntry().Error(message)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(message string) {
	sl.getBaseEntry().Fatal(message)
}

// LogPaymentStart logs the start of payment processing
func (sl *StructuredLogger) LogPaymentStart(correlationID string, amount float64) {
	sl.WithPaymentFields(PaymentFields{
		CorrelationID: correlationID,
		Amount:        amount,
		Status:        "started",
	}).WithField(OperationKey, "payment_start").Info("Payment processing started")
}

// LogPaymentSuccess logs successful payment processing
func (sl *StructuredLogger) LogPaymentSuccess(correlationID string, amount float64, processor string, latencyMs int64) {
	sl.WithPaymentFields(PaymentFields{
		CorrelationID: correlationID,
		Amount:        amount,
		Processor:     processor,
		Status:        "success",
	}).WithField(LatencyKey, latencyMs).WithField(OperationKey, "payment_success").Info("Payment processed successfully")
}

// LogPaymentFailure logs failed payment processing
func (sl *StructuredLogger) LogPaymentFailure(correlationID string, amount float64, processor string, err error, latencyMs int64) {
	sl.WithPaymentFields(PaymentFields{
		CorrelationID: correlationID,
		Amount:        amount,
		Processor:     processor,
		Status:        "failed",
	}).WithField(LatencyKey, latencyMs).WithError(err).WithField(OperationKey, "payment_failure").Error("Payment processing failed")
}

// LogHTTPRequest logs HTTP request information
func (sl *StructuredLogger) LogHTTPRequest(method, url, remoteAddr string) {
	sl.WithHTTPFields(HTTPFields{
		Method:     method,
		URL:        url,
		RemoteAddr: remoteAddr,
	}).WithField(OperationKey, "http_request").Info("HTTP request received")
}

// LogHTTPResponse logs HTTP response information
func (sl *StructuredLogger) LogHTTPResponse(method, url string, status int, latencyMs int64) {
	sl.WithHTTPFields(HTTPFields{
		Method: method,
		URL:    url,
		Status: status,
	}).WithField(LatencyKey, latencyMs).WithField(OperationKey, "http_response").Info("HTTP response sent")
}

// LogHealthCheck logs health check information
func (sl *StructuredLogger) LogHealthCheck(processor string, isHealthy bool, responseTime int64) {
	entry := sl.getBaseEntry().WithFields(logrus.Fields{
		ProcessorKey: processor,
		"healthy":    isHealthy,
		"response_time_ms": responseTime,
		OperationKey: "health_check",
	})
	
	if isHealthy {
		entry.Info("Health check passed")
	} else {
		entry.Warn("Health check failed")
	}
}

// GetUnderlyingLogger returns the underlying logrus logger for compatibility
func (sl *StructuredLogger) GetUnderlyingLogger() *logrus.Logger {
	return sl.logger
}