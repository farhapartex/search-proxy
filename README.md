# Search Proxy - Federated Search Engine (Go gRPC Service)

A high-performance gRPC service written in Go that concurrently fetches search results from multiple platforms (GitHub, StackOverflow, Reddit) using the Fan-out/Fan-in pattern.

## Overview

This service acts as the "Concurrency Engine" in our federated search architecture. It receives search queries from the Python/Django service via gRPC and returns normalized results within 500ms.

### Architecture

```
Python/Django Service
        ↓ (gRPC Request)
    Go Service (This Project)
        ↓ (Concurrent HTTP Calls via Goroutines)
[GitHub API] [StackOverflow API] [Reddit API]
        ↓ (Fan-in Results)
    Go Service (Normalization)
        ↓ (gRPC Response)
Python/Django Service
```

## Features

- **gRPC Server**: High-performance RPC communication with Python service
- **Concurrent Fetching**: Fan-out/Fan-in pattern using Goroutines
- **Context-Based Timeouts**: 500ms global, 400ms per-API
- **Result Normalization**: Unified data structure across platforms
- **Privacy Proxy**: Shields user IP from external APIs
- **Graceful Degradation**: Returns partial results if some APIs fail
- **Circuit Breaker**: Prevents cascading failures

### Folder Explanation

- **`cmd/server/`**: Application entry point. Keeps `main.go` separate from business logic.
- **`internal/`**: Private application code (cannot be imported by other projects).
  - `grpc/`: gRPC server setup and implementation
  - `handlers/`: Business logic (orchestrates fetchers)
  - `fetchers/`: External API clients (GitHub, SO, Reddit)
  - `models/`: Data structures (internal representation)
  - `config/`: Configuration management
- **`proto/`**: Protocol Buffer definitions and generated code.
- **`pkg/`**: Public libraries (reusable across projects).

## Getting Started

### Prerequisites

1. **Go 1.21+**
   ```bash
   go version
   ```

2. **Protocol Buffers Compiler**
   ```bash
   # Ubuntu/Debian
   sudo apt install -y protobuf-compiler

   # macOS
   brew install protobuf

   # Verify
   protoc --version
   ```

3. **Go gRPC Plugins**
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

   # Add to PATH
   export PATH="$PATH:$(go env GOPATH)/bin"
   ```

### Installation

1. **Clone and navigate to project**
   ```bash
   cd /home/ubuntu/Documents/goUpp/federated_search_engine/search-proxy
   ```

2. **Initialize Go module**
   ```bash
   go mod init github.com/yourusername/search-proxy
   go mod tidy
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env and add your API tokens
   ```

4. **Get API Tokens**

   - **GitHub**: https://github.com/settings/tokens
     - Permissions: `public_repo` (read-only)

   - **StackOverflow**: https://stackapps.com/apps/oauth/register
     - Type: Server-side app

   - **Reddit**: https://www.reddit.com/prefs/apps
     - Type: Script

5. **Generate gRPC code**
   ```bash
   make proto
   # Or manually:
   protoc --go_out=. --go-grpc_out=. proto/search.proto
   ```

6. **Install dependencies**
   ```bash
   go mod download
   ```

### Running the Service

```bash
# Development mode
make run

# Or directly
go run cmd/server/main.go
```

The gRPC server will start on `localhost:50051`.

You should see output like:
```
Loading configuration...
Configuration loaded successfully
Server will listen on port: 50051
🚀 gRPC server starting on :50051
Press Ctrl+C to stop
```

### Testing the gRPC Service

#### 1. Install grpcurl (gRPC testing tool)

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
export PATH="$PATH:$(go env GOPATH)/bin"

# Verify installation
which grpcurl
```

#### 2. Start the Server (Terminal 1)

```bash
cd /home/ubuntu/Documents/goUpp/federated_search_engine/search-proxy
export PATH="$PATH:$(go env GOPATH)/bin"
make run
```

#### 3. Test the Service (Terminal 2 - New Terminal)

**Test Health Check:**
```bash
grpcurl -plaintext localhost:50051 search.SearchService/HealthCheck
```

Expected output:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "1735689600"
}
```

**Test Simple Search:**
```bash
grpcurl -plaintext -d '{"query": "golang", "max_results": 5}' \
  localhost:50051 search.SearchService/FederatedSearch
```

**Test Search with Specific Platforms:**
```bash
grpcurl -plaintext -d '{
  "query": "React performance optimization",
  "max_results": 10,
  "platforms": ["github", "stackoverflow", "reddit"]
}' localhost:50051 search.SearchService/FederatedSearch
```

**Test GitHub Only:**
```bash
grpcurl -plaintext -d '{"query": "docker", "platforms": ["github"]}' \
  localhost:50051 search.SearchService/FederatedSearch
```

**List Available Services:**
```bash
# See all services
grpcurl -plaintext localhost:50051 list

# See methods in SearchService
grpcurl -plaintext localhost:50051 list search.SearchService

# Describe the FederatedSearch method
grpcurl -plaintext localhost:50051 describe search.SearchService.FederatedSearch
```

#### 4. Expected Server Logs

When you run a search, the server (Terminal 1) will show:
```
Received search request: query="golang", max_results=5, platforms=[github stackoverflow reddit]
Platform github returned 5 results in 234ms
Platform stackoverflow returned 5 results in 189ms
Platform reddit returned 5 results in 156ms
Search completed in 245ms. Total results: 15 (Success: 3, Timeout: 0, Error: 0)
```

#### 5. Example Response

```json
{
  "results": [
    {
      "platform": "github",
      "title": "golang/go",
      "snippet": "The Go programming language",
      "url": "https://github.com/golang/go",
      "timestamp": "1287542880",
      "metadata": {
        "forks": "18000",
        "language": "Go",
        "stars": "120000"
      }
    },
    {
      "platform": "stackoverflow",
      "title": "How to install Go on Ubuntu?",
      "snippet": "How to install Go on Ubuntu? | Tags: go, ubuntu, installation",
      "url": "https://stackoverflow.com/questions/12345",
      "timestamp": "1609459200",
      "metadata": {
        "answer_count": "5",
        "is_answered": "true",
        "score": "42",
        "tags": "go,ubuntu,installation",
        "view_count": "15000"
      }
    }
  ],
  "totalCount": 15,
  "platformsSuccess": ["github", "stackoverflow", "reddit"],
  "platformsTimeout": [],
  "platformsError": [],
  "metadata": {
    "responseTimeMs": 245,
    "platformsQueried": 3
  }
}
```

### Unit Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test
go test ./internal/fetchers -v
```

## API Documentation

### gRPC Service

**Service**: `SearchService`

**Method**: `FederatedSearch`

**Request** (`SearchRequest`):
```protobuf
{
  "query": "React performance",
  "max_results": 50,
  "platforms": ["github", "stackoverflow", "reddit"]
}
```

**Response** (`SearchResponse`):
```protobuf
{
  "results": [
    {
      "platform": "github",
      "title": "React Performance Tips",
      "snippet": "Optimize your React app...",
      "url": "https://github.com/...",
      "timestamp": 1704067200,
      "metadata": {
        "stars": "1234",
        "language": "javascript"
      }
    }
  ],
  "total_count": 47,
  "platforms_success": ["github", "stackoverflow"],
  "platforms_timeout": ["reddit"],
  "platforms_error": []
}
```

See `proto/search.proto` for complete definitions.

## Development

### Makefile Commands

```bash
make help          # Show all commands
make proto         # Generate gRPC code from .proto
make build         # Build binary
make run           # Run server
make test          # Run tests
make test-coverage # Run tests with coverage
make lint          # Run linter
make clean         # Clean build artifacts
```

### Adding a New Platform

1. Create `internal/fetchers/newplatform.go`
2. Implement the `Fetcher` interface:
   ```go
   type Fetcher interface {
       Fetch(ctx context.Context, query string, maxResults int) ([]*models.Result, error)
   }
   ```
3. Register in `internal/handlers/search.go`
4. Add configuration to `.env`

### Code Style

- Follow [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- Use `gofmt` for formatting
- Add comments for exported functions
- Write tests for all public functions

## Performance

### Benchmarks

- **Response Time**: P95 < 500ms
- **Throughput**: 1000+ RPS
- **Concurrency**: 10,000+ Goroutines
- **Memory**: < 2GB under normal load

### Optimization Tips

1. **Connection Pooling**: Reuse HTTP clients
2. **Context Propagation**: Pass deadlines from client to APIs
3. **Partial Results**: Return what's available, don't wait for all
4. **Circuit Breaker**: Fail fast on repeated errors

## Security

- **No User Data Logging**: Never log queries or user info
- **Privacy Proxy**: External APIs only see server IP
- **Environment Variables**: Store API keys in `.env` (never commit!)
- **Input Validation**: Sanitize all inputs
- **Rate Limiting**: Respect external API limits

## Monitoring

### Health Check

```bash
# gRPC health check (requires grpcurl)
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

### Metrics

- Total requests
- Response time (P50, P95, P99)
- Error rate per platform
- Timeout rate
- Active Goroutines

## Troubleshooting

### Common Issues

**"protoc: command not found"**
```bash
# Install Protocol Buffers compiler
sudo apt install -y protobuf-compiler
```

**"plugin not found"**
```bash
# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**"rate limit exceeded"**
- Add API tokens to `.env`
- Implement caching for popular queries

**"context deadline exceeded"**
- Increase timeout in `.env` (`PER_API_TIMEOUT_MS`)
- Check network connectivity

## Learning Resources

- **gRPC Basics**: See `GRPC_GUIDE.md`
- **Protocol Buffers**: https://protobuf.dev/
- **Go Concurrency**: https://go.dev/tour/concurrency/1
- **Fan-out/Fan-in Pattern**: https://go.dev/blog/pipelines

## Contributing

1. Read the PRD (`PRD.txt`)
2. Follow the code style guide
3. Write tests for new features
4. Update documentation
5. Submit PR with clear description

## License

MIT License (or your preferred license)

## Contact

For questions or issues, please open a GitHub issue or contact the team.

---

**Next Steps**: Read `GRPC_GUIDE.md` to learn how gRPC works, then start implementing!
