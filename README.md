# PulseDB

PulseDB is a Redis-like in-memory database built in Go with advanced features including time-travel queries (MVCC), event-driven WASM functions, and enhanced streams.

## Features

### Core Features
- **RESP3-compatible protocol** - Works with standard Redis clients
- **Sharded in-memory storage** - 64 shards for low lock contention
- **TTL management** - Efficient O(1) expiration using timing wheel
- **Persistence** - Append-only log (AOF) + periodic snapshots (planned)

### Advanced Features
- **Time-Travel Queries (MVCC)** - Query key values at any point in time
- **Event-Driven WASM Functions** - Upload and execute WASM code on key events (planned)
- **Enhanced Streams** - Exactly-once delivery and idempotent writes (planned)

## Architecture

```
PulseDB/
├── cmd/pulsedb/          # Entry point
├── internal/
│   ├── server/           # TCP server (RESP protocol)
│   ├── proto/            # RESP parser/writer
│   ├── store/            # Sharded in-memory store with MVCC
│   ├── wasm/             # WASM runtime (planned)
│   ├── streams/          # Streams implementation (planned)
│   ├── http/             # HTTP REST API
│   └── metrics/          # Prometheus metrics
```

## Quick Start

### Build and Run

```bash
# Clone the repository
git clone <repository-url>
cd redis-clone

# Build the project
go build -o pulsedb cmd/pulsedb/main.go

# Run PulseDB
./pulsedb
```

PulseDB will start on:
- TCP port 6380 (RESP protocol)
- HTTP port 8080 (REST API)

### Using Redis CLI

```bash
# Connect using redis-cli
redis-cli -p 6380

# Basic operations
127.0.0.1:6380> PING
PONG
127.0.0.1:6380> SET mykey "Hello World"
OK
127.0.0.1:6380> GET mykey
"Hello World"
```

## Commands

### Basic Commands
- `PING [message]` - Ping the server
- `SET key value [EX seconds] [PX milliseconds]` - Set a key-value pair with optional TTL
- `GET key` - Get the value of a key
- `DEL key [key ...]` - Delete one or more keys
- `EXPIRE key seconds` - Set TTL for a key
- `TTL key` - Get remaining TTL for a key

### Time-Travel Commands (MVCC)
- `GETAT key timestamp` - Get value of key at specific Unix millisecond timestamp
- `HIST key [limit]` - Get version history of a key (newest first)

### Examples

```bash
# Set a key with TTL
127.0.0.1:6380> SET session:123 "user_data" EX 3600
OK

# Basic operations
127.0.0.1:6380> SET counter 1
OK
127.0.0.1:6380> SET counter 2
OK
127.0.0.1:6380> SET counter 3
OK

# Get current value
127.0.0.1:6380> GET counter
"3"

# Get value at specific timestamp (Unix milliseconds)
127.0.0.1:6380> GETAT counter 1693353600000
"1"

# Get version history (timestamp, value pairs)
127.0.0.1:6380> HIST counter 5
1) (integer) 1693353602000
2) "3"
3) (integer) 1693353601000
4) "2"
5) (integer) 1693353600000
6) "1"
```

## HTTP API

### Endpoints

#### Key-Value Operations
- `GET /kv/{key}` - Get a key's value
- `POST /kv/{key}` - Set a key's value
- `DELETE /kv/{key}` - Delete a key

#### Health and Metrics
- `GET /health` - Health check and stats
- `GET /metrics` - Prometheus metrics (planned)

### Examples

```bash
# Set a key via HTTP
curl -X POST http://localhost:8080/kv/mykey \
  -H "Content-Type: application/json" \
  -d '{"value": "Hello HTTP", "ttl": 3600}'

# Get a key via HTTP
curl http://localhost:8080/kv/mykey

# Response:
# {"key":"mykey","value":"Hello HTTP","found":true}

# Health check
curl http://localhost:8080/health

# Response:
# {"status":"healthy","stats":{"shard_count":64,"total_keys":1,"total_versions":1}}
```

## Configuration

Currently, PulseDB uses hardcoded configuration:
- TCP Port: 6380
- HTTP Port: 8080
- Shard Count: 64
- Max Versions per Key: 10
- TTL Check Interval: 1 second

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific package tests
go test ./internal/store/
go test ./internal/proto/
```

### Project Structure

- `cmd/pulsedb/main.go` - Application entry point with server startup
- `internal/proto/` - RESP protocol implementation
- `internal/store/` - Core storage engine with MVCC support
- `internal/server/` - TCP server and command dispatcher
- `internal/http/` - HTTP API server
- `internal/metrics/` - Prometheus metrics (planned)

## Planned Features

### Event-Driven WASM Functions
```bash
# Upload WASM function
FUNC.LOAD audit_logger <wasm_bytes>

# Bind function to key events
ON.SET user:* audit_logger
ON.EXPIRE session:* cleanup_handler

# WASM functions can use host functions:
# - get(key) -> value
# - set(key, value)
# - publish(channel, message)
```

### Enhanced Streams
```bash
# Add entry with idempotency
XADD events * user_id 123 action login IDEMPOTENT req_uuid_456

# Read with exactly-once semantics
XREADGROUP GROUP processors worker1 COUNT 10 BLOCK 1000 STREAMS events >
```

### Persistence
- Append-only file (AOF) for durability
- Periodic snapshots (RDB-like) for faster startup
- Background compaction and optimization

## Performance

PulseDB is designed for high performance:
- **Sharded storage** - 64 shards minimize lock contention
- **Lock-free reads** - MVCC allows concurrent reads
- **Efficient TTL** - Timing wheel provides O(1) expiration
- **Background processing** - Non-blocking cleanup and maintenance

## License

This project is intended for educational and demonstration purposes.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Troubleshooting

### Common Issues

1. **Port already in use**
   - Change the default ports in `main.go`
   - Kill existing processes: `lsof -ti:6380 | xargs kill`

2. **Connection refused**
   - Ensure PulseDB is running
   - Check firewall settings
   - Verify port configuration

3. **Out of memory**
   - PulseDB stores all data in memory
   - Monitor memory usage via `/health` endpoint
   - Implement TTL for automatic cleanup
# pulsedb
