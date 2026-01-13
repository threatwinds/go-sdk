# ThreatWinds Catcher - Error Handling, Logging and Retry System

Complete error handling, structured logging and retry operations system for ThreatWinds APIs.

## 🎯 Features

- 🔧 **Robust error handling** with complete stack traces and unique codes
- 📝 **Dual logging system** - Error() for errors, Info() for informational events
- ⚡ **High Performance** - Optional asynchronous logging to minimize latency
- 🔍 **Controllable Verbosity** - Optional stack trace generation to save CPU/memory
- 🎨 **Visual Severity** - Severity icons for better log readability
- 🔄 **Advanced retry system** with exponential backoff, jitter support, and granular configuration
- 🏷️ **Enriched metadata** for better debugging and monitoring
- 🔗 **Native integration** with Gin framework and HTTP status codes
- 🎯 **Structured logging** - JSON with unique codes and stack traces

## Benchmarks

The `catcher` package has been designed to offer an optimal balance between observability and performance. Below are the results obtained on an AMD Ryzen 9 9950X3D processor:

| Scenario | Time (ns/op) | Memory (B/op) | Allocations (allocs/op) |
| :--- | :--- | :--- | :--- |
| **Catcher Info Async (No Trace)** | **708.3** | 1104 | 12 |
| **Catcher Error Async (No Trace)** | **792.4** | 1344 | 14 |
| **Catcher Info Sync (No Trace)** | **853.0** | 960 | 12 |
| **Catcher Error Sync (No Trace)** | **933.2** | 1169 | 14 |
| **Catcher Info Sync (With Trace)** | **2459.0** | 2315 | 27 |
| **Catcher Error Sync (With Trace)** | **2181.0** | 2331 | 27 |
| **slog JSON (Standard)** | **503.3** | 48 | 1 |
| **Catcher Nested Errors (3 levels)** | **1174.0** | 2329 | 23 |
| **slog Nested Errors (3 levels)** | **884.4** | 280 | 9 |
| **Catcher Nested Errors (6 levels)** | **1490.0** | 3362 | 32 |
| **slog Nested Errors (6 levels)** | **1144.0** | 552 | 15 |
| **Standard Log** | **193.2** | 48 | 1 |
| **Catcher Info Async (Parallel)** | **444.8** | 1421 | 19 |
| **Catcher Error Async (Parallel)** | **454.8** | 1557 | 19 |
| **Catcher Info Sync (Parallel)** | **535.1** | 1267 | 19 |
| **Catcher Error Sync (Parallel)** | **560.2** | 1387 | 19 |
| **slog JSON (Parallel)** | **384.5** | 56 | 1 |

### 💡 Performance Analysis and Clarifications

1.  **Concurrency and Asynchronous Mode**: 
    - In highly concurrent environments (Parallel benchmarks), `Catcher` shows its true strength. The time per operation in Async mode stays around **~450ns** when running in parallel, demonstrating excellent scalability and the efficiency of delegating I/O to a dedicated goroutine.
    - `Catcher Error` in parallel performs almost identically to `Catcher Info` (~454ns vs ~444ns), confirming that the overhead of error creation (Cause, additional fields) and metadata processing is negligible in high-load scenarios.
    - Compared to `slog` (Parallel: 384ns), `Catcher` provides significantly richer metadata, unique error IDs, and structured trace information for a minimal performance trade-off.
    - The asynchronous mechanism uses a dedicated goroutine and a buffered channel, allowing multiple logging goroutines to hand off their work without blocking on I/O.
2.  **Realistic Nested Errors and Short-circuit**:
    - **Realistic Simulation**: Benchmarks now use recursive function calls to simulate a real call stack where errors are enriched or propagated across layers.
    - **Short-circuit Efficiency**: `catcher` detects if an error is already of type `SdkError`. If it is, it propagates it immediately without re-processing traces, generating new hashes, or emitting duplicate logs. This results in **O(1)** cost for upper layers.
    - **Comparison**: While `slog` shows good performance, its nesting cost involves manual `fmt.Errorf` wrapping and serialization of the entire error chain. `Catcher` automates this with high efficiency while maintaining full traceability from the origin.

---

## 📦 Installation

```bash
go get github.com/threatwinds/go-sdk/catcher
```

## 🚀 Quick Start

### Basic Error Handling

```go
package main

import (
    "errors"
    "github.com/threatwinds/go-sdk/catcher"
)

func main() {
	// Create an enriched error
    err := catcher.Error("database operation failed", 
        errors.New("connection timeout"), 
        map[string]any{
            "operation": "insert",
            "table": "users",
            "status": 500,
        })

	// Error is automatically logged
	// Output: {"code":"abc123...", "trace":[...], "msg":"database operation failed", ...}
}
```

### Basic Logging

```go
func main() {
// Informational startup log
catcher.Info("service starting", map[string]any{
"service": "api-gateway",
"version": "v1.0.0",
"port": 8080,
})

// Create error with context
err := catcher.Error("database connection failed", dbErr, map[string]any{
"host": "localhost:5432",
"status": 500,
})
}
```

### Retry with Logging

```go
func fetchData() error {
config := &catcher.RetryConfig{
MaxRetries: 5,
WaitTime:   2 * time.Second,
}

return catcher.Retry(func () error {
data, err := apiCall()
if err != nil {
return catcher.Error("API call failed", err, map[string]any{
"endpoint": "/api/data",
"status": 500,
})
}

// Log successful operation
catcher.Info("data fetched successfully", map[string]any{
"endpoint": "/api/data",
"records": len(data),
})

return nil
}, config, "authentication_failed")
}
```

## ⚙️ Configuration

The system can be configured using the following environment variables:

| Variable           | Description                                                                                                                                                  | Default |
|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| `CATCHER_ASYNC`    | If set to `true`, enables asynchronous logging using a buffered channel (10,000 entries). This drastically reduces latency in high-concurrency environments. | `true`  |
| `CATCHER_NO_TRACE` | If set to `true`, disables the automatic generation of stack traces. Recommended for production environments to save CPU and memory.                         | `true`  |
| `CATCHER_BEAUTY`   | If set to `true`, adds severity icons (e.g., ❌, ⚠️, ℹ️) to the logs for better readability in terminal.                                                      | `true`  |

## ⚙️ Retry Configuration

```go
type RetryConfig struct {
MaxRetries int           // Maximum number of retries (0 = infinite)
WaitTime   time.Duration // Wait time between retries
}

// Default configuration
var DefaultRetryConfig = &RetryConfig{
MaxRetries: 5,
WaitTime:   1 * time.Second,
}
```

## 📝 Logging System

The catcher package provides two distinct logging systems for different purposes:

### 🔴 Error Logging - For Error Conditions

**Purpose**: Exclusively for logging **real error conditions** with complete context for debugging.

```go
// Returns *SdkError, logs automatically
err := catcher.Error("operation failed", originalErr, map[string]any{
"operation": "payment",
"status": 500,
})
```

**Features**:

- ✅ **Complete stack trace** (25 frames, optional)
- ✅ **Severity Icons** (optional)
- ✅ **Unique MD5 code** based on the message content
- ✅ **Error chaining** with original cause
- ✅ **Enriched metadata** in `args`
- ✅ **Gin integration** with `GinError()`
- ✅ **Automatic logging** when creating error

### 🔵 Info Logging - For Informational Events

**Purpose**: For logging **important informational events** with structured context, without being errors.

```go
// Logs directly, returns no value
catcher.Info("operation completed", map[string]any{
"operation": "payment",
"success": true,
})
```

**Features**:

- ✅ **Lightweight stack trace** for context (or none if disabled)
- ✅ **Severity Icons** (optional)
- ✅ **Unique MD5 code** based on the message content
- ✅ **Structured metadata** in `args`
- ✅ **Consistent JSON format**
- ❌ **No error chaining** (not an error)
- ✅ **Direct logging** without returning object

### When to Use Each System

| Use `Error()`                   | Use `Info()`           |
|---------------------------------|------------------------|
| ❌ Connection failures           | ✅ Service startup      |
| ❌ Validation errors             | ✅ Operations completed |
| ❌ Timeouts                      | ✅ Configuration loaded |
| ❌ Exceptions                    | ✅ Important metrics    |
| ❌ Authentication failures       | ✅ Business events      |
| ❌ Resource not found (critical) | ✅ System state changes |

### Log Structure Comparison

**Error Log Structure**:

```json
{
  "code": "a1b2c3d4e5f6789...",
  "trace": [
    "main.processPayment 123",
    "api.handleRequest 45"
  ],
  "msg": "payment processing failed",
  "cause": "connection timeout",
  "args": {
    "payment_id": "pay_123",
    "amount": 100.00,
    "status": 500
  }
}
```

**Info Log Structure**:

```json
{
  "code": "b7c8d9e0f1a2b3c4...",
  "trace": [
    "main.startService 89",
    "config.initDatabase 34"
  ],
  "msg": "service started successfully",
  "args": {
    "service": "payment-processor",
    "version": "v1.2.3",
    "port": 8080,
    "environment": "production"
  }
}
```

## 🔧 Available Retry Functions

### 1. `Retry` - Limited retry with maximum attempts

```go
err := catcher.Retry(func () error {
return performOperation()
}, config, "exception1", "exception2")
```

### 2. `InfiniteRetry` - Infinite retry until success or exception

```go
err := catcher.InfiniteRetry(func () error {
return connectToDatabase()
}, config, "auth_failed")
```

### 3. `InfiniteLoop` - Infinite loop until exception

```go
catcher.InfiniteLoop(func () error {
return processMessages()
}, config, "shutdown_signal")
```

### 4. `InfiniteRetryIfXError` - Retry only on specific error

```go
err := catcher.InfiniteRetryIfXError(func () error {
return connectToService()
}, config, "connection_timeout")
```

### 5. `RetryWithBackoff` - Retry with exponential backoff

```go
err := catcher.RetryWithBackoff(func () error {
return callExternalAPI()
}, config,
30*time.Second, // max backoff
2.0, // multiplier
"rate_limited")
```

## 🔍 Error Handling

### Creating Enriched Errors

```go
// Basic error
err := catcher.Error("operation failed", originalErr, map[string]any{
"user_id": "123",
"status": 500,
})

// Database operation error
err := catcher.Error("database query failed", dbErr, map[string]any{
"query": "SELECT * FROM users",
"table": "users",
"operation": "select",
"status": 500,
"retry_able": true,
})

// External API error
err := catcher.Error("external API call failed", apiErr, map[string]any{
"service": "payment_processor",
"endpoint": "/api/v1/charge",
"method": "POST",
"status": 502,
"external": true,
})
```

### Checking Error Types

```go
// Basic exception checking
if catcher.IsException(err, "not_found", "forbidden") {
// Handle specific exception
}

// Advanced checking for SdkError
if sdkErr := catcher.ToSdkError(err); sdkErr != nil {
// Access error metadata
if operation, ok := sdkErr.Args["operation"]; ok {
log.Printf("Failed operation: %s", operation)
}

// Check exceptions in SdkError
if catcher.IsSdkException(sdkErr, "timeout") {
// Handle timeout specifically
}
}
```

## 🌐 Gin Integration

```go
func handleRequest(c *gin.Context) {
err := performOperation()
if err != nil {
// If it's a SdkError, it will be sent automatically with appropriate headers
if sdkErr := catcher.ToSdkError(err); sdkErr != nil {
sdkErr.GinError(c)
return
}

// For other errors, create SdkError
sdkErr := catcher.Error("request failed", err, map[string]any{
"status": 500,
"request_id": c.GetHeader("X-Request-ID"),
})
sdkErr.GinError(c)
}
}
```

## 📋 Practical Examples

### Database Operation

```go
func getUserByID(userID string) (*User, error) {
var user *User

config := &catcher.RetryConfig{
MaxRetries: 5,
WaitTime:   500 * time.Millisecond,
}

err := catcher.RetryWithBackoff(func () error {
u, err := db.GetUser(userID)
if err != nil {
return catcher.Error("failed to get user", err, map[string]any{
"user_id": userID,
"operation": "getUserByID",
"table": "users",
"status": 500,
})
}
user = u
return nil
}, config, 2*time.Second, 2.0, "user_not_found")

return user, err
}
```

### Connect to External Service

```go
func connectToRedis() error {
return catcher.InfiniteRetryIfXError(func () error {
err := redis.Connect()
if err != nil {
return catcher.Error("redis connection failed", err, map[string]any{
"service": "redis",
"host": "localhost:6379",
"critical": true,
"status": 500,
})
}

// Log successful connection
catcher.Info("redis connected successfully", map[string]any{
"service": "redis",
"host": "localhost:6379",
"pool_size": 10,
})

return nil
}, &catcher.RetryConfig{
WaitTime: 5 * time.Second,
}, "connection_refused")
}
```

### Process Message Queue

```go
func processMessageQueue() {
catcher.InfiniteLoop(func () error {
message, err := queue.GetNext()
if err != nil {
return catcher.Error("failed to get message", err, map[string]any{
"queue": "processing",
"operation": "getMessage",
})
}

if message != nil {
err = processMessage(message)
if err != nil {
// Log error but continue processing
catcher.Error("failed to process message", err, map[string]any{
"message_id": message.ID,
"queue": "processing",
})
} else {
// Log successful processing
catcher.Info("message processed successfully", map[string]any{
"message_id": message.ID,
"queue": "processing",
})
}
}

return nil
}, &catcher.RetryConfig{
WaitTime: 1 * time.Second,
}, "shutdown")
}
```

## 📊 Logging and Monitoring

### Complete Application Example

```go
package main

import (
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/gin-gonic/gin"
)

func main() {
	// Informational startup log
	catcher.Info("payment service starting", map[string]any{
		"version": "v1.0.0",
		"port":    8080,
	})

	r := gin.Default()
	r.POST("/payment", handlePayment)

	catcher.Info("payment service ready", map[string]any{
		"endpoints": []string{"/payment"},
		"status":    "ready",
	})

	r.Run(":8080")
}

func handlePayment(c *gin.Context) {
	paymentID := c.Param("id")

	// Informational operation log
	catcher.Info("processing payment", map[string]any{
		"payment_id": paymentID,
		"user_id":    c.GetString("user_id"),
	})

	err := processPayment(paymentID)
	if err != nil {
		// Error log with complete context
		sdkErr := catcher.Error("payment processing failed", err, map[string]any{
			"payment_id": paymentID,
			"user_id":    c.GetString("user_id"),
			"status":     500,
		})
		sdkErr.GinError(c)
		return
	}

	// Informational success log
	catcher.Info("payment processed successfully", map[string]any{
		"payment_id": paymentID,
		"status":     "completed",
	})

	c.JSON(200, gin.H{"status": "success"})
}
```

### Automatic Retry Logging

The system automatically logs:

- ✅ **Retry start** with configuration
- 🔄 **Failed attempts** with error details
- ✅ **Success after retries**
- ❌ **Final failure** after maximum retries
- 🛑 **Exception stop**

## 🧪 Testing

```go
func TestRetryOperation(t *testing.T) {
attempts := 0

err := catcher.Retry(func () error {
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
```

## 🔍 Debugging and Monitoring

### Filter by Type

```bash
# Only errors (have "cause")
jq 'select(.cause != null)' app.log

# Only info logs (no "cause")  
jq 'select(.cause == null)' app.log

# Filter by specific code
jq 'select(.code == "a1b2c3d4e5f6789...")' app.log
```

### Error Analysis

```bash
# Top most frequent errors
jq -r '.code' app.log | sort | uniq -c | sort -nr | head -10

# Errors from specific service
jq 'select(.args.service == "payment-processor" and .cause != null)' app.log
```

## 🚀 Monitoring Integration

Both systems generate structured logs ideal for:

- **📊 Elasticsearch/OpenSearch** - Indexing and search
- **📈 Grafana** - Dashboards and alerts
- **🔔 Alertmanager** - Notifications by error codes
- **📋 Jaeger/Zipkin** - Distributed tracing using unique codes

## 📈 Benefits of the Catcher System

1. **⚡ High Performance**: Low-latency logging (down to ~450 ns/op) with asynchronous mode and reduced allocations.
2. **🔍 Better Debugging**: Complete stack traces and unique error codes.
3. **📊 Advanced Monitoring**: Rich metadata for alerts and metrics.
4. **⚙️ Flexibility**: Granular retry configuration per operation and controllable verbosity.
5. **🚀 Reliability**: Automatic fallback to synchronous logging if the async buffer is full.
6. **🛠️ Maintainability**: Clear separation between logging and retry logic.
7. **🔗 Integration**: Native support for web frameworks like Gin.

## 🆘 Troubleshooting

### ❓ **Problem**: Why don't I see successful retry logs?

**✅ Solution**: This is intentional - catcher only logs real errors, not successful operations

### ❓ **Problem**: Complex configuration

**✅ Solution**: Use `catcher.DefaultRetryConfig` or create reusable configs

---

## 💡 Tips and Best Practices

1. **Use descriptive metadata** in your errors for better debugging
2. **Configure retry strategies** specific to operation type
3. **Avoid infinite retry** in time-critical operations
4. **Use exponential backoff** for external services
5. **Group configurations** by application domain (DB, API, etc.)
6. **Use Error() only for real errors** - not for informational events
7. **Include unique identifiers** (IDs) when relevant
8. **Don't include sensitive information** in logs

The catcher system is ready to improve the robustness and observability of your ThreatWinds applications! 🚀