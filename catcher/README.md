# ThreatWinds Catcher - Sistema de Manejo de Errores y Retry

Sistema avanzado de manejo de errores y operaciones de retry para las APIs de ThreatWinds, migrado y mejorado desde el
sistema logger.

## 🎯 Características

- 🔧 **Manejo de errores robusto** con stack traces completos y códigos únicos
- 🔄 **Sistema de retry avanzado** con backoff exponencial y configuración granular
- 🏷️ **Metadatos enriquecidos** para mejor debugging y monitoreo
- 🔗 **Integración nativa** con Gin framework y HTTP status codes
- 🎯 **Solo registra errores** - no hace logging de operaciones exitosas
- ⬆️ **Migración sencilla** desde el sistema logger con compatibilidad hacia atrás

## 📦 Instalación

```bash
go get github.com/threatwinds/go-sdk/catcher
```

## 🚀 Inicio Rápido

### Manejo Básico de Errores

```go
package main

import (
    "errors"
    "github.com/threatwinds/go-sdk/catcher"
)

func main() {
    // Crear un error enriquecido
    err := catcher.Error("database operation failed", 
        errors.New("connection timeout"), 
        map[string]any{
            "operation": "insert",
            "table": "users",
            "status": 500,
        })
    
    // El error se registra automáticamente
    // Salida: {"code":"abc123...", "trace":[...], "msg":"database operation failed", ...}
}
```

### Retry Básico

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
return nil
}, config, "authentication_failed")
}
```

## 🔄 Migración desde Logger

### Antes (Logger)

```go
package helpers

import "github.com/threatwinds/logger"

var Logger = logger.NewLogger(&logger.Config{
	Retries: 30,
	Wait:    60 * time.Second,
})

func processData() error {
	return Logger.Retry(func() error {
		return operation()
	}, "not_found")
}
```

### Después (Catcher)

```go
package helpers

import "github.com/threatwinds/go-sdk/catcher"

var RetryConfig = &catcher.RetryConfig{
	MaxRetries: 30,
	WaitTime:   60 * time.Second,
}

func processData() error {
	return catcher.Retry(func() error {
		err := operation()
		if err != nil {
			return catcher.Error("operation failed", err, map[string]any{
				"operation": "processData",
				"status":    500,
			})
		}
		return nil
	}, RetryConfig, "not_found")
}
```

### Migración Rápida (Compatibilidad)

```go
// Migración mínima - mantén la misma signatura
func processDataQuick() error {
return catcher.RetryLegacy(func () error {
return operation()
}, 30, 60*time.Second, "not_found")
}
```

## ⚙️ Configuración de Retry

```go
type RetryConfig struct {
MaxRetries int           // Número máximo de reintentos (0 = infinito)
WaitTime   time.Duration // Tiempo de espera entre reintentos
}

// Configuración por defecto
var DefaultRetryConfig = &RetryConfig{
MaxRetries: 5,
WaitTime:   1 * time.Second,
}
```

## 🔧 Funciones de Retry Disponibles

### 1. `Retry` - Retry con límite máximo

```go
err := catcher.Retry(func () error {
return performOperation()
}, config, "exception1", "exception2")
```

### 2. `InfiniteRetry` - Retry infinito hasta éxito o excepción

```go
err := catcher.InfiniteRetry(func () error {
return connectToDatabase()
}, config, "auth_failed")
```

### 3. `InfiniteLoop` - Loop infinito hasta excepción

```go
catcher.InfiniteLoop(func () error {
return processMessages()
}, config, "shutdown_signal")
```

### 4. `InfiniteRetryIfXError` - Retry solo en error específico

```go
err := catcher.InfiniteRetryIfXError(func () error {
return connectToService()
}, config, "connection_timeout")
```

### 5. `RetryWithBackoff` - Retry con backoff exponencial

```go
err := catcher.RetryWithBackoff(func () error {
return callExternalAPI()
}, config,
30*time.Second, // max backoff
2.0, // multiplier
"rate_limited")
```

## 🔍 Manejo de Errores

### Crear Errores Enriquecidos

```go
// Error básico
err := catcher.Error("operation failed", originalErr, map[string]any{
"user_id": "123",
"status": 500,
})

// Error para operaciones de base de datos
err := catcher.Error("database query failed", dbErr, map[string]any{
"query": "SELECT * FROM users",
"table": "users",
"operation": "select",
"status": 500,
"retry_able": true,
})

// Error para APIs externas
err := catcher.Error("external API call failed", apiErr, map[string]any{
"service": "payment_processor",
"endpoint": "/api/v1/charge",
"method": "POST",
"status": 502,
"external": true,
})
```

### Verificar Tipos de Error

```go
// Verificación básica de excepciones
if catcher.IsException(err, "not_found", "forbidden") {
// Manejar excepción específica
}

// Verificación avanzada para SdkError
if sdkErr := catcher.ToSdkError(err); sdkErr != nil {
// Acceder a metadata del error
if operation, ok := sdkErr.Args["operation"]; ok {
log.Printf("Failed operation: %s", operation)
}

// Verificar excepciones en SdkError
if catcher.IsSdkException(sdkErr, "timeout") {
// Manejar timeout específicamente
}
}
```

## 🌐 Integración con Gin

```go
func handleRequest(c *gin.Context) {
err := performOperation()
if err != nil {
// Si es un SdkError, se enviará automáticamente con headers apropiados
if sdkErr := catcher.ToSdkError(err); sdkErr != nil {
sdkErr.GinError(c)
return
}

// Para otros errores, crear SdkError
sdkErr := catcher.Error("request failed", err, map[string]any{
"status": 500,
"request_id": c.GetHeader("X-Request-ID"),
})
sdkErr.GinError(c)
}
}
```

## 📋 Ejemplos Prácticos

### Operación de Base de Datos

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

### Conectar a Servicio Externo

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
return nil
}, &catcher.RetryConfig{
WaitTime:  5 * time.Second,
}, "connection_refused")
}
```

### Procesar Cola de Mensajes

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
}
}

return nil
}, &catcher.RetryConfig{
WaitTime:  1 * time.Second,
}, "shutdown")
}
```

## 📊 Logging y Monitoreo

### Estructura de Logs

```json
{
  "code": "a1b2c3d4e5f6789...",
  "trace": [
    "main.processData 123",
    "catcher.Retry 45"
  ],
  "msg": "operation failed",
  "cause": "connection timeout",
  "args": {
    "operation": "fetchData",
    "status": 500,
    "retries_attempted": 3,
    "max_retries": 5
  }
}
```

### Logs de Retry

El sistema automáticamente registra:

- ✅ **Inicio de retry** con configuración
- 🔄 **Intentos fallidos** con detalles del error
- ✅ **Éxito después de reintentos**
- ❌ **Fallo final** después de máximo de reintentos
- 🛑 **Parada por excepción**

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

## 🔧 Funciones de Compatibilidad

Para migración gradual, usa las funciones legacy:

```go
// Compatibilidad directa con logger.Retry
err := catcher.RetryLegacy(func () error {
return operation()
}, 30, 60*time.Second, "exception1", "exception2")

// Compatibilidad con logger.InfiniteRetry
err := catcher.InfiniteRetryLegacy(func () error {
return operation()
}, 60*time.Second, "exception1")

// Compatibilidad con logger.InfiniteRetryIfXError
err := catcher.InfiniteRetryIfXErrorLegacy(func () error {
return operation()
}, 60*time.Second, "specific_error")
```

## 📈 Beneficios del Sistema Catcher

1. **🔍 Mejor Debugging**: Stack traces completos y códigos únicos de error
2. **📊 Monitoreo Avanzado**: Metadata rica para alertas y métricas
3. **⚙️ Flexibilidad**: Configuración granular de retry por operación
4. **🚀 Performance**: Backoff exponencial para servicios externos
5. **🛠️ Mantenibilidad**: Separación clara entre logging y retry logic
6. **🔗 Integración**: Soporte nativo para frameworks web

## 📚 Referencias

- [Guía de Migración Completa](./RETRY_MIGRATION.md)
- [Funciones de Compatibilidad](./migration.go)
- [Tests de Ejemplo](./integration_test.go)
- [Tests Unitarios](./retry_test.go)

## 🆘 Troubleshooting

### ❓ **Problema**: ¿Por qué no veo logs de retry exitosos?

**✅ Solución**: Esto es intencional - catcher solo registra errores reales, no operaciones exitosas

### ❓ **Problema**: Migración gradual necesaria

**✅ Solución**: Usar funciones `*Legacy` para compatibilidad inmediata

### ❓ **Problema**: Configuración compleja

**✅ Solución**: Usar `catcher.DefaultRetryConfig` o crear configs reutilizables

### ❓ **Problema**: Error codes duplicados

**✅ Solución**: Los códigos MD5 son únicos por combinación de mensaje + stack trace

---

## 💡 Tips y Mejores Prácticas

1. **Usa metadata descriptiva** en tus errores para mejor debugging
2. **Configura retry strategies** específicas por tipo de operación
3. **Evita retry infinito** en operaciones críticas de tiempo
4. **Usa backoff exponencial** para servicios externos
5. **Agrupa configuraciones** por dominio de aplicación (DB, API, etc.)

¡El sistema catcher está listo para mejorar la robustez y observabilidad de tus aplicaciones ThreatWinds! 🚀