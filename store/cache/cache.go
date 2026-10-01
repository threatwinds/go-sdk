// Package cache is a thin, nil-safe wrapper around a shared Redis/Valkey
// instance. It exists for the narrow case a dataset store is the wrong tool
// for: deciding, among several concurrent callers racing over the same key,
// which one goes first — something a MergeTree-backed store cannot answer
// quickly, because a row it was just asked to write is not guaranteed
// queryable back the instant the write call returns.
//
// Every method is a safe no-op (a permanent miss) on a nil *Client, so a
// deployment that has not configured Redis keeps working exactly as it did
// before this package existed: callers fall through to their real source of
// truth every time.
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type Client struct {
	rdb *redis.Client
}

// New returns nil when Addr is empty.
func New(cfg Config) *Client {
	if cfg.Addr == "" {
		return nil
	}
	return &Client{rdb: redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})}
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	return c.rdb.Close()
}

// Get returns the cached value and whether it was present. A nil Client, a
// connection error, and an actual miss all read the same to the caller:
// false — there is no real-time-vs-expired distinction a check-then-act
// caller needs to make, since all three mean "consult the real source of
// truth instead."
func (c *Client) Get(ctx context.Context, key string) (string, bool) {
	if c == nil {
		return "", false
	}
	v, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

// Set stores value under key for ttl. Best-effort: a write failure here just
// means the next Get falls through to the real source of truth again, same
// as if the key had never been set.
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) {
	if c == nil {
		return
	}
	_ = c.rdb.Set(ctx, key, value, ttl).Err()
}

// Claim atomically sets key to value only if key does not already exist —
// the primitive a caller needs to pick exactly one winner among several
// concurrent, otherwise-identical attempts, which Get-then-Set cannot do
// because two callers can both Get a miss before either Sets.
//
// claimed is true when this call is the one that set it. When it is false,
// current is whatever value is there now — either another concurrent
// caller's winning claim, or one set earlier by Set — and the caller should
// treat that as the answer rather than as a reason to claim again.
//
// A nil Client always reports an unclaimed miss (claimed=false, current="",
// err=nil): without a real cache behind it there is no winner to pick, and
// the caller's ordinary fallback path is what decides instead.
func (c *Client) Claim(ctx context.Context, key, value string, ttl time.Duration) (claimed bool, current string, err error) {
	if c == nil {
		return false, "", nil
	}
	ok, err := c.rdb.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, "", err
	}
	if ok {
		return true, value, nil
	}
	v, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		// The key we just lost a race for is already gone again (expired, or
		// raced a second time) — reading that as a plain miss is safer than
		// surfacing an error for what is actually just contention.
		return false, "", nil
	}
	return false, v, nil
}

// Touch resets key's TTL without changing its value, so a group still
// receiving events keeps its fast-path entry alive instead of falling out of
// cache mid-burst and forcing every caller back to the real source of truth.
func (c *Client) Touch(ctx context.Context, key string, ttl time.Duration) {
	if c == nil {
		return
	}
	_ = c.rdb.Expire(ctx, key, ttl).Err()
}
