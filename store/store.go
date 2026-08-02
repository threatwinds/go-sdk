package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNoTenant    = errors.New("store: scope has no tenant")
	ErrUnsupported = errors.New("store: operation not supported by this driver")
	ErrNotFound    = errors.New("store: record not found")
)

type Store interface {
	Reader
	Writer
	Admin
}

type Reader interface {
	Count(ctx context.Context, s Scope, filters []Filter) (int64, error)

	// CountInWindow counts over a trailing window ending at Scope.To, or at now.
	CountInWindow(ctx context.Context, s Scope, filters []Filter, window time.Duration) (int64, error)

	FetchByID(ctx context.Context, s Scope, id string) (json.RawMessage, error)
	FetchN(ctx context.Context, s Scope, filters []Filter, n int) ([]json.RawMessage, error)
	FetchPage(ctx context.Context, s Scope, filters []Filter, page Page) ([]json.RawMessage, int64, error)
	TopValues(ctx context.Context, s Scope, field string, filters []Filter, n int) ([]Bucket, error)
	Timeline(ctx context.Context, s Scope, filters []Filter, interval Interval) ([]Point, error)
	TimelineByField(ctx context.Context, s Scope, field string, filters []Filter, interval Interval, n int) ([]Series, error)
	GroupBy(ctx context.Context, s Scope, fields []string, filters []Filter, opts GroupOpts) ([]Group, error)

	// DescribeFields lists the queryable fields of a dataset.
	DescribeFields(ctx context.Context, s Scope) ([]Field, error)
}

type Writer interface {
	// Insert generates an id when one is not given; an explicit id means upsert.
	Insert(ctx context.Context, s Scope, id string, doc any) error

	BulkWriter(dataset Dataset) (BulkWriter, error)
	UpdateWhere(ctx context.Context, s Scope, filters []Filter, patch map[string]any) (int64, error)

	// Flush makes prior writes visible to reads. Engines that are already
	// read-your-writes may implement it as a no-op.
	Flush(ctx context.Context, dataset Dataset) error
}

type Admin interface {
	Usage(ctx context.Context) ([]DatasetUsage, error)
	Retention(ctx context.Context, dataset Dataset) (Retention, error)
	SetRetention(ctx context.Context, dataset Dataset, r Retention) error
	Health(ctx context.Context) (Health, error)

	// Drop deletes every record matching the scope, and only those: it is
	// bounded by the scope's tenant like any other operation.
	Drop(ctx context.Context, s Scope) (int64, error)
}

// BulkWriter batches records. Write reports failures instead of dropping them.
type BulkWriter interface {
	Write(s Scope, doc []byte) error
	Flush(ctx context.Context) error
	Close(ctx context.Context) error
}

// RawQuerier carries engine-specific query text. It stays outside Store because
// anything reaching through it is unportable by construction.
type RawQuerier interface {
	Dialect() string
	RawQuery(ctx context.Context, s Scope, query string) ([]map[string]any, error)
}
