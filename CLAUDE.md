# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Testing
```bash
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

# Run a specific test
go test -run TestValidateIP ./entities

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go coverage -html=coverage.out
```

### Module Management
```bash
# Update dependencies
go mod tidy

# Download dependencies
go mod download

# Update specific dependency
go get -u github.com/gin-gonic/gin@latest

# Verify dependencies
go mod verify
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run go vet for static analysis
go vet ./...

# Install and run golangci-lint (recommended linter)
golangci-lint run
```

## High-Level Architecture

### Package Structure

The SDK is organized into distinct packages, each serving a specific purpose:

1. **catcher/** - Error handling, logging, and retry system
   - Provides structured error handling with stack traces and unique error codes
   - Dual logging system: Error() for errors, Info() for informational events
   - Advanced retry mechanisms with exponential backoff
   - Native integration with Gin framework

2. **entities/** - Data validation and type definitions
   - Implements validators for various data types (IP, URL, email, hashes, etc.)
   - Provides base entity structures with attributes system
   - Each entity type has its own file with validation logic and tests

3. **opensearch/** - OpenSearch client wrapper
   - Manages connection to OpenSearch clusters
   - Provides bulk operations, indexing, searching, and deletion functionality
   - Uses singleton pattern for client management

4. **plugins/** - Plugin system with gRPC support
   - Implements plugin architecture with protobuf definitions
   - Handles plugin configuration, logging, and notifications
   - CEL (Common Expression Language) integration for dynamic expressions
   - Socket-based communication support

5. **utils/** - Common utility functions
   - JSON/YAML handling
   - File and folder operations
   - Network utilities
   - Type conversions and pointer helpers
   - Request handling and metrics

### Key Design Patterns

1. **Error Handling**: All errors use the catcher package for consistent error handling with metadata and stack traces. Errors are automatically logged with unique codes.

2. **Validation**: The entities package uses a consistent validation pattern where each type has a Validate() method that returns an error if validation fails.

3. **Testing**: Every major component has corresponding test files following Go conventions (*_test.go). Tests use the testify/assert library for assertions.

4. **Singleton Pattern**: The opensearch client uses sync.Once to ensure single initialization.

### Integration Points

1. **Gin Framework**: The catcher package provides GinError() method for seamless integration with Gin HTTP handlers.

2. **OpenSearch**: The opensearch package wraps the official OpenSearch Go client with additional functionality.

3. **gRPC**: The plugins package uses gRPC for plugin communication with generated protobuf code.

4. **CEL**: The plugins package integrates Google's CEL for expression evaluation.

### Important Conventions

1. **Error Messages**: Always use catcher.Error() for error conditions with descriptive messages and metadata.

2. **Logging**: Use catcher.Info() for informational events, not catcher.Error().

3. **Testing**: Write tests for all new functionality. Tests should be in the same package as the code they test.

4. **Validation**: Entity validators should return descriptive error messages indicating what validation failed.

5. **Package Independence**: Packages should minimize dependencies on other packages in the SDK.