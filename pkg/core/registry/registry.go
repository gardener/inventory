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

// NewRegistry creates a new empty registry.
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

// MustRegister registers the tasks, or panics in case of errors.
func (r *Registry) MustRegister(name string, handler asynq.Handler) {
	if err := r.Register(name, handler); err != nil {
		panic(err)
	}
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

// Register registers the given task and handler in the default registry.
func Register(name string, handler asynq.Handler) error {
	return DefaultRegistry.Register(name, handler)
}

// Unregister unregisters the given task from the default registry.
func Unregister(name string) {
	DefaultRegistry.Unregister(name)
}

// MustRegister registers the given task and handler in the default registry.
func MustRegister(name string, handler asynq.Handler) {
	DefaultRegistry.MustRegister(name, handler)
}

// Get returns the task handler with the given name from the default registry.
func Get(name string) (asynq.Handler, bool) {
	return DefaultRegistry.Get(name)
}

// Range iterates over the items from the default registry and calls f for each
// item. If f returns a non-nil error Range will stop the iteration.
func Range(f RangeFunc) {
	DefaultRegistry.Range(f)
}
