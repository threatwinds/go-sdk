package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return New(Config{Addr: mr.Addr()})
}

func TestNilClientIsASafePermanentMiss(t *testing.T) {
	var c *Client
	ctx := context.Background()

	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("Get on a nil Client reported a hit")
	}
	c.Set(ctx, "k", "v", time.Minute) // must not panic
	claimed, current, err := c.Claim(ctx, "k", "v", time.Minute)
	if err != nil || claimed || current != "" {
		t.Fatalf("Claim on a nil Client = (%v, %q, %v), want (false, \"\", nil)", claimed, current, err)
	}
	c.Touch(ctx, "k", time.Minute) // must not panic
	if err := c.Close(); err != nil {
		t.Fatalf("Close on a nil Client: %v", err)
	}
}

func TestGetMissesUntilSet(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	if _, ok := c.Get(ctx, "group:a"); ok {
		t.Fatal("Get found a key nobody set")
	}

	c.Set(ctx, "group:a", "alert-1", time.Minute)

	v, ok := c.Get(ctx, "group:a")
	if !ok || v != "alert-1" {
		t.Fatalf("Get = (%q, %v), want (\"alert-1\", true)", v, ok)
	}
}

// This is the property the whole package exists for: two callers racing over
// the same key must not both win. Get-then-Set cannot guarantee that — only
// an atomic claim can.
func TestClaimPicksExactlyOneWinner(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	claimed1, current1, err := c.Claim(ctx, "group:a", "alert-1", time.Minute)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if !claimed1 || current1 != "alert-1" {
		t.Fatalf("first claim = (%v, %q), want (true, \"alert-1\")", claimed1, current1)
	}

	claimed2, current2, err := c.Claim(ctx, "group:a", "alert-2", time.Minute)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if claimed2 {
		t.Fatal("second claim won a key the first claim already holds")
	}
	if current2 != "alert-1" {
		t.Fatalf("second claim's current = %q, want the first winner's value %q", current2, "alert-1")
	}
}

func TestTouchExtendsTheTTLWithoutChangingTheValue(t *testing.T) {
	mr := miniredis.RunT(t)
	c := New(Config{Addr: mr.Addr()})
	ctx := context.Background()

	c.Set(ctx, "group:a", "alert-1", 2*time.Second)
	mr.FastForward(1 * time.Second)

	c.Touch(ctx, "group:a", time.Minute)
	mr.FastForward(5 * time.Second) // past the original 2s TTL

	v, ok := c.Get(ctx, "group:a")
	if !ok || v != "alert-1" {
		t.Fatalf("Get after Touch = (%q, %v), want the entry still alive with its original value", v, ok)
	}
}

// This is the scenario the whole package was built for: a burst of truly
// concurrent callers (a real race, not a sequential approximation of one)
// all trying to become the root of the same alert group. Exactly one may win.
func TestClaimUnderRealConcurrencyStillPicksExactlyOneWinner(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	const callers = 50
	var wins int64
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func(i int) {
			defer wg.Done()
			claimed, _, err := c.Claim(ctx, "group:burst", fmt.Sprintf("alert-%d", i), time.Minute)
			if err != nil {
				t.Errorf("caller %d: %v", i, err)
				return
			}
			if claimed {
				atomic.AddInt64(&wins, 1)
			}
		}(i)
	}
	wg.Wait()

	if wins != 1 {
		t.Fatalf("%d of %d concurrent callers won the claim, want exactly 1", wins, callers)
	}
}

func TestClaimOnAnExpiredKeyClaimsAgain(t *testing.T) {
	mr := miniredis.RunT(t)
	c := New(Config{Addr: mr.Addr()})
	ctx := context.Background()

	claimed1, _, err := c.Claim(ctx, "group:a", "alert-1", time.Second)
	if err != nil || !claimed1 {
		t.Fatalf("first claim = (%v, %v), want (true, nil)", claimed1, err)
	}

	mr.FastForward(2 * time.Second)

	claimed2, current2, err := c.Claim(ctx, "group:a", "alert-2", time.Minute)
	if err != nil {
		t.Fatalf("claim after expiry: %v", err)
	}
	if !claimed2 || current2 != "alert-2" {
		t.Fatalf("claim after expiry = (%v, %q), want (true, \"alert-2\")", claimed2, current2)
	}
}
