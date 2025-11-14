# CLAUDE.md - ThreatWinds Go SDK

## Library Overview

The ThreatWinds Go SDK is the **core utility library** for all ThreatWinds services and processors. It provides essential components for error handling, threat intelligence data models, OpenSearch operations, and common utilities. This is NOT an API client - it's a foundational library used by EventProcessor and all microservices.

**Key Components:**
- **catcher**: Advanced error handling and retry system with structured logging
- **entities**: Shared threat intelligence data models (IP, domain, hash, etc.)
- **opensearch**: OpenSearch client utilities and query builders
- **utils**: Common utilities (HTTP requests, JSON/YAML parsing, file operations, etc.)

## Installation

```go
import (
    "github.com/threatwinds/go-sdk/catcher"
    "github.com/threatwinds/go-sdk/entities"
    "github.com/threatwinds/go-sdk/opensearch"
    "github.com/threatwinds/go-sdk/utils"
)
```

## Core Features

### 1. Catcher - Error Handling & Retry System

The catcher package provides **robust error handling** with complete stack traces, unique error codes, and advanced retry mechanisms.

#### Key Features:
- **SdkError**: Custom error type with stack traces, unique MD5 codes, and metadata
- **Dual logging**: `Error()` for errors, `Info()` for informational events
- **Retry functions**: Multiple retry strategies (limited, infinite, exponential backoff)
- **Gin integration**: Native HTTP error handling for Gin framework
- **Structured logging**: JSON format with unique codes for monitoring

#### Critical Rules:
- **ALWAYS use `catcher.Error()` - NEVER return standard Go errors**
- **ALWAYS include `status` in error args for HTTP status codes**
- All errors are automatically logged when created
- Use `Info()` for successful operations, `Error()` only for failures

### 2. Entities - Threat Intelligence Models

Shared data structures for threat intelligence entities across ThreatWinds platform.

#### Key Types:
- **EntityConsolidated**: Aggregated entity data (IP, domain, hash, etc.)
- **EntityHistory**: Historical entity snapshots with user attribution
- **RelationConsolidated**: Entity relationships (e.g., domain → IP)
- **RelationHistory**: Historical relationship records
- **Comment**: User comments on entities
- **Entity**: Entity submission structure with associations
- **Attributes**: Type-specific attributes (45+ types including IP, FQDN, MD5, SHA256, Email, etc.)

### 3. OpenSearch - Search Utilities

Simplified OpenSearch operations with query builders.

#### Features:
- Index management (create, delete, exists checks)
- Bulk operations for high-throughput writes
- Query builders with Bool, Term, Range, Match queries
- Source filtering and aggregations
- Search with visibility controls (groups-based filtering)

### 4. Utils - Common Utilities

Shared utility functions for ThreatWinds services.

#### Available Utilities:
- **HTTP**: `DoReq()` - Generic HTTP request handler with timeout and TLS
- **JSON/YAML**: Marshal/unmarshal with error handling
- **File Operations**: Read, write, list files
- **CSV**: Parse and generate CSV data
- **Network**: IP parsing, CIDR validation
- **Type Casting**: Safe type conversions
- **Metrics**: Performance measurement utilities
- **Protobuf**: Protobuf JSON conversion

## Usage Examples

### Error Handling with Catcher

```go
package main

import (
    "github.com/threatwinds/go-sdk/catcher"
    "net/http"
)

func getUserByID(userID string) (*User, error) {
    user, err := db.Query("SELECT * FROM users WHERE id = ?", userID)
    if err != nil {
        // ALWAYS use catcher.Error, NEVER return err directly
        return nil, catcher.Error("failed to get user", err, map[string]any{
            "user_id": userID,
            "operation": "getUserByID",
            "status": http.StatusInternalServerError, // REQUIRED for severity
        })
    }

    if user == nil {
        return nil, catcher.Error("user not found", nil, map[string]any{
            "user_id": userID,
            "status": http.StatusNotFound,
        })
    }

    // Log successful operation
    catcher.Info("user retrieved successfully", map[string]any{
        "user_id": userID,
        "operation": "getUserByID",
    })

    return user, nil
}
```

### Retry with Exponential Backoff

```go
import (
    "github.com/threatwinds/go-sdk/catcher"
    "time"
)

func connectToExternalAPI() error {
    config := &catcher.RetryConfig{
        MaxRetries: 5,
        WaitTime:   1 * time.Second,
    }

    return catcher.RetryWithBackoff(func() error {
        err := externalAPI.Connect()
        if err != nil {
            return catcher.Error("API connection failed", err, map[string]any{
                "service": "external-api",
                "status": 502,
            })
        }
        return nil
    }, config,
        30*time.Second, // max backoff
        2.0,            // multiplier
        "auth_failed")  // stop on this exception
}
```

### Infinite Retry for Critical Services

```go
func connectToDatabase() error {
    // Retry forever until success or specific exception
    return catcher.InfiniteRetryIfXError(func() error {
        err := db.Connect()
        if err != nil {
            return catcher.Error("database connection failed", err, map[string]any{
                "host": "localhost:5432",
                "status": 500,
            })
        }

        catcher.Info("database connected", map[string]any{
            "host": "localhost:5432",
        })

        return nil
    }, &catcher.RetryConfig{
        WaitTime: 5 * time.Second,
    }, "connection_refused")
}
```

### Gin Framework Integration

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/threatwinds/go-sdk/catcher"
)

func handleRequest(c *gin.Context) {
    data, err := processRequest(c)
    if err != nil {
        // SdkError automatically sets headers and status
        if sdkErr := catcher.ToSdkError(err); sdkErr != nil {
            sdkErr.GinError(c) // Sets x-error-id, x-error headers + status
            return
        }

        // For non-SdkError, convert it
        catcher.Error("request failed", err, map[string]any{
            "status": 500,
        }).GinError(c)
        return
    }

    c.JSON(200, data)
}
```

### Working with Entities

```go
import (
    "github.com/threatwinds/go-sdk/entities"
    "github.com/google/uuid"
)

func createIPEntity(ip string, reputation int) *entities.Entity {
    ipAttr := entities.IP{}
    err := ipAttr.Set(ip)
    if err != nil {
        return nil
    }

    return &entities.Entity{
        Type: "ip",
        Attributes: entities.Attributes{
            IP: &ipAttr,
        },
        Reputation: reputation,
        Tags: []string{"scanner", "malicious"},
        VisibleBy: []string{"public"},
        Correlate: []string{}, // No correlations for IP
    }
}

func createFileEntity(md5, sha256 string) *entities.Entity {
    md5Attr := entities.MD5{}
    md5Attr.Set(md5)

    sha256Attr := entities.SHA256{}
    sha256Attr.Set(sha256)

    return &entities.Entity{
        Type: "object",
        Attributes: entities.Attributes{
            MD5:    &md5Attr,
            SHA256: &sha256Attr,
        },
        Reputation: -3,
        Tags: []string{"malware"},
        VisibleBy: []string{"public"},
        Correlate: []string{"md5", "sha256"}, // Correlate by hashes
    }
}
```

### OpenSearch Operations

```go
import (
    "github.com/threatwinds/go-sdk/opensearch"
    "context"
)

func searchEntities(ctx context.Context, ipAddress string) error {
    // Connect to OpenSearch (singleton)
    err := opensearch.Connect([]string{"https://localhost:9200"}, "admin", "password")
    if err != nil {
        return catcher.Error("opensearch connection failed", err, nil)
    }

    // Build search query
    query := opensearch.SearchRequest{
        Size: 100,
        Query: &opensearch.Query{
            Bool: &opensearch.Bool{
                Filter: []opensearch.Query{
                    {Term: map[string]map[string]interface{}{
                        "type": {"value": "ip"},
                    }},
                    {Term: map[string]map[string]interface{}{
                        "attributes.ip.keyword": {"value": ipAddress},
                    }},
                },
            },
        },
        Source: &opensearch.Source{
            Includes: []string{"id", "reputation", "tags"},
        },
    }

    // Execute search with group-based visibility
    results, err := query.SearchIn(ctx, []string{"entities"}, []string{"public", "myorg"})
    if err != nil {
        return catcher.Error("search failed", err, map[string]any{
            "status": 500,
        })
    }

    // Process results
    for _, hit := range results.Hits.Hits {
        // hit.Source contains the document
        fmt.Printf("Found entity: %s\n", hit.ID)
    }

    return nil
}
```

### HTTP Requests with Utils

```go
import (
    "github.com/threatwinds/go-sdk/utils"
)

type APIResponse struct {
    Status string `json:"status"`
    Data   any    `json:"data"`
}

func callExternalAPI() error {
    payload := map[string]string{
        "action": "query",
        "param": "value",
    }

    data, _ := json.Marshal(payload)

    response, status, err := utils.DoReq[APIResponse](
        "https://api.example.com/endpoint",
        data,
        "POST",
        map[string]string{
            "Content-Type": "application/json",
            "Authorization": "Bearer token",
        },
    )

    if err != nil {
        return catcher.Error("API request failed", err, map[string]any{
            "status": status,
        })
    }

    // response is typed as APIResponse
    fmt.Printf("API Status: %s\n", response.Status)

    return nil
}
```

## Configuration

### Catcher Beauty Mode

Enable emoji icons in logs for better readability in development:

```bash
export CATCHER_BEAUTY=true
```

Output example:
```
ℹ️  {"timestamp":"2025-01-14T10:00:00Z","code":"abc123","msg":"service started","severity":"INFO"}
❌ {"timestamp":"2025-01-14T10:00:05Z","code":"def456","msg":"connection failed","cause":"timeout","severity":"ERROR"}
```

### Retry Configuration

```go
// Default configuration
catcher.DefaultRetryConfig = &catcher.RetryConfig{
    MaxRetries: 5,
    WaitTime:   1 * time.Second,
}

// Custom configuration
config := &catcher.RetryConfig{
    MaxRetries: 10,
    WaitTime:   2 * time.Second,
}
```

## Common Patterns

### Service Initialization Pattern

```go
func initializeService() error {
    catcher.Info("service initializing", map[string]any{
        "service": "my-service",
        "version": "v1.0.0",
    })

    // Connect to dependencies with retry
    err := catcher.InfiniteRetry(func() error {
        return connectToDatabase()
    }, nil, "shutdown")

    if err != nil {
        return catcher.Error("service initialization failed", err, map[string]any{
            "status": 500,
        })
    }

    catcher.Info("service ready", map[string]any{
        "service": "my-service",
        "status": "ready",
    })

    return nil
}
```

### Background Task Pattern

```go
func processQueue() {
    catcher.InfiniteLoop(func() error {
        msg, err := queue.GetNext()
        if err != nil {
            return catcher.Error("queue read failed", err, map[string]any{
                "queue": "tasks",
                "status": 500,
            })
        }

        if msg != nil {
            err = processMessage(msg)
            if err != nil {
                catcher.Error("message processing failed", err, map[string]any{
                    "message_id": msg.ID,
                    "status": 500,
                })
            } else {
                catcher.Info("message processed", map[string]any{
                    "message_id": msg.ID,
                })
            }
        }

        return nil
    }, &catcher.RetryConfig{
        WaitTime: 1 * time.Second,
    }, "shutdown")
}
```

### Error Exception Checking

```go
func handleError(err error) {
    // Check for specific exceptions
    if catcher.IsException(err, "not_found", "forbidden") {
        // Handle expected errors
        return
    }

    // Check SdkError metadata
    if sdkErr := catcher.ToSdkError(err); sdkErr != nil {
        if operation, ok := sdkErr.Args["operation"]; ok {
            log.Printf("Failed operation: %s", operation)
        }

        // Check specific exceptions in SdkError
        if catcher.IsSdkException(sdkErr, "timeout") {
            // Retry logic
        }
    }
}
```

## Testing

### Unit Tests with Catcher

```go
func TestRetryOperation(t *testing.T) {
    attempts := 0

    err := catcher.Retry(func() error {
        attempts++
        if attempts < 3 {
            return errors.New("temporary error")
        }
        return nil
    }, &catcher.RetryConfig{
        MaxRetries: 5,
        WaitTime:   10 * time.Millisecond,
    })

    assert.NoError(t, err)
    assert.Equal(t, 3, attempts)
}

func TestErrorCreation(t *testing.T) {
    err := catcher.Error("test error", nil, map[string]any{
        "status": 500,
        "user_id": "123",
    })

    assert.NotNil(t, err)
    assert.Equal(t, "test error", err.Msg)
    assert.NotEmpty(t, err.Code)
    assert.NotEmpty(t, err.Trace)

    // Check metadata
    assert.Equal(t, 500, err.Args["status"])
    assert.Equal(t, "123", err.Args["user_id"])
}
```

### Testing with Entities

```go
func TestIPValidation(t *testing.T) {
    ip := entities.IP{}

    // Valid IP
    err := ip.Set("192.168.1.1")
    assert.NoError(t, err)
    assert.Equal(t, "192.168.1.1", ip.Get())

    // Invalid IP
    err = ip.Set("invalid")
    assert.Error(t, err)
}
```

## Common Gotchas

### 1. NEVER Return Standard Go Errors

```go
// ❌ WRONG - Never do this
func badExample() error {
    return errors.New("something failed")
}

// ✅ CORRECT - Always use catcher.Error
func goodExample() error {
    return catcher.Error("something failed", nil, map[string]any{
        "status": 500,
    })
}
```

### 2. ALWAYS Include Status in Error Args

```go
// ❌ WRONG - Missing status
catcher.Error("operation failed", err, nil)

// ✅ CORRECT - Status determines severity
catcher.Error("operation failed", err, map[string]any{
    "status": 500, // Required for proper severity level
})
```

### 3. Use Error() Only for Real Errors

```go
// ❌ WRONG - Don't use Error() for success
catcher.Error("user created successfully", nil, map[string]any{"status": 200})

// ✅ CORRECT - Use Info() for successful operations
catcher.Info("user created successfully", map[string]any{
    "user_id": "123",
})
```

### 4. Entity Attribute Validation

```go
// All entity attributes have validation
ip := entities.IP{}
err := ip.Set("invalid-ip")
// err != nil - validation failed

// Always check Set() errors
if err := attr.Set(value); err != nil {
    return catcher.Error("invalid attribute", err, map[string]any{
        "status": 400,
    })
}
```

### 5. OpenSearch Connection is Singleton

```go
// Connection happens once, subsequent calls return cached client
opensearch.Connect(nodes, user, pass) // First call connects
opensearch.Connect(nodes, user, pass) // Second call returns existing client
```

### 6. Retry Functions Swallow Success Logs

```go
// Catcher retry functions do NOT log successful completions
// If you need to log success, do it in your function:
catcher.Retry(func() error {
    err := doSomething()
    if err != nil {
        return catcher.Error("failed", err, nil)
    }

    // Log success explicitly
    catcher.Info("operation succeeded", map[string]any{
        "operation": "doSomething",
    })

    return nil
}, nil)
```

## Dependencies

### Required:
- `github.com/gin-gonic/gin` - HTTP framework (for Gin integration)
- `github.com/google/uuid` - UUID generation
- `github.com/opensearch-project/opensearch-go/v2` - OpenSearch client
- `github.com/tidwall/gjson` - JSON parsing
- `golang.org/x/crypto` - Cryptographic operations
- `google.golang.org/protobuf` - Protocol buffers
- `gopkg.in/yaml.v3` - YAML parsing

### Import Paths:
```go
github.com/threatwinds/go-sdk/catcher
github.com/threatwinds/go-sdk/entities
github.com/threatwinds/go-sdk/opensearch
github.com/threatwinds/go-sdk/utils
```

## Reference

### Related Documentation:
- Catcher README: `/go-sdk/catcher/README.md` - Comprehensive error handling guide
- Main README: `/go-sdk/README.md`
- Used by: All ThreatWinds microservices, EventProcessor, processors
- Related libraries: `logger`, `opensearch-go-wrapper`, `datastore-go-wrapper`

### Catcher vs Logger:
- **catcher**: Modern error handling with SdkError, use in new code
- **logger**: Legacy logging system, deprecated in favor of catcher
- Both exist for backward compatibility, prefer catcher for new development

### Key Concepts:
- **SdkError**: Custom error type with stack traces and metadata
- **Retry Strategies**: Limited, infinite, exponential backoff, conditional
- **Entities**: Threat intelligence data models (45+ attribute types)
- **Visibility**: Group-based access control (`visibleBy` field)
- **Correlation**: Entity linking via `correlate` field (e.g., hash correlation)
