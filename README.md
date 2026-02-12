<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Gin-1.9+-008EC4?style=for-the-badge&logo=gin" alt="Gin Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/coverage-90%25+-brightgreen?style=for-the-badge" alt="Coverage">
</p>

<h1 align="center">🐙 Octopus - Short Link Service</h1>

<p align="center">
  <b>A high-performance, scalable short link service built with Go best practices</b>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#api-documentation">API</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#deployment">Deployment</a> •
  <a href="#architecture">Architecture</a>
</p>

---

## Features

- **High Performance** - Built on Gin framework with Redis caching
- **Collision Detection** - Redis Bloom Filter for fast duplicate checking
- **Flexible Encoding** - Base32 encoding with 4-6 character short codes
- **302 Redirect** - Standard HTTP redirect with query parameter merging
- **Analytics** - Real-time PV/UV tracking and source analysis
- **Async Processing** - RocketMQ for high-throughput access log processing
- **Production Ready** - Docker, Kubernetes, and docker-compose support
- **Well Tested** - Comprehensive unit tests with 90%+ coverage

## Quick Start

### Prerequisites

- Go 1.21+
- Redis 7.0+
- MySQL 8.0+
- (Optional) RocketMQ 5.0+

### Run with Docker Compose

```bash
# Clone the repository
git clone https://github.com/MariexAos/octopus.git
cd octopus

# Start all services
docker-compose up -d

# Check service status
docker-compose ps
```

### Run Locally

```bash
# Install dependencies
go mod download

# Start infrastructure (Redis, MySQL)
docker-compose up -d redis mysql

# Run the service
make run

# Or with hot reload
make dev
```

The service will be available at `http://localhost:8080`

### API Usage

**Generate Short Link**

```bash
curl -X POST http://localhost:8080/api/v1/shortlink/generate \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/very/long/url",
    "params": {
      "utm_source": "newsletter",
      "campaign": "spring2024"
    }
  }'
```

Response:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "short_link": "http://localhost:8080/AbCd",
    "short_code": "AbCd",
    "original_url": "https://example.com/very/long/url"
  }
}
```

**Access Short Link**

```bash
curl -I http://localhost:8080/AbCd

# Response: HTTP/1.1 302 Found
# Location: https://example.com/very/long/url?utm_source=newsletter&campaign=spring2024
```

**Get Analytics**

```bash
curl http://localhost:8080/api/v1/analytics/AbCd
```

Response:
```json
{
  "code": 0,
  "data": {
    "short_code": "AbCd",
    "pv": 10000,
    "uv": 3500,
    "top_sources": [
      {"source": "google.com", "count": 4000},
      {"source": "twitter.com", "count": 2500}
    ]
  }
}
```

## API Documentation

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/shortlink/generate` | Generate a new short link |
| GET | `/{shortCode}` | Redirect to original URL |
| GET | `/api/v1/analytics/{shortCode}` | Get analytics data |
| GET | `/swagger/index.html` | Swagger UI |

Access Swagger UI at: `http://localhost:8080/swagger/index.html`

## Configuration

Configuration file: `configs/config.yaml`

```yaml
server:
  port: 8080
  mode: release  # debug, release, test

database:
  mysql:
    dsn: "user:password@tcp(localhost:3306)/shortlink?charset=utf8mb4&parseTime=True"
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0

bloom:
  capacity: 1000000000  # 1 billion
  error_rate: 0.01      # 1%

rocketmq:
  nameserver: "localhost:9876"
  topic: "access_log"
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | Server port | `8080` |
| `MYSQL_DSN` | MySQL connection string | - |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `ROCKETMQ_NAMESERVER` | RocketMQ name server | - |

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│   Gin API   │────▶│   Service   │
└─────────────┘     └─────────────┘     └─────────────┘
                           │                    │
                           ▼                    ▼
                    ┌─────────────┐     ┌─────────────┐
                    │ Middleware  │     │  Repository │
                    └─────────────┘     └─────────────┘
                                               │
                              ┌────────────────┼────────────────┐
                              ▼                ▼                ▼
                       ┌───────────┐    ┌───────────┐    ┌───────────┐
                       │   Redis   │    │   MySQL   │    │ RocketMQ  │
                       └───────────┘    └───────────┘    └───────────┘
```

### Project Structure

```
octopus/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── encoder/         # Base32 encoder
│   ├── handler/         # HTTP handlers
│   ├── model/           # Data models
│   ├── mq/              # RocketMQ producer/consumer
│   ├── repository/      # Data access layer
│   ├── service/         # Business logic layer
│   └── mocks/           # Mock implementations for testing
├── pkg/
│   ├── middleware/      # HTTP middleware
│   └── util/            # Utility functions
├── configs/             # Configuration files
├── deployments/         # Docker & K8s manifests
├── scripts/             # Database migrations
├── docs/                # Documentation
├── Makefile             # Build automation
└── docker-compose.yaml  # Local development stack
```

## Deployment

### Docker

```bash
# Build image
docker build -f deployments/docker/Dockerfile -t octopus:latest .

# Run container
docker run -d \
  --name octopus \
  -p 8080:8080 \
  -e MYSQL_DSN="user:pass@tcp(mysql:3306)/shortlink" \
  -e REDIS_ADDR="redis:6379" \
  octopus:latest
```

### Kubernetes

```bash
# Apply manifests
kubectl apply -f deployments/k8s/
```

## Development

### Makefile Commands

```bash
make help          # Show all available commands
make build         # Build the binary
make run           # Run the service
make test          # Run all tests
make test-coverage # Run tests with coverage report
make lint          # Run linter
make docker-build  # Build Docker image
make swagger       # Generate Swagger docs
make migrate-up    # Run database migrations
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# View coverage report
open coverage.html
```

## Roadmap

- [ ] Custom short code support
- [ ] Link expiration management
- [ ] QR code generation
- [ ] Batch import/export
- [ ] Admin dashboard
- [ ] Rate limiting
- [ ] GraphQL API

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Gin](https://github.com/gin-gonic/gin) - Web framework
- [GORM](https://gorm.io/) - ORM
- [go-redis](https://github.com/redis/go-redis) - Redis client
- [RocketMQ](https://rocketmq.apache.org/) - Message queue

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/MariexAos">MariexAos</a>
</p>
