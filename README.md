# ThreatWinds Go SDK

The official Go SDK for building applications and plugins within the ThreatWinds ecosystem.

## 📦 Modules

This SDK is organized into several modules, each serving a specific purpose:

- [**`catcher`**](./catcher): Robust error handling, structured logging, and advanced retry system.
- [**`entities`**](./entities): Data validation and standard ThreatWinds schema definitions.
- [**`os`**](./os): Simplified OpenSearch client with fluent query builders and group-based access control.
- [**`plugins`**](./plugins): Core infrastructure for developing Analysis, Notification, Parsing, and Correlation plugins.
- [**`utils`**](./utils): Common helper functions for type casting, I/O, HTTP requests, and more.

## 🚀 Installation

To use the SDK in your project:

```bash
go get github.com/threatwinds/go-sdk
```

Or install specific modules:

```bash
go get github.com/threatwinds/go-sdk/catcher
go get github.com/threatwinds/go-sdk/os
# etc.
```

## 🛠️ Usage

Refer to the individual module directories for detailed documentation and examples.

## 🤝 Contribution

Contributions are welcome! Please feel free to submit a Pull Request.

## 📜 License

This project is licensed under the MIT License.
