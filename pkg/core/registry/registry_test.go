// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"errors"
	"testing"
)

func TestRegistryLengthAfterAdd(t *testing.T) {
	registry := New[string, int]()

	registry.Register("key", 1)

	if registry.Length() != 1 {
		t.Fatalf("Adding one key/value pair to a new registry results in length different than 1.")
	}
}

func TestRegistryGetAfterAdd(t *testing.T) {
	registry := New[string, int]()

	const key = "key"
	const value = 42

	registry.Register(key, value)

	outValue, exists := registry.Get(key)
	if !exists {
		t.Fatalf("No value found for registered key %q", key)
	}

	if outValue != value {
		t.Fatalf("Registry returned value %q, expected %q.", outValue, value)
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

func TestRangeReturnNilOnNoError(t *testing.T) {
	registry := New[string, int]()
	registry.Register("key", 1)

	rangeFunc := func(key string, val int) error {
		return nil
	}

	out := registry.Range(rangeFunc)

	if out != nil {
		t.Fatalf("Range didn't return nil when no errors were encounted.")
	}
}

func TestRangePassesError(t *testing.T) {
	registry := New[string, int]()
	registry.Register("key", 1)

	err := errors.New("custom error")

	rangeFunc := func(key string, val int) error {
		return err
	}

	out := registry.Range(rangeFunc)

	if out != err {
		t.Fatalf("Range encountered an error and didn't return it.")
	}
}
