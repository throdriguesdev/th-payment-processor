# Observability Improvements

## Overview

This document describes the enhanced structured logging and distributed tracing improvements implemented in the payment processor application.

## Features Implemented

### 1. Enhanced Structured Logging

#### New Components
- **Structured Logger** (`internal/logging/structured_logger.go`): Comprehensive logging utility with payment-specific contexts
- **Logging Configuration** (`internal/logging/config.go`): Environment-based configuration management
- **Enhanced Middleware** (`internal/middleware/correlation_middleware.go`): Improved context propagation

#### Key Features
- **JSON Format**: All logs are in structured JSON format for easy parsing by log aggregation systems
- **Correlation IDs**: Every request gets a unique correlation ID that flows through all log entries
- **Request IDs**: Additional request-specific identifiers for fine-grained tracing
- **Trace Context**: Automatic inclusion of OpenTelemetry trace and span IDs in log entries
- **Payment-Specific Fields**: Dedicated logging methods for payment operations
- **HTTP Request/Response Logging**: Comprehensive HTTP interaction logging
- **Health Check Logging**: Detailed logging for service health monitoring

#### Log Structure Example
```json
{
  "@timestamp": "2024-01-15T14:30:45.123Z",
  "level": "info",
  "message": "Payment processing started",
  "service_name": "th-payment-processor",
  "service_version": "1.0.0",
  "component": "payment_service",
  "operation": "payment_start",
  "correlation_id": "123e4567-e89b-12d3-a456-426614174000",
  "request_id": "987fcdeb-51a2-43d7-9876-543210987654",
  "trace_id": "1234567890abcdef1234567890abcdef",
  "span_id": "abcdef0123456789",
  "payment_id": "pay_001",
  "amount": 100.50,
  "processor": "default",
  "status": "started"
}
```

### 2. Enhanced Distributed Tracing

#### Improvements
- **Context Propagation**: Improved trace context propagation across service boundaries
- **Enhanced Span Attributes**: More detailed span attributes for better observability
- **Correlation ID Integration**: Correlation IDs are automatically added to trace spans
- **HTTP Header Propagation**: Correlation IDs are propagated via HTTP headers to external services

#### Trace Attributes Added
- `http.correlation_id`: Request correlation ID
- `http.request_id`: Request-specific ID
- `payment.correlation_id`: Payment correlation ID
- `payment.amount`: Payment amount
- `service.operation`: Current operation being performed
- `http.response.duration_ms`: Request duration in milliseconds
- `http.response.size_bytes`: Response size

### 3. Environment Configuration

#### Environment Variables
- `LOG_LEVEL`: Log level (debug, info, warn, error, fatal) - default: info
- `LOG_FORMAT`: Log format (json, text) - default: json
- `LOG_OUTPUT`: Log output (stdout, stderr) - default: stdout
- `LOG_ENABLE_FILE`: Enable file logging (true/false) - default: false
- `LOG_FILE_PATH`: Log file path - default: logs/app.log

#### Usage Examples
```bash
# Debug logging with text format
export LOG_LEVEL=debug
export LOG_FORMAT=text

# Production logging with JSON format to stderr
export LOG_LEVEL=info
export LOG_FORMAT=json
export LOG_OUTPUT=stderr
```

## Integration with Monitoring Stack

### Grafana Integration
- All structured logs can be queried using correlation_id, trace_id, or payment-specific fields
- Enhanced log aggregation capabilities through structured fields
- Better correlation between logs, metrics, and traces

### Loki Integration
- JSON logs are automatically parsed and indexed by Loki
- Easy filtering by service_name, component, operation, or any custom field
- Support for complex queries using structured log fields

### Jaeger/Tempo Integration
- Enhanced trace context with correlation IDs
- Better span relationships and attributes
- Improved service graph visibility

## Usage Examples

### In Application Code

#### Using Structured Logger
```go
import "th_payment_processor/internal/logging"

// Create structured logger with context
logger := logging.NewStructuredLogger("payment_service").WithContext(ctx)

// Log with payment-specific fields
logger.LogPaymentStart(correlationID, amount)
logger.LogPaymentSuccess(correlationID, amount, processor, latencyMs)
logger.LogPaymentFailure(correlationID, amount, processor, err, latencyMs)

// Log HTTP operations
logger.LogHTTPRequest(method, url, remoteAddr)
logger.LogHTTPResponse(method, url, status, latencyMs)

// Log health checks
logger.LogHealthCheck(processor, isHealthy, responseTime)
```

#### Using Context Propagation
```go
import "th_payment_processor/internal/middleware"

// Get correlation ID from context
correlationID := middleware.GetCorrelationID(ctx)

// Get structured logger from context
logger := middleware.GetStructuredLogger(ctx)

// Propagate context to child operations
childCtx := middleware.PropagateContext(ctx)
```

### Query Examples in Grafana/Loki

#### Find all payment failures
```logql
{service_name="th-payment-processor"} | json | status="failed"
```

#### Find logs for a specific correlation ID
```logql
{service_name="th-payment-processor"} | json | correlation_id="123e4567-e89b-12d3-a456-426614174000"
```

#### Find high-latency operations
```logql
{service_name="th-payment-processor"} | json | latency_ms > 1000
```

#### Find payment processor errors
```logql
{service_name="th-payment-processor"} | json | component="payment_processor" | level="error"
```

## Benefits

1. **Better Debugging**: Correlation IDs allow tracking requests across the entire system
2. **Enhanced Monitoring**: Structured logs enable powerful queries and alerting
3. **Improved Performance Analysis**: Latency tracking in all operations
4. **Better Error Tracking**: Detailed error context and stack traces
5. **Compliance**: Structured logging supports audit trails and compliance requirements
6. **Operational Insights**: Better understanding of system behavior and bottlenecks

## Next Steps

1. **Business Metrics**: Implement payment-specific metrics and KPIs
2. **Custom Dashboards**: Create Grafana dashboards using structured log data
3. **Alerting**: Set up alerts based on structured log patterns
4. **Log Aggregation**: Consider implementing log sampling for high-volume environments
5. **Performance Monitoring**: Add APM-style performance tracking using trace data

## Monitoring Dashboard Queries

The structured logging enables powerful dashboard queries:

1. **Payment Success Rate**: Query successful vs failed payments by processor
2. **Latency Distribution**: Analyze payment processing latency percentiles
3. **Error Rate Trends**: Monitor error rates over time by component
4. **Health Check Status**: Track processor health status and response times
5. **Request Volume**: Monitor request volume by endpoint and status

This enhanced observability foundation provides deep insights into system behavior and enables proactive monitoring and debugging capabilities.