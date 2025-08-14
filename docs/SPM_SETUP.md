# Service Performance Monitoring (SPM) Setup Guide

## Overview

Your payment processor now has complete **Service Performance Monitoring (SPM)** with Jaeger-Prometheus-Grafana integration. This provides unified observability combining distributed tracing, metrics, and correlation.

## Architecture

### Complete Observability Stack
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │────│    Jaeger       │────│    Grafana      │
│                 │    │ (Traces)        │    │ (Visualization) │
│ - HTTP Requests │    │                 │    │                 │
│ - Payment Flow  │    │ - Span Data     │    │ - SPM Dashboard │
│ - Error Handling│    │ - Trace Context │    │ - Trace Browser │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐               │
         └──────────────│   Prometheus    │───────────────┘
                        │   (Metrics)     │
                        │                 │
                        │ - RED Metrics   │
                        │ - Exemplars     │
                        │ - Jaeger Stats  │
                        └─────────────────┘
```

## Key Features

### 1. **Distributed Tracing (Jaeger)**
- **End-to-end request tracking** across all payment processing components
- **Service dependency mapping** showing component interactions  
- **Performance bottleneck identification** at the span level
- **Error correlation** linking failures to specific trace segments

### 2. **Metrics Collection (Prometheus)**
- **RED Metrics**: Rate, Errors, Duration for all HTTP endpoints
- **Payment-specific metrics**: Amount distributions, processor performance
- **Trace exemplars**: Links metrics data points to specific traces
- **Jaeger ingestion metrics**: Trace collection performance statistics

### 3. **Unified Visualization (Grafana)**
- **SPM Dashboard**: Combined traces and metrics view
- **Exemplar correlation**: Click metric points to view related traces
- **Service topology**: Visual service dependency graph
- **Recent trace browser**: Latest payment processing traces

## Access Points

### Web Interfaces
- **SPM Dashboard**: http://localhost:3000/d/spm-dashboard (admin/admin123)
- **RED Metrics Dashboard**: http://localhost:3000/d/payment-processor-red  
- **Jaeger Trace UI**: http://localhost:16686
- **Prometheus Metrics**: http://localhost:9090

### API Endpoints
- **Payment Service**: http://localhost:9999
- **Metrics Endpoints**: http://localhost:2113-2116/metrics
- **Jaeger Admin**: http://localhost:14269/metrics

## Data Sources Configuration

### Prometheus → Grafana
```yaml
datasources:
  - name: Prometheus
    type: prometheus
    url: http://prometheus:9090
    uid: prometheus
    jsonData:
      exemplarTraceIdDestinations:
        - name: traceID
          datasourceUid: jaeger
```

### Jaeger → Grafana  
```yaml
datasources:
  - name: Jaeger
    type: jaeger
    url: http://jaeger:16686
    uid: jaeger
    jsonData:
      tracesToMetrics:
        datasourceUid: prometheus
        queries:
          - name: "Request rate"
            query: "sum(rate(http_requests_total{service_name=\"$service\"}[5m]))"
```

## SPM Dashboard Panels

### 1. **Request Rate with Trace Exemplars**
- **Metric**: `rate(http_requests_total[5m])`
- **Exemplars**: Click points to see related traces
- **Purpose**: Identify traffic patterns and trace specific requests

### 2. **Response Time with Trace Exemplars** 
- **Metrics**: 95th and 50th percentile latencies
- **Exemplars**: Links slow requests to trace analysis
- **Purpose**: Performance monitoring with root cause analysis

### 3. **Service Dependency Map**
- **Source**: Jaeger trace topology
- **Purpose**: Visualize service interactions and dependencies
- **Features**: Interactive node exploration

### 4. **Payment Processing Metrics**
- **Metrics**: Payment amounts by processor and status
- **Exemplars**: Links payment data to processing traces
- **Purpose**: Business metrics with operational context

### 5. **Jaeger Ingestion Statistics**
- **Metrics**: Trace/span ingestion rates
- **Purpose**: Monitor trace collection health
- **Alerts**: Detect trace collection issues

### 6. **Recent Payment Traces**
- **Source**: Jaeger trace search
- **Purpose**: Browse latest payment processing traces  
- **Features**: Direct trace drill-down

## Correlation Workflows

### Metrics → Traces
1. **Spot anomaly** in metrics dashboard (e.g., latency spike)
2. **Click exemplar dot** on the metric graph
3. **Jump to trace view** showing the specific request
4. **Analyze trace spans** to find root cause
5. **Correlate with other traces** from same time period

### Traces → Metrics
1. **Find problematic trace** in Jaeger UI
2. **Note service and operation** details
3. **Switch to metrics view** for that service
4. **Analyze trends** around the trace timestamp
5. **Identify patterns** across multiple traces

## Key Metrics for SPM

### RED Metrics
```promql
# Rate - Requests per second
rate(http_requests_total[5m])

# Errors - Error percentage  
rate(http_errors_total[5m]) / rate(http_requests_total[5m]) * 100

# Duration - Response time percentiles
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Payment Business Metrics
```promql
# Payment volume by processor
sum(rate(payment_amount_dollars_count[5m])) by (processor, status)

# Payment amount distribution
histogram_quantile(0.95, rate(payment_amount_dollars_bucket[5m]))
```

### Trace Collection Metrics
```promql  
# Trace ingestion rate
rate(jaeger_traces_received_total[5m])

# Span ingestion rate
rate(jaeger_spans_received_total[5m])
```

## Troubleshooting SPM

### Missing Exemplars
- **Symptom**: No dots on metrics graphs
- **Check**: Trace context in HTTP requests
- **Fix**: Verify OpenTelemetry middleware is active

### Broken Trace Links
- **Symptom**: Clicking exemplars doesn't open traces
- **Check**: Jaeger datasource configuration in Grafana
- **Fix**: Verify datasource UIDs match in configuration

### Missing Traces
- **Symptom**: Empty service list in Jaeger
- **Check**: Application trace export configuration
- **Fix**: Verify Jaeger endpoint and network connectivity

### Service Map Empty
- **Symptom**: No services shown in dependency graph
- **Check**: Trace span relationships and service names
- **Fix**: Ensure proper span hierarchy in application code

## Advanced Features

### Custom Trace Queries
```promql
# Find traces with high payment amounts
jaeger_query{service="th-payment-processor", operation="POST /payments", amount=">100"}
```

### Metric Correlation Queries  
```promql
# Correlate error rates with trace volumes
(
  rate(http_errors_total[5m]) and 
  rate(jaeger_spans_received_total[5m]) > 10
)
```

### Service Performance Alerts
```promql
# Alert on high error rate with trace context
(
  rate(http_errors_total[5m]) / rate(http_requests_total[5m]) > 0.05
) and (
  rate(jaeger_traces_received_total[5m]) > 0
)
```

## Best Practices

### 1. **Trace Sampling**
- Use probabilistic sampling for high-volume services
- Ensure error traces are always captured
- Configure sampling based on service criticality

### 2. **Exemplar Strategy**
- Link high-latency requests to traces
- Correlate error metrics with failure traces
- Use exemplars for business-critical operations

### 3. **Dashboard Organization**  
- Group related metrics and traces
- Use consistent time ranges across panels
- Provide clear drill-down paths

### 4. **Alert Integration**
- Create alerts that reference both metrics and traces  
- Include trace links in alert notifications
- Use trace context to reduce false positives

## Maintenance

### Regular Tasks
- **Monitor trace storage usage** in Jaeger
- **Validate exemplar links** in Grafana dashboards
- **Review service topology** for accuracy
- **Update dashboard queries** as services evolve

### Performance Optimization
- **Adjust trace sampling rates** based on volume
- **Optimize metric collection intervals**
- **Archive old trace data** according to retention policy
- **Monitor SPM system resource usage**

## Future Enhancements

### Planned Features
- **Log correlation** with traces and metrics
- **Business KPI dashboards** with trace context
- **Automated anomaly detection** using ML
- **Cross-cluster trace correlation** for distributed deployments
- **Custom trace analysis** tools and workflows

---

## Quick Start Commands

```bash
# Start SPM stack
docker compose up -d

# Validate SPM setup
./validate-spm.sh

# Generate test traces
./generate-test-traffic.sh

# Access SPM dashboard
open http://localhost:3000/d/spm-dashboard
```

Your Service Performance Monitoring setup provides complete observability with seamless correlation between traces, metrics, and business outcomes!