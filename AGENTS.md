# go-sdk — Core Go SDK

Core utilities, error handling, OpenSearch wrapper, and data validation shared across all ThreatWinds services.

**Module:** `github.com/threatwinds/go-sdk`
**Parent:** See [`../AGENTS.md`](../AGENTS.md) for private module setup (`GOPRIVATE`), Go workspace, and org-wide conventions.

## Packages

| Package | Purpose |
|---------|---------|
| `go-sdk/catcher` | Structured error handling, logging, and retry utilities. Used by ALL services. |
| `go-sdk/utils` | Type casts, HTTP requests, JSON/YAML/CSV, file operations, gjson helpers. |
| `go-sdk/entities` | Cyber-threat data types with validation: IP, CIDR, email, SHA, MD5, URL, UUID, MAC, etc. |
| `go-sdk/os` | OpenSearch client wrapper with fluent query builders, bulk ops, and index management. |
| `go-sdk/plugins` | Plugin infrastructure: Analysis, Notification, Parsing, Correlation. CEL expression engine. |
| `go-sdk/client` | Unified ThreatWinds API client with `auth`, `billing`, `compute` sub-clients. |

## Key Types

### `go-sdk/catcher` — **MUST USE IN ALL SERVICES**
- `Error(msg, cause, args map[string]any) *SdkError` — Primary error constructor. **No `Unwrap()`** — `errors.As` cannot traverse.
- `*SdkError` — Rich error: `Msg`, `Code` (MD5 hash), `Trace`, `Args`, `Cause`, `Severity`
  - `(e *SdkError).GinError(c *gin.Context)` — Writes JSON error response with status code
  - `ToSdkError(err error) *SdkError` — Cast to SdkError or nil
- `SdkLog` / `Log(msg, args)` / `Info(msg, args)` — Structured logging
- `Retry(f, config, exceptions...)` / `InfiniteRetry(f, config, exceptions...)` / `RetryWithBackoff(f, config, maxBackoff, multiplier, exceptions...)` — Retry utilities
- `IsException(err, patterns...)` / `IsSdkException(err, patterns...)` — Error matching

**`catcher.Error` is the single error constructor across all TW services.** All error handling flows through it.

### `go-sdk/utils`
- `CastInt(val any) int`, `CastFloat(val any) float64`, `CastString(val any) string` — Type casters
- `Pointer(val) *T` — Generic pointer helper
- `Request(method, url, body, headers)` — HTTP request helper
- `ToJson(val) string`, `FromJson(data, dst)` — Fast JSON (sonic)
- `ToYaml(val)` / `FromYaml(data, dst)` — YAML marshal/unmarshal
- `ToCsv(rows)` / `FromCsv(data)` — CSV helpers
- `ReadFile(path)`, `WriteFile(path, data)` — File I/O
- `ListFiles(dir, ext)`, `EnsureDir(path)` — Directory helpers
- `GJson` helpers: `GetJsonPath(data, path)`, `ForEachJson(data, path, fn)`

### `go-sdk/entities` — Schema Definitions
- `*Schema` — Defines field names, types, and validation rules. Used for CTI data validation.
- `*Validator` — Runs data against a schema. `Validate(obj map[string]any) []error`
- 40+ pre-built field types: `IP`, `CIDR`, `Email`, `URL`, `UUID`, `FQDN`, `MAC`, `Port`, `Country`, `City`, `MIME`, `Base64`, `Hexadecimal`, `MD5`, `SHA1/256/384/512`, `SHA3-*`, `Adversary`, `PhoneNumber`, `Path`, `Boolean`, `Integer`, `Float`, `RegularExpression`
- Each type has `Validate(value any) bool` and `Parse(value any) (parsed, error)`
- `*ObjectField` for nested object validation
- `*AttributesField` for key-value attribute maps

### `go-sdk/os` — OpenSearch Wrapper
- `Connect(nodes, user, password)` — Singleton OpenSearch connection (TLS, skip verify)
- `Search()` — Fluent query builder. `Index()`, `Query()`, `TermQuery()`, `MatchQuery()`, `RangeQuery()`, `BoolQuery()`, `Must()`, `Filter()`, `Should()`, `MustNot()`, `Pagination(offset, size)`, `SortField()`, `HighlightFields()`, `SourceFields()`, `AggField()`, `TermsAgg()`, `Execute()`, `ExecuteTyped[T any]()`, `ExecuteWithTotal()`
- `BulkInsert(index)`, `BulkInsertWithID(index)`, `Delete(index, id)`, `Update(index, id)`, `Upsert(index, id)` — Document ops
- `CreateIndex()`, `UpdateMapping()`, `DeleteIndex()`, `IndexExists()` — Index management
- `CreateAlias(index, alias)`, `UpdateAliases(actions...)`, `DeleteAlias(alias)` — Alias management
- `CreateSnapshot(repository, name, indices...)`, `RestoreSnapshot(repository, name)` — Snapshots
- `ExecuteSQL(query)` — SQL interface
- `GetClusterHealth()` — Cluster status
- `ExecuteRaw(query map[string]any) (*Response, error)` — Raw query for complex DSL

### `go-sdk/plugins` — Plugin Infrastructure
- `*Input` — Plugin input message: `EventType`, `Payload`, `Source`, `ID`
- `GetPluginName(fullPath, sep)` — Extracts plugin name from file path
- CEL (Common Expression Language) integration for dynamic rule evaluation

### `go-sdk/client` — Unified API Client
- `New(opts ...Option) *Client` — Client factory. `WithURL()`, `WithAPIKey()`, `WithBearer()`, `WithTimeout()`, `WithTransport()`, `WithMaxRetries()`

## Testing

```bash
go test ./...              # All unit tests
go test -cover ./...       # With coverage
go test -run TestBulk      # By name pattern
```

`*integration_test.go` files require a running OpenSearch instance. Run with `-tags=integration`.

## Adding a New Package or Function

1. Create directory `go-sdk/<name>/`
2. Use `package <name>` in all files
3. **All errors must go through `catcher.Error()`**, not bare `fmt.Errorf()` or `errors.New()`
4. Export helpers should be pure or take `context.Context` as first arg
5. Add tests: `<name>_test.go` alongside each source file
6. For entities: add field type implementing `Validate(val any) bool` and `Parse(val any) (any, error)`

## Conventions from Org AGENTS.md

- **Private module** — requires `GOPRIVATE=github.com/threatwinds/*`
- **All Go services depend on this module** — changes here affect everything downstream
