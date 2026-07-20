# ThreatWinds Entities - Data Validation and Schema Definitions

This package provides a comprehensive set of tools for data validation, schema definitions, and entity management within the ThreatWinds ecosystem.

## 🎯 Features

- ✅ **Extensive Data Validation**: Support for dozens of data types including IP, FQDN, Email, CIDR, Hashes (MD5, SHA1, SHA256, etc.), URLs, MAC addresses, and more.
- 🏗️ **Structured Schemas**: Definitions for Consolidated Entities, Entity History, Relations, and Comments.
- 🔍 **Universal Validator**: A single `ValidateValue` function that routes to the appropriate validator based on type definitions.
- 🏷️ **Metadata Support**: Flexible attribute management and tagging for entities.
- 🔗 **Relationship Management**: Support for entity associations and aggregations.

## 📦 Installation

```bash
go get github.com/threatwinds/go-sdk/entities
```

## 🚀 Quick Start

### Validating a Value

```go
package main

import (
    "fmt"
    "github.com/threatwinds/go-sdk/entities"
)

func main() {
    value := "8.8.8.8"
    typeStr := "ip"
    
    validatedValue, hash, err := entities.ValidateValue(value, typeStr)
    if err != nil {
        fmt.Printf("Validation failed: %v\n", err)
        return
    }
    
    fmt.Printf("Validated Value: %v\n", validatedValue)
    fmt.Printf("SHA3-256 Hash: %s\n", hash)
}
```

### Using Entity Schemas

```go
package main

import (
    "github.com/threatwinds/go-sdk/entities"
)

func main() {
    entity := entities.Entity{
        Type: "ip",
        Attributes: entities.Attributes{
            // Add attributes here
        },
        Reputation: -1,
        Tags: []string{"dns-server"},
        VisibleBy: []string{"public"},
    }
    // Process entity...
}
```

## 🛠️ Supported Data Types

The package includes validators for:

- **Network**: IP (IPv4/IPv6), CIDR, FQDN, MAC, Port.
- **Hashes**: MD5, SHA1, SHA224, SHA256, SHA384, SHA512, SHA3-224, SHA3-256, SHA3-384, SHA3-512, SHA512-224, SHA512-256.
- **Identity/System**: Email, URL, UUID, Path, UserID, Adversary, Identifier, Regex.
- **Geographic**: City, Country.
- **Common**: String, Case-Insensitive String (ISTR), Integer, Float, Boolean, Date, Datetime, Hexadecimal, Base64, MIME, Phone.

## 📝 Schema Definitions

### EntityConsolidated
Represents the current state of an entity with its reputation and accuracy.

### EntityHistory
Tracks changes to an entity over time.

### RelationConsolidated & RelationHistory
Manage relationships between different entities.

### Comment
Allows adding comments and threaded discussions to entities.

## 🤝 Contribution

Contributions are welcome! Please feel free to submit a Pull Request.
