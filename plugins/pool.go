package plugins

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/threatwinds/go-sdk/catcher"
)

// Pool is a generic object pool
type Pool[T any] struct {
	new   func() T    // Function to create new objects
	reset func(obj T) // Function to reset objects before returning to pool
	pool  sync.Pool   // Underlying sync.Pool
}

// NewPool creates a new object pool
func NewPool[T any](newFn func() T, resetFn func(obj T)) *Pool[T] {
	p := &Pool[T]{
		new:   newFn,
		reset: resetFn,
	}

	p.pool = sync.Pool{
		New: func() interface{} {
			return p.new()
		},
	}

	return p
}

// Get retrieves an object from the pool or creates a new one if the pool is empty
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns an object to the pool after resetting it
func (p *Pool[T]) Put(obj T) {
	defer func() {
		if r := recover(); r != nil {
			_ = catcher.Error("panic in pool Put operation", fmt.Errorf("%v", r), map[string]any{
				"operation": "pool.Put",
				"poolType":  reflect.TypeFor[T]().String(),
			})
		}
	}()

	// Check if obj is nil using reflection for generic types
	objValue := reflect.ValueOf(obj)
	if !objValue.IsValid() || (objValue.Kind() == reflect.Ptr && objValue.IsNil()) {
		_ = catcher.Error("cannot put nil object to pool", nil, nil)
		return
	}

	p.reset(obj)
	p.pool.Put(obj)
}
