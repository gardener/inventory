package registry

import (
	"errors"
	"fmt"
	"sync"
)

// ErrItemAlreadyRegistered is returned when attempting to register an item,
// which is already present in the registry.
var ErrItemAlreadyRegistered = errors.New("item is already registered")

// ErrStopIteration is an error, which is used to stop iterating over the
// registry.
var ErrStopIteration = errors.New("stop iteration")

// Registry is a concurrent-safe registry.
type Registry[K comparable, V any] struct {
	sync.Mutex
	items map[K]V
}

// New creates a new empty registry.
func New[K comparable, V any]() *Registry[K, V] {
	r := &Registry[K, V]{
		items: make(map[K]V),
	}

	return r
}

// Register registers the task handler with the given name
func (r *Registry[K, V]) Register(key K, val V) error {
	r.Lock()
	defer r.Unlock()

	_, exists := r.items[key]
	if exists {
		return fmt.Errorf("%w: %v", ErrItemAlreadyRegistered, key)
	}

	r.items[key] = val
	return nil
}

// MustRegister registers the key and value, or panics in case of errors.
func (r *Registry[K, V]) MustRegister(key K, val V) {
	if err := r.Register(key, val); err != nil {
		panic(err)
	}
}

// Unregister removes the key (if present) from the registry.
func (r *Registry[K, V]) Unregister(key K) {
	r.Lock()
	defer r.Unlock()

	_, exists := r.items[key]
	if exists {
		delete(r.items, key)
	}
}

// Get returns the value associated with the given key and a boolean indicating
// whether the key is present in the registry.
func (r *Registry[K, V]) Get(key K) (V, bool) {
	r.Lock()
	defer r.Unlock()

	handler, ok := r.items[key]
	return handler, ok
}

// RangeFunc is a function which is called when iterating over the registry.
type RangeFunc[K comparable, V any] func(key K, val V) error

// Range calls f for each key/value pair in the registry. If f returns an error,
// Range will stop the iteration.
func (r *Registry[K, V]) Range(f RangeFunc[K, V]) {
	r.Lock()
	defer r.Unlock()

	for k, v := range r.items {
		err := f(k, v)
		if err != nil {
			return
		}
	}
}
