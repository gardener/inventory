package registry

import (
	"errors"
	"fmt"
	"sync"

	"github.com/hibiken/asynq"
)

// DefaultRegistry is the default task registry.
var DefaultRegistry = NewRegistry()

// ErrTaskAlreadyRegistered is returned when attempting to register a task, for
// which there is an already registered handler.
var ErrTaskAlreadyRegistered = errors.New("task is already registered")

// ErrStopIteration is an error, which is used to stop iterating over the
// registry.
var ErrStopIteration = errors.New("stop iteration")

// Task is the tasks registry
type Registry struct {
	sync.Mutex
	items map[string]asynq.Handler
}

// NewRegistry creates a new emptry registry.
func NewRegistry() *Registry {
	r := &Registry{
		items: make(map[string]asynq.Handler),
	}

	return r
}

// Register registers the task handler with the given name
func (r *Registry) Register(name string, handler asynq.Handler) error {
	r.Lock()
	defer r.Unlock()

	_, exists := r.items[name]
	if exists {
		return fmt.Errorf("%w: %s", ErrTaskAlreadyRegistered, name)
	}

	r.items[name] = handler
	return nil
}

// Unregister removes the handler associated with the given name (if present)
// from the registry.
func (r *Registry) Unregister(name string) {
	r.Lock()
	defer r.Unlock()

	_, exists := r.items[name]
	if exists {
		delete(r.items, name)
	}
}

// Get returns the [asynq.Handler] associated with the given task name and a
// boolean indicating whether the handler is present in the registry.
func (r *Registry) Get(name string) (asynq.Handler, bool) {
	r.Lock()
	defer r.Unlock()

	handler, ok := r.items[name]
	return handler, ok
}

// RangeFunc is a function which is called when iterating over the registry.
type RangeFunc func(name string, handler asynq.Handler) error

// Range calls f for each key/value pair in the registry. If f returns an error,
// Range will stop the iteration.
func (r *Registry) Range(f RangeFunc) {
	r.Lock()
	defer r.Unlock()

	for n, h := range r.items {
		err := f(n, h)
		if err != nil {
			return
		}
	}
}
