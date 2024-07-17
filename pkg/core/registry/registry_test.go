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

	if err := registry.Register("key", 1); err != nil {
		t.Fatal(err)
	}

	if registry.Length() != 1 {
		t.Fatalf("Adding one key/value pair to a new registry results in length different than 1.")
	}
}

func TestRegistryGetAfterAdd(t *testing.T) {
	registry := New[string, int]()

	const key = "key"
	const value = 42

	if err := registry.Register(key, value); err != nil {
		t.Fatal(err)
	}

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
	if err := registry.Register(key, 1); err != nil {
		t.Fatal(err)
	}

	registry.Unregister(key)

	if registry.Length() != 0 {
		t.Fatalf("After registering and unregistering a single item, registry must have a length of 0.")
	}
}

func TestMustRegisterPanicsOnDuplicateKey(t *testing.T) {
	registry := New[string, int]()

	key := "key"
	if err := registry.Register(key, 1); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("MustRegister did not panic when registering duplicate key.")
		}
	}()

	registry.MustRegister(key, 1)
}

func TestRange(t *testing.T) {
	r := New[string, string]()
	r.MustRegister("foo", "bar")
	r.MustRegister("bar", "baz")
	r.MustRegister("baz", "qux")

	type testCase struct {
		desc    string
		wantErr error
		walker  func(k, v string) error
	}

	dummyErr := errors.New("dummy error")
	testCases := []testCase{
		{
			desc:    "returns nil on ErrStopIteration",
			wantErr: nil,
			walker:  func(k, v string) error { return ErrStopIteration },
		},
		{
			desc:    "returns nil on success",
			wantErr: nil,
			walker:  func(k, v string) error { return nil },
		},
		{
			desc:    "propagates error back to caller",
			wantErr: dummyErr,
			walker:  func(k, v string) error { return dummyErr },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := r.Range(tc.walker)
			if err != tc.wantErr {
				t.Fatalf("want error %v, got error %v", tc.wantErr, err)
			}
		})
	}
}
