// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"errors"
	"fmt"
	"sync"
)

// ErrKeyAlreadyRegistered is returned when attempting to register a key, which
// is already present in the registry.
var ErrKeyAlreadyRegistered = errors.New("key is already registered")

// ErrStopIteration is an error, which is used to stop iterating over the
// registry.
var ErrStopIteration = errors.New("stop iteration")

// ErrContinue is a no-op error, which is used to signal [Registry.Range] to
// continue the iteration.
var ErrContinue = errors.New("continue iteration")

// Registry is a concurrent-safe registry.
type Registry[K comparable, V any] struct {
	mu    sync.Mutex
	items map[K]V
}

// New creates a new empty registry.
func New[K comparable, V any]() *Registry[K, V] {
	r := &Registry[K, V]{
		items: make(map[K]V),
	}

	return r
}

// Register registers the key and value with the registry
func (r *Registry[K, V]) Register(key K, val V) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.items[key]
	if exists {
		return fmt.Errorf("%w: %v", ErrKeyAlreadyRegistered, key)
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
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.items[key]
	if exists {
		delete(r.items, key)
	}
}

// Overwrite replaces the key specified by K with the value V in the registry.
func (r *Registry[K, V]) Overwrite(key K, val V) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items[key] = val
}

// Get returns the value associated with the given key and a boolean indicating
// whether the key is present in the registry.
func (r *Registry[K, V]) Get(key K) (V, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	val, ok := r.items[key]

	return val, ok
}

// Exists returns a boolean indicating whether the given key exists in the
// registry.
func (r *Registry[K, V]) Exists(key K) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.items[key]

	return exists
}

// Length returns the number of items in the registry.
func (r *Registry[K, V]) Length() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.items)
}

// RangeFunc is a function which is called when iterating over the registry
// items. In order to stop iteration callers should return [ErrStopIteration].
type RangeFunc[K comparable, V any] func(key K, val V) error

// Range calls f for each item in the registry. If f returns an error, Range
// will stop the iteration.
func (r *Registry[K, V]) Range(f RangeFunc[K, V]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, v := range r.items {
		err := f(k, v)
		switch err {
		case nil, ErrContinue:
			continue
		case ErrStopIteration:
			return nil
		default:
			return err
		}
	}

	return nil
}
