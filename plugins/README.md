# ThreatWinds Plugins SDK

This package provides the core infrastructure for developing ThreatWinds plugins. It includes support for various plugin types, configuration management, and communication between components using gRPC over UNIX sockets.

## 🎯 Features

- 🧩 **Multiple Plugin Types**: Easy-to-use initializers for Analysis, Notification, and Input plugins.
- 🚀 **gRPC Infrastructure**: Built-in gRPC server/client management for inter-plugin communication.
- ⚙️ **Dynamic Configuration**: Shared configuration system with locking mechanisms for synchronized updates.
- 📊 **CEL Expression Evaluation**: Support for Common Expression Language (CEL) with optimized caching for high-performance data processing.
- 📬 **Asynchronous Communication**: Channel-based log and notification enqueuing to minimize processing latency.
- 🔒 **Lifecycle Management**: Automatic UNIX socket cleanup and graceful shutdown handling.

## 📦 Installation

```bash
go get github.com/threatwinds/go-sdk/plugins
```

## 🚀 Quick Start

### Creating an Analysis Plugin

```go
package main

import (
    "github.com/threatwinds/go-sdk/plugins"
)

func main() {
    err := plugins.InitAnalysisPlugin("my-plugin", func(event *plugins.Event, srv plugins.Analysis_AnalyzeServer) error {
        // Your analysis logic here
        return nil
    })
    if err != nil {
        // Handle error
    }
}
```

### Sending Logs from an Input Plugin

```go
package main

import (
    "github.com/threatwinds/go-sdk/plugins"
)

func main() {
    // Start the background sender
    go plugins.SendLogsFromChannel()
    
    // Enqueue a log
    log := &plugins.Log{ /* ... */ }
    err := plugins.EnqueueLog(log)
}
```

## 🛠️ Key Components

### Plugin Initialization
- `InitAnalysisPlugin`: For plugins that analyze incoming events.
- `InitNotificationPlugin`: For plugins that handle outgoing notifications.
- `InitParsingPlugin`: For plugins that transform raw logs into structured drafts.
- `InitCorrelationPlugin`: For plugins that correlate multiple events into new insights.
- `SendLogsFromChannel`: For input plugins to send logs to the engine.

### Configuration (`GetCfg`, `PluginCfg`)
Access shared configuration and plugin-specific settings. The system handles file-based persistence and synchronized updates.

### CEL Caching (`CELCache`)
Highly efficient CEL expression evaluation with:
- LRU caching of compiled programs for high performance.
- Automatic expression transformation (e.g., handling keyword fields).
- Safe concurrent access with granular locking.
- Support for custom CEL environment options and overloads.

### Communication
Most plugins communicate via UNIX sockets located in the `sockets` directory within the plugin's working directory.

## 🤝 Contribution

Contributions are welcome! Please feel free to submit a Pull Request.
