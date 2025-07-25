# Configuration Directory

This directory contains configuration files for different environments.

## Environment Variables

The application uses the following environment variables:

### Server Configuration
- `SERVER_PORT` - Port for the HTTP server (default: 8080)

### Payment Processor URLs
- `DEFAULT_PROCESSOR_URL` - Default processor endpoint (default: http://payment-processor-default:8080)
- `FALLBACK_PROCESSOR_URL` - Fallback processor endpoint (default: http://payment-processor-fallback:8080)

### Health Monitoring
- `HEALTH_CHECK_INTERVAL` - Health check frequency (default: 5s)
- `REQUEST_TIMEOUT` - HTTP request timeout (default: 10s)

### Observability
- `JAEGER_ENDPOINT` - Jaeger tracing endpoint (default: http://jaeger:14268/api/traces)

## Configuration Files

Future configuration files can be added here for different environments:
- `development.yaml` - Development environment config
- `production.yaml` - Production environment config
- `testing.yaml` - Testing environment config