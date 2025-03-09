# Game Ad Service

A highly scalable service for handling game advertisements with priority queuing and distributed analysis.

## Architecture Overview

The Game Ad Service is designed as a distributed system with the following key components:

1. **API Layer**: RESTful endpoints for ad submission and status retrieval
2. **Queue Management System**: Priority-based queuing with horizontal scaling
3. **Worker Pool**: Distributed workers for concurrent ad processing
4. **Storage Layer**: Flexible storage with caching capabilities
5. **Service Discovery**: For dynamic worker registration and health monitoring
6. **Monitoring & Observability**: Comprehensive metrics, logging, and tracing

## System Design Diagram

```
┌─────────────────┐     ┌───────────────────┐     ┌─────────────────────┐
│                 │     │                   │     │                     │
│  Load Balancer  │────▶│  API Service      │────▶│  Service Registry   │
│                 │     │  (Multiple Nodes) │     │                     │
└─────────────────┘     └───────────────────┘     └─────────────────────┘
                                 │                           │
                                 ▼                           │
                         ┌───────────────┐                   │
                         │               │                   │
                         │  Message Bus  │                   │
                         │               │                   │
                         └───────────────┘                   │
                                 │                           │
                                 ▼                           ▼
┌────────────────┐      ┌───────────────┐     ┌─────────────────────────┐
│                │      │               │     │                         │
│  Cache Layer   │◀────▶│  Worker Pool  │────▶│  Worker Nodes           │
│  (Redis)       │      │  Manager      │     │  (Auto-scaling Workers) │
│                │      │               │     │                         │
└────────────────┘      └───────────────┘     └─────────────────────────┘
        │                       │                          │
        │                       │                          │
        ▼                       ▼                          ▼
┌────────────────────────────────────────────────────────────────────────┐
│                                                                        │
│                            Storage Layer                               │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

## Distributed Systems Architecture

### API Layer

- **Multiple API Nodes**: Stateless API servers that can be scaled horizontally
- **Load Balancing**: Use of Kubernetes or a dedicated load balancer to distribute traffic
- **Rate Limiting**: Distributed rate limiting using Redis or a similar mechanism
- **API Gateway**: Optional gateway for additional features like authentication, throttling

### Queue Management

- **Distributed Queue**: Using Redis, NATS, or Apache Kafka for reliable message passing
- **Priority Support**: Implementation of priority queues with aging mechanism
- **Queue Partitioning**: Sharded queues for high throughput
- **Dead Letter Queue**: For handling failed processing attempts

### Worker System

- **Auto-scaling Workers**: Worker pods that can scale based on queue depth
- **Work Stealing**: Algorithm for efficient task distribution
- **Periodic Rebalancing**: Dynamic rebalancing of workloads
- **Supervised Execution**: Monitoring for timeouts and failures

### Storage Strategy

- **Polyglot Persistence**: Different storage solutions based on data access patterns
- **Caching Layer**: Redis for caching frequently accessed data
- **Data Partitioning**: Horizontal sharding for high-volume data
- **Optional Persistence**: Support for PostgreSQL/MongoDB for durable storage

### Service Discovery & Orchestration

- **Service Registry**: For dynamic discovery of workers and API nodes
- **Health Monitoring**: Continuous health checks with automatic remediation
- **Configuration Management**: Centralized configuration with real-time updates
- **Circuit Breaking**: Protection against cascading failures

### Observability Stack

- **Distributed Tracing**: Using OpenTelemetry for end-to-end tracing
- **Metrics Collection**: Prometheus for real-time metrics
- **Centralized Logging**: ELK/EFK stack for log aggregation
- **Alerting System**: Proactive monitoring with defined SLOs

## Scaling Strategies

### Horizontal Scaling

- API layer scales based on incoming request rate
- Worker pool scales based on queue depth and processing latency
- Storage layer partitioned by ad ID for even distribution

### Data Consistency

- Eventual consistency model for status updates
- Strong consistency for initial ad submission
- Optimistic concurrency control for updates

### Performance Optimization

- Batch processing for analytics when possible
- Connection pooling for database interactions
- Efficient serialization formats (Protocol Buffers/MessagePack)
- Caching of frequently accessed ads and analysis results

## Fault Tolerance & Resilience

### Failure Modes

- Graceful degradation under load
- Retry mechanisms with exponential backoff
- Automatic failover for critical components
- Circuit breakers to prevent cascading failures

### Data Redundancy

- Replicated storage for high availability
- Periodic snapshots of in-memory state
- Write-ahead logging for recovery

## Project Structure

```
game-ad-service/
├── cmd/
│   ├── api/
│   │   └── main.go               # API server entry point
│   └── worker/
│       └── main.go               # Worker entry point
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── ad_handler.go     # API endpoint handlers
│   │   │   └── response.go       # Response utilities
│   │   ├── middleware/
│   │   │   ├── logging.go        # Logging middleware
│   │   │   ├── ratelimit.go      # Rate limiting
│   │   │   └── recovery.go       # Panic recovery
│   │   └── router.go             # Router setup
│   ├── domain/
│   │   ├── models/
│   │   │   └── ad.go             # Core domain models
│   │   └── events/
│   │       └── events.go         # Domain events
│   ├── processor/
│   │   ├── analyzer.go           # Analysis logic
│   │   ├── queue.go              # Priority queue
│   │   ├── worker.go             # Worker implementation
│   │   └── dispatcher.go         # Task dispatcher
│   ├── storage/
│   │   ├── cache/
│   │   │   └── redis.go          # Redis cache
│   │   ├── memory/
│   │   │   └── memory_store.go   # In-memory storage
│   │   └── repository.go         # Storage interface
│   ├── service/
│   │   ├── ad_service.go         # Business logic
│   │   └── metrics.go            # Service metrics
│   └── config/
│       └── config.go             # Configuration
├── pkg/
│   ├── logger/
│   │   └── logger.go             # Structured logging
│   ├── messaging/
│   │   ├── kafka/
│   │   │   └── producer.go       # Kafka implementation
│   │   ├── nats/
│   │   │   └── jetstream.go      # NATS implementation
│   │   └── messaging.go          # Messaging interface
│   ├── discovery/
│   │   └── registry.go           # Service discovery
│   ├── telemetry/
│   │   ├── metrics.go            # Prometheus metrics
│   │   └── tracing.go            # Distributed tracing
│   └── utils/
│       ├── id_generator.go       # ID generation
│       └── retry.go              # Retry mechanisms
├── deployments/
│   ├── kubernetes/
│   │   ├── api-deployment.yaml
│   │   ├── worker-deployment.yaml
│   │   └── redis-statefulset.yaml
│   └── docker/
│       ├── api.Dockerfile
│       └── worker.Dockerfile
├── docker-compose.yml            # Local development setup
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- kubectl (for Kubernetes deployment)

### Running Locally

```bash
# Start all services
docker-compose up -d

# Submit a sample ad
curl -X POST http://localhost:8080/ads -d '{
  "title": "Dragon Kingdom: Rise to Power",
  "description": "Build your kingdom, train your dragons, and conquer new territories!",
  "genre": "Strategy",
  "targetAudience": ["18-34", "Strategy Gamers"],
  "visualElements": ["Dragons", "Castle", "Battle Scenes"],
  "callToAction": "Download Now & Claim 1000 Free Gems!",
  "duration": 30,
  "priority": 2
}'
```

### Deployment

```bash
// docker-compose
change config.yaml.example -> config.yaml (can edit value to match your env)

// shell
docker-compose up
```

## Performance Characteristics

- **Throughput**: Can handle 1000+ ad submissions per second
- **Latency**: P95 processing time under 10 seconds
- **Scalability**: Linear scaling with added worker nodes
- **Reliability**: 99.9% availability target

## Future Enhancements

1. **Machine Learning Pipeline**: Replace mock analysis with actual ML models
2. **Geographic Distribution**: Multi-region deployment for global reach
3. **Enhanced Analytics**: Real-time dashboard for ad performance
4. **A/B Testing Framework**: Compare different ad variations
5. **Webhook Notifications**: Push updates to clients upon completion
