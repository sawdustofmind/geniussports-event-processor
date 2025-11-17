# GeniusSports Event Processor

A Go-based event processing system that replays American Football game events from a data file, with a producer service sending messages to a consumer service that stores game data in Redis.

## Architecture

- **Producer Service**: Parses the `PIT_LAC.txt` file, replays messages with updated timestamps, and sends them to the consumer
- **Consumer Service**: Exposes HTTP endpoints to receive messages and stores fixture and score data in Redis
- **Redis**: Stores fixture information, team names, and score progression with timestamps

## Features

- ✅ Message parsing from large data files (>200MB)
- ✅ Timestamp replacement with current time
- ✅ Configurable replay speed
- ✅ Heartbeat monitoring (every 60 seconds)
- ✅ Redis storage for fixtures and scores
- ✅ Dockerized services with docker-compose
- ✅ Graceful shutdown handling

## Redis Data Structure

### Fixture Information
- **Key**: `fixture:{fixture_id}`
- **Type**: Hash
- **Fields**: `away_team`, `home_team`, `fixture_status`, `start_time`, `away`, `home`, `is_confirmed`, `period_num`, `period_type`, `is_running`

## Prerequisites

- Go 1.24+
- Redis (or use Docker)
- [Task](https://taskfile.dev/) - Task runner (optional but recommended)
- [golangci-lint](https://golangci-lint.run/) - For linting (auto-installed by Task)

### Install Task (optional)

```bash
# macOS
brew install go-task

# Linux/WSL
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

# Or with Go
go install github.com/go-task/task/v3/cmd/task@latest
```

## Quick Commands with Task

```bash
task              # Show all available tasks
task up           # Build and start all services with docker-compose
task lint         # Run golangci-lint
task docker-build # Build Docker images
task ci           # Run full CI pipeline (lint, test, build)
```

## Running Locally

### Prerequisites
- Go 1.24+
- Redis (or use Docker)

### Run with Go

1. **Start Redis**:
   ```bash
   docker run -d -p 6379:6379 redis:7-alpine
   ```

2. **Download dependencies**:
   ```bash
   go mod download
   ```

3. **Run Consumer** (in one terminal):
   ```bash
   go run cmd/consumer/main.go -port 8080 -redis localhost:6379
   ```

4. **Run Producer** (in another terminal):
   ```bash
   go run cmd/producer/main.go -file PIT_LAC.txt -consumer http://localhost:8080 -speed 100ms
   ```

### Command Line Options

**Producer**:
- `-file`: Path to data file (default: `PIT_LAC.txt`)
- `-consumer`: Consumer service URL (default: `http://localhost:8080`)
- `-speed`: Replay speed multiplier (default: `10.0`, use `1.0` for real-time)

**Consumer**:
- `-port`: Port to listen on (default: `8080`)
- `-redis`: Redis address (default: `localhost:6379`)

## Running with Docker

### Build and run all services:

**With Task:**
```bash
task up
```

**With docker-compose:**
```bash
docker-compose up --build
```

This will start:
- Redis on port 6379
- Consumer service on port 8080
- Producer service (runs once and exits)

### View logs:

```bash
docker-compose logs -f producer
docker-compose logs -f consumer
```

### Stop services:

```bash
docker-compose down
```

### Clean up including volumes:

```bash
docker-compose down -v
```

## API Endpoints

### POST /heartbeat
Health check endpoint.

**Response**:
```json
{
  "status": "ok"
}
```

### POST /process-msg
Processes incoming event messages.

**Request Body**:
```json
{
  "Header": {
    "Retry": 0,
    "MessageGuid": "uuid",
    "TimeStampUtc": "2025-11-15T12:00:00Z"
  },
  "Fixture": { ... },
  "AmericanFootballMatchState": { ... }
}
```

**Response**: 200 OK

## Querying Redis

Connect to Redis and query the data:

```bash
# Connect to Redis
docker exec -it event-processor-redis redis-cli

# Get fixture info
HGETALL fixture:12282289
```

## Development

### Using Task (Recommended)

```bash
# Show all available tasks
task

# Run linter
task lint

# Fix linting issues automatically
task lint-fix

# Format code
task fmt

# Run tests
task test

# Build locally
task build

# Build Docker images
task docker-build

# Run full CI checks
task ci

# Start dev environment (Redis + Consumer)
task dev

# Clean build artifacts
task clean
```

### Manual Commands

```bash
# Add new dependencies
go get <package>
go mod tidy

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint code
golangci-lint run ./...
```

## Message Types

The system processes two types of messages:

1. **Fixture Messages**: Contains game metadata and team information
2. **Match State Messages**: Contains current score and period information

All timestamps in incoming messages are replaced with current UTC time before processing.

## License

MIT

## Author

David Basenko

