# Standard Error Response Body

## Overview

Add a JSON error response body to `SdkError.GinError()` with a standard error format, while preserving the existing `x-error` and `x-error-id` headers.

## Motivation

The current `GinError()` method only sends error information via HTTP headers (`x-error` and `x-error-id`). A JSON error response body allows consumers to parse errors programmatically from the response body rather than from headers.

## Design

### Approach

Modify `SdkError.GinError()` in `catcher/errors.go` to write a JSON response body in addition to the existing headers. Uses `c.AbortJSON()` instead of `c.AbortWithStatus()` to include both the status code and the JSON body with proper `Content-Type: application/json` header.

### New Types

Two unexported structs for serialization, added to `errors.go`:

```go
type errorParam struct {
    Message string `json:"message"`
    Type    string `json:"type"`
    Code    string `json:"code"`
}

type errorResponse struct {
    Error errorParam `json:"error"`
}
```

### Field Mapping

| Response Field | Source              | Notes                           |
|----------------|---------------------|---------------------------------|
| `message`      | `e.SecureString()`  | Same value as `x-error` header  |
| `type`         | `e.Severity`        | Uppercase as-is (e.g. `ERROR`)  |
| `code`         | `e.Code`            | MD5 hash, same as `x-error-id`  |

### Modified Method

```go
func (e SdkError) GinError(c *gin.Context) {
    c.Header("x-error-id", e.Code)

    status, ok := e.Args["status"]
    if !ok {
        c.Header("x-error", e.SecureString())
        c.AbortJSON(http.StatusInternalServerError, errorResponse{
            Error: errorParam{
                Message: e.SecureString(),
                Type:    e.Severity,
                Code:    e.Code,
            },
        })
    } else {
        c.Header("x-error", e.SecureString())
        c.AbortJSON(castInt(status), errorResponse{
            Error: errorParam{
                Message: e.SecureString(),
                Type:    e.Severity,
                Code:    e.Code,
            },
        })
    }
}
```

### Example Response

**400 error:**
```
HTTP/1.1 400 Bad Request
x-error-id: a1b2c3d4e5f6...
x-error: validation failed: missing required field. Args: {}
Content-Type: application/json

{"error":{"message":"validation failed: missing required field. Args: {}","type":"WARNING","code":"a1b2c3d4e5f6..."}}
```

**500 error:**
```
HTTP/1.1 500 Internal Server Error
x-error-id: b7c8d9e0f1a2...
x-error: database connection failed
Content-Type: application/json

{"error":{"message":"database connection failed","type":"ERROR","code":"b7c8d9e0f1a2..."}}
```

### Backward Compatibility

- `x-error` and `x-error-id` headers are unchanged
- HTTP status codes are unchanged
- The JSON body is additive — consumers reading only headers see no change

### Scope

- **File changed**: `catcher/errors.go` (one method modified, two unexported structs added)
- **File changed**: `catcher/errors_test.go` (new tests)
- No other files affected

## Testing

Tests to add in `errors_test.go`:

1. **Response body is valid JSON** — verify `Content-Type: application/json` header is set
2. **Error body structure** — verify `error.message`, `error.type`, `error.code` fields exist
3. **Field mapping** — verify `message` = `SecureString()`, `type` = `Severity`, `code` = SdkError code
4. **Headers preserved** — verify `x-error` and `x-error-id` headers are still populated
5. **Status code with status arg** — verify correct status code when `Args["status"]` is set
6. **Default status code** — verify 500 when no status arg is present

## Error Handling

No new error paths introduced. Serialization involves only strings (zero failure risk). `c.AbortJSON()` handles HTTP response writing.
