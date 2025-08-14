
# TH Payment Processor

High-performance payment processing intermediary service with intelligent routing, automatic failover, and comprehensive testing suite. Originally designed for the Rinha de Backend competition.

## Quick Start

```bash
# Initialize and test everything
make init
make test
make clean
```

**Alternative using scripts:**
```bash
cd scripts
./init.sh
./test_payments.sh  
./cleanup.sh
```

## Key Features

- ✅ **Smart Payment Routing**: Default processor (1% fee) → Fallback processor (5% fee)
- ✅ **Load Balancing**: Two app instances behind Nginx with session affinity
- ✅ **Health Monitoring**: Background processor health checks every 5 seconds
- ✅ **Performance Target**: P99 < 11ms response time
- ✅ **Resource Efficient**: Exactly 1.5 CPU, 350MB memory total
- ✅ **Complete Testing**: Automated API, integration, and performance tests
- ✅ **Observability**: OpenTelemetry tracing and structured logging

## Service Endpoints

- **Backend API**: http://localhost:9999
- **Payment Summary**: http://localhost:9999/payments-summary
- **Default Processor**: http://localhost:8001 (1% fee)
- **Fallback Processor**: http://localhost:8002 (5% fee)

## Documentation

- **[Quick Start Guide](docs/QUICKSTART.md)** - Get running in 30 seconds
- **[API Endpoints](docs/ENDPOINTS.md)** - Complete endpoint reference
- **[Testing Guide](docs/TESTING.md)** - Comprehensive testing documentation
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment instructions
- **[Architecture Overview](docs/ARCHITECTURE.md)** - Technical architecture details
- **[Implementation Details](docs/IMPLEMENTATION.md)** - Code implementation specifics

## Architecture

```
Client → Nginx Load Balancer → App1/App2 → External Payment Processors
                                   ↓
                            In-Memory Storage
                                   ↓
                         OpenTelemetry → Jaeger
```

## Test Results Summary

✅ **POST /payments**: Smart routing with default→fallback logic  
✅ **GET /payments-summary**: Time-filtered aggregation  
✅ **Load Balancing**: Nginx distributes between 2 instances  
✅ **Health Monitoring**: Background checks every 5 seconds  
✅ **Performance**: P99 < 11ms target achieved  
✅ **Resource Limits**: 1.5 CPU, 350MB exactly as required  
✅ **Integration**: External payment processor network connectivity
