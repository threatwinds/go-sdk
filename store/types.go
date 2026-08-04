package store

import (
	"encoding/json"
	"time"
)

// Dataset names a family of records. The values are defined by the application,
// not here: what datasets exist and how they map to physical storage is product
// knowledge, and a driver receives that mapping as configuration.
type Dataset string

// Scope narrows an operation. Tenant is mandatory and drivers inject it
// themselves: empty is an error, not "every tenant".
type Scope struct {
	Tenant   string
	Dataset  Dataset
	DataType string
	From, To time.Time
}

// AllTenants is the only way to read across tenants, so that doing so is
// explicit and greppable.
const AllTenants = "*"

type Op string

const (
	OpEq          Op = "eq"
	OpNotEq       Op = "not_eq"
	OpIn          Op = "in"
	OpNotIn       Op = "not_in"
	OpGt          Op = "gt"
	OpGte         Op = "gte"
	OpLt          Op = "lt"
	OpLte         Op = "lte"
	OpBetween     Op = "between" // Value is [2]any{lo, hi}
	OpContains    Op = "contains"
	OpNotContains Op = "not_contains"
	OpExists      Op = "exists" // Value ignored
)

// Filter is one predicate. Filters in a slice are ANDed; there is no OR and no
// nesting, so that a driver never has to implement a boolean evaluator.
type Filter struct {
	Field string
	Op    Op
	Value any
}

type SortOrder string

const (
	Asc  SortOrder = "asc"
	Desc SortOrder = "desc"
)

type Page struct {
	Offset int
	Limit  int
	SortBy string
	Order  SortOrder
}

type CalendarUnit string

const (
	CalendarNone  CalendarUnit = ""
	CalendarHour  CalendarUnit = "hour"
	CalendarDay   CalendarUnit = "day"
	CalendarWeek  CalendarUnit = "week"
	CalendarMonth CalendarUnit = "month"
)

// Interval buckets a timeline. Calendar takes precedence over Duration.
type Interval struct {
	Duration time.Duration
	Calendar CalendarUnit
}

// Field describes one queryable field of a dataset.
type Field struct {
	Name       string
	Type       string
	Filterable bool // supports exact-match filters
	Searchable bool // supports text matching
	Conflict   bool // type differs across the underlying storage
}

type Bucket struct {
	Key   string
	Count int64
}

type Point struct {
	At    time.Time
	Count int64
}

type Series struct {
	Key    string
	Points []Point
}

type GroupOpts struct {
	Limit   int
	TopHits int
	SortBy  string
	Order   SortOrder
}

type Group struct {
	Field    string
	Key      string
	Count    int64
	Children []Group
	Hits     []json.RawMessage
}

type DatasetUsage struct {
	Dataset   Dataset
	DataType  string
	Documents int64
	Bytes     int64
	Oldest    time.Time
	Newest    time.Time
}

type Retention struct {
	Keep      time.Duration
	ColdAfter time.Duration
}

func (r Retention) Tiered() bool { return r.ColdAfter > 0 }

type HealthStatus string

const (
	HealthOK          HealthStatus = "ok"
	HealthDegraded    HealthStatus = "degraded"
	HealthUnavailable HealthStatus = "unavailable"
)

type Health struct {
	Status      HealthStatus
	Message     string
	DiskUsedPct float64
}
