# ThreatWinds Catcher - Error Handling, Logging and Retry System

Complete error handling, structured logging and retry operations system for ThreatWinds APIs.

## 🎯 Features

- 🔧 **Robust error handling** with complete stack traces and unique codes
- 📝 **Dual logging system** - Error() for errors, Info() for informational events
- 🔄 **Advanced retry system** with exponential backoff and granular configuration
- 🏷️ **Enriched metadata** for better debugging and monitoring
- 🔗 **Native integration** with Gin framework and HTTP status codes
- 🎯 **Structured logging** - JSON with unique codes and stack traces

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

- ✅ **Complete stack trace** (25 frames)
- ✅ **Unique MD5 code** based on message
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

- ✅ **Lightweight stack trace** for context
- ✅ **Unique MD5 code** based on message
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

1. **🔍 Better Debugging**: Complete stack traces and unique error codes
2. **📊 Advanced Monitoring**: Rich metadata for alerts and metrics
3. **⚙️ Flexibility**: Granular retry configuration per operation
4. **🚀 Performance**: Exponential backoff for external services
5. **🛠️ Maintainability**: Clear separation between logging and retry logic
6. **🔗 Integration**: Native support for web frameworks

## 🆘 Troubleshooting

### ❓ **Problem**: Why don't I see successful retry logs?

**✅ Solution**: This is intentional - catcher only logs real errors, not successful operations

### ❓ **Problem**: Complex configuration

**✅ Solution**: Use `catcher.DefaultRetryConfig` or create reusable configs

### ❓ **Problem**: Duplicate error codes

**✅ Solution**: MD5 codes are unique per message + stack trace combination

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