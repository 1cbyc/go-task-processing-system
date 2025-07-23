# Go Task Processing System

Distributed task processing system built in Go. This system implements a worker pool pattern with advanced features for handling concurrent job processing, monitoring, and scalability.

## Features

- **Concurrent Job Processing**: Efficient worker pool with configurable concurrency
- **Job Priority System**: Four priority levels (Low, Normal, High, Urgent)
- **Retry Logic**: Automatic retry with configurable attempts and delays
- **Health Monitoring**: Comprehensive health checks and statistics
- **HTTP API**: RESTful API for job submission and system monitoring
- **Graceful Shutdown**: Proper shutdown handling with signal management
- **Structured Logging**: JSON-formatted logging for production use
- **Configuration Management**: Environment-based configuration with validation
- **Thread Safety**: All operations are thread-safe with proper synchronization

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

1. Clone the repository:
```bash
git clone https://github.com/1cbyc/go-task-processing-system.git
cd go-task-processing-system
```

2. Install dependencies:
```bash
go mod tidy
```

3. Run the application:
```bash
go run main.go
```

The system will start on `http://localhost:8080` by default.

### Configuration

The system can be configured using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `SERVER_HOST` | 0.0.0.0 | HTTP server host |
| `WORKER_NUM_WORKERS` | 10 | Number of worker goroutines |
| `WORKER_MAX_CONCURRENCY` | 100 | Maximum concurrent jobs |
| `WORKER_JOB_TIMEOUT` | 5m | Job processing timeout |
| `WORKER_RETRY_ATTEMPTS` | 3 | Number of retry attempts |
| `LOG_LEVEL` | info | Logging level |

Example:
```bash
export WORKER_NUM_WORKERS=20
export WORKER_MAX_CONCURRENCY=200
go run main.go
```

## API Reference

### Health Check

Check system health:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### System Statistics

Get system performance metrics:

```bash
curl http://localhost:8080/stats
```

Response:
```json
{
  "jobs_submitted": 150,
  "jobs_completed": 145,
  "jobs_failed": 5,
  "workers_active": 10,
  "queue_size": 0,
  "uptime": "1h30m45s"
}
```

### Submit Job

Submit a new job for processing:

```bash
curl -X POST "http://localhost:8080/jobs?type=email&priority=high"
```

Response:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "submitted"
}
```

Parameters:
- `type`: Job type (default: "default")
- `priority`: Job priority - "low", "normal", "high", "urgent" (default: "normal")

### Worker Information

Get information about active workers:

```bash
curl http://localhost:8080/workers
```

Response:
```json
{
  "workers": 10
}
```

## Job Types and Priorities

### Job Types

The system supports different job types for categorization:

- `email`: Email processing jobs
- `report`: Report generation jobs
- `backup`: Data backup jobs
- `sync`: Data synchronization jobs
- `cleanup`: Cleanup and maintenance jobs
- `default`: Generic jobs

### Priority Levels

Jobs can have different priority levels:

- `PriorityLow` (1): Low priority jobs
- `PriorityNormal` (2): Normal priority jobs (default)
- `PriorityHigh` (3): High priority jobs
- `PriorityUrgent` (4): Urgent priority jobs

## Architecture

The system consists of several key components:

### Dispatcher
- Manages the worker pool
- Handles job distribution
- Collects statistics and health information
- Provides HTTP API endpoints

### Workers
- Process jobs concurrently
- Handle job lifecycle (start, complete, fail)
- Track individual statistics
- Support timeout and retry logic

### Jobs
- Represent individual tasks
- Have unique IDs, types, and priorities
- Track status and metadata
- Support result storage

## Development

### Project Structure

```
.
├── config/          # Configuration management
├── dispatcher/      # Job dispatcher and worker pool
├── job/            # Job definition and lifecycle
├── worker/         # Worker implementation
├── docs/           # Documentation
├── main.go         # Application entry point
└── go.mod          # Go module definition
```

### Building

Build the application:

```bash
go build -o task-processor main.go
```

### Testing

Run tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

### Running in Production

For production deployment:

1. Build the binary:
```bash
go build -o task-processor main.go
```

2. Set environment variables:
```bash
export SERVER_PORT=8080
export WORKER_NUM_WORKERS=20
export WORKER_MAX_CONCURRENCY=200
export LOG_LEVEL=info
```

3. Run the application:
```bash
./task-processor
```

## Monitoring

### Health Checks

The system provides health check endpoints for monitoring:

- `/health`: Basic health status
- `/stats`: Performance metrics
- `/workers`: Worker information

### Logging

The system uses structured JSON logging:

```json
{
  "level": "info",
  "msg": "Job completed successfully",
  "worker_id": "worker-1",
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "duration": "1.234s",
  "time": "2024-01-01T12:00:00Z"
}
```

### Metrics

Key metrics to monitor:

- Job submission rate
- Job completion rate
- Job failure rate
- Worker utilization
- Queue size
- Processing latency

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:

- Create an issue on GitHub
- Check the documentation in the `docs/` directory
- Review the technical explanation in `docs/explanation.md`

## Roadmap

See `docs/whats-next.md` for the development roadmap and future plans.
