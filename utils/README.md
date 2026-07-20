# ThreatWinds Utils - Common Helper Functions

A collection of utility functions used across the ThreatWinds Go SDK and applications, providing simplified operations for type casting, file management, HTTP requests, and more.

## 🎯 Features

- 🔄 **Robust Type Casting**: Safe conversion between common Go types (int64, float64, bool, string) with the `casts` package.
- 🌐 **Simplified HTTP Requests**: Generic-based `DoReq` function for easy API consumption with built-in unmarshalling and error handling.
- 📁 **File & Folder Management**: Helpers for creating, deleting, and listing files and directories.
- 🧬 **JSON & ProtoJSON Helpers**: Simplified serialization and deserialization, including support for Protocol Buffers.
- ⛓️ **Pointer Helpers**: Fast generation of pointers for primitive types.

## 📦 Installation

```bash
go get github.com/threatwinds/go-sdk/utils
```

## 🚀 Quick Start

### Safe Type Casting

```go
package main

import (
    "fmt"
    "github.com/threatwinds/go-sdk/utils"
)

func main() {
    // Cast from interface to int64
    var val interface{} = "123"
    i64 := utils.CastInt64(val)
    fmt.Println(i64) // 123
    
    // Cast from interface to bool
    var bVal interface{} = "true"
    b := utils.CastBool(bVal)
    fmt.Println(b) // true
}
```

### Making an HTTP Request

```go
package main

import (
    "github.com/threatwinds/go-sdk/utils"
)

type MyResponse struct {
    Status string `json:"status"`
}

func main() {
    url := "https://api.example.com/data"
    headers := map[string]string{"Authorization": "Bearer token"}
    
    // DoReq[T] automatically unmarshals JSON response to struct T
    resp, statusCode, err := utils.DoReq[MyResponse](url, nil, "GET", headers)
    if err != nil {
        // Handle error
    }
}
```

## 🛠️ Package Modules

- **`casts.go`**: Safe type conversions from `interface{}` (Int64, Float64, Bool, String).
- **`request.go`**: High-level generic HTTP client with security defaults (TLS 1.2+).
- **`files.go` & `folders.go`**: Basic I/O operations and path manipulation helpers.
- **`json.go` & `protojson.go`**: Simplified wrappers for standard and protobuf JSON.
- **`yaml.go`**: Simplified YAML parsing and marshaling.
- **`gjson.go`**: Helper to get Go values from `gjson.Result`.
- **`csv.go`**: Simplified CSV file reading.
- **`network.go`**: Utilities for IP and network-related operations.
- **`pointers.go`**: Functions like `PointerOf` to easily get addresses of literals.
- **`queue.go`**: Thread-safe generic queue implementation.
- **`retry.go`**: Lightweight retry logic for simple operations.

## 🤝 Contribution

Contributions are welcome! Please feel free to submit a Pull Request.
