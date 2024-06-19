package registry

import (
    "testing"
)

func TestRegistryLengthAfterAdd(t *testing.T) {
    registry := New[string, int]()

    registry.Register("key", 1)

    if registry.Length() != 1 {
        t.Fatalf("Adding one key/value pair to a new registry results in length different than 1.")
    }
}

func TestNewRegistryLength(t *testing.T) {
    registry := New[string, int]()

    if registry.Length() != 0 {
        t.Fatalf("New registry must have a length of 0.")
    }
}

func TestUnregisterReducesLength(t *testing.T) {
    registry := New[string, int]()

    key := "key"
    registry.Register(key, 1)
    registry.Unregister(key)

    if registry.Length() != 0 {
        t.Fatalf("After registering and unregistering a single item, registry must have a length of 0.")
    }
}

func TestMustRegisterPanicsOnDuplicateKey(t *testing.T) {
    registry := New[string, int]()

    key := "key"
    registry.Register(key, 1)

    defer func() {
        if r := recover(); r == nil {
            t.Fatalf("MustRegister did not panic when registering duplicate key.")
        }
    }()

    registry.MustRegister(key, 1)
}

func TestRangeStopsOnError(t *testing.T) {
    registry := New[string, int]()
    registry.Register("key", 1)
    
    rangeFunc := func(key string, val int) error {
        return ErrStopIteration
    }

    out := registry.Range(rangeFunc)

    if out != nil {
        t.Fatalf("Range didn't explicitly stop at ErrStopIteration error.")
    }
}
