# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./catcher
go test ./entities
go test ./opensearch
go test ./plugins
go test ./utils

# Run a single test
go test ./catcher -run TestRetry
go test ./entities -run TestValidateIP

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./catcher
```

## Library Overview

The ThreatWinds Go SDK is a **core utility library** (not an API client) used by all ThreatWinds services and processors. It provides error handling, threat intelligence data models, OpenSearch operations, and common utilities.

## Package Architecture

```
go-sdk/
├── catcher/     # Error handling, logging, and retry system
├── entities/    # Threat intelligence data models (IP, domain, hash, etc.)
├── opensearch/  # OpenSearch client utilities and query builders
├── plugins/     # Plugin utilities (CEL expressions, notifications, gRPC)
└── utils/       # Common utilities (HTTP, JSON, YAML, files, network)
```

### Package Dependencies Flow
- `utils` → standalone utilities, no internal dependencies
- `catcher` → uses `utils` for JSON marshaling
- `entities` → uses `catcher` for error handling
- `opensearch` → uses `catcher` for errors, `utils` for JSON
- `plugins` → uses `catcher` for errors, contains protobuf definitions

## Critical Rules

### Error Handling
- **ALWAYS use `catcher.Error()` - NEVER return standard Go errors**
- **ALWAYS include `status` in error args** for HTTP status codes (determines severity)
- All errors are automatically logged when created
- Use `catcher.Info()` for successful operations, `catcher.Error()` only for failures

```go
// WRONG - Never do this
return errors.New("operation failed")

// CORRECT - Always use catcher.Error
return catcher.Error("operation failed", err, map[string]any{
    "status": 500,
})
```

### Entity Attributes
- All entity attributes have validation via `Set()` method
- Always check `Set()` errors before using attributes
- Available types: IP, FQDN, MD5, SHA256, Email, CIDR, URL, and 40+ others

## Key Patterns

### Retry Functions
- `Retry()` - Limited retry with max attempts
- `InfiniteRetry()` - Retry forever until success or exception
- `RetryWithBackoff()` - Exponential backoff for external services
- `InfiniteLoop()` - Continuous processing until shutdown signal

### OpenSearch
- Connection is singleton - `Connect()` caches client on first call
- Use `SearchRequest` struct to build queries
- Visibility controls via `visibleBy` field for group-based access

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `CATCHER_BEAUTY=true` | Enable emoji icons in log output for development |

## Documentation References

- Detailed catcher docs: [catcher/README.md](catcher/README.md)
- Entity attribute types: see `entities/*.go` files (each type has its own file)
- OpenSearch query building: see `opensearch/search.go` and `opensearch/schema.go`
