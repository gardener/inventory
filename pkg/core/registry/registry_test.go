// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gardener/inventory/pkg/core/registry"
)

func TestRegistryLength(t *testing.T) {
	testCases := []struct {
		desc  string
		items map[string]string
	}{
		{
			desc:  "empty registry",
			items: map[string]string{},
		},
		{
			desc:  "non-empty registry",
			items: map[string]string{"foo": "bar", "bar": "baz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			r := registry.New[string, string]()
			for k, v := range tc.items {
				r.MustRegister(k, v)
			}
			if r.Length() != len(tc.items) {
				t.Fatalf("want len %d, got len %d", len(tc.items), r.Length())
			}
		})
	}
}

func TestRegistryGet(t *testing.T) {
	r := registry.New[string, int]()

	key := "key"
	value := 42

	r.MustRegister(key, value)
	gotValue, exists := r.Get(key)
	if !exists {
		t.Fatalf("expected value %v not found", value)
	}

	if gotValue != value {
		t.Fatalf("want value %v, got value %v", value, gotValue)
	}
}

func TestRegistryRegister(t *testing.T) {
	testCaseItems := []map[string]string{
		{},
		{"foo": "bar"},
		{"bar": "baz", "baz": "qux"},
	}

	for _, tci := range testCaseItems {
		r := registry.New[string, string]()
		t.Run(fmt.Sprintf("registry with %d items", len(tci)), func(t *testing.T) {
			for k, v := range tci {
				// First time registering it should succeed
				if err := r.Register(k, v); err != nil {
					t.Fatalf("expected nil error on Register(), got %v", err)
				}
				// Second time registering the same K/V should result in
				// ErrKeyAlreadyExists error
				if err := r.Register(k, v); !errors.Is(err, registry.ErrKeyAlreadyRegistered) {
					t.Fatalf("expected ErrKeyAlreadyExists error, got %v", err)
				}
			}
		})
	}
}

func TestRegistryUnregister(t *testing.T) {
	testCaseItems := []map[string]string{
		{},
		{"foo": "bar"},
		{"bar": "baz", "baz": "qux"},
	}

	for _, tci := range testCaseItems {
		r := registry.New[string, string]()
		t.Run(fmt.Sprintf("registry with %d items", len(tci)), func(t *testing.T) {
			for k, v := range tci {
				r.MustRegister(k, v)
				r.Unregister(k)
			}
			if r.Length() != 0 {
				t.Fatal("registry length must be 0")
			}
		})
	}
}

func TestRegistryMustRegister(t *testing.T) {
	r := registry.New[string, string]()
	r.MustRegister("foo", "bar")

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected MustRegister() to panic")
		}
	}()

	r.MustRegister("foo", "qux")
}

func TestRegistryRange(t *testing.T) {
	r := registry.New[string, string]()
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
			walker:  func(_, _ string) error { return registry.ErrStopIteration },
		},
		{
			desc:    "returns nil on success",
			wantErr: nil,
			walker:  func(_, _ string) error { return nil },
		},
		{
			desc:    "returns nil on ErrContinue",
			wantErr: nil,
			walker:  func(_, _ string) error { return registry.ErrContinue },
		},
		{
			desc:    "propagates error back to caller",
			wantErr: dummyErr,
			walker:  func(_, _ string) error { return dummyErr },
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

func TestRegistryOverwrite(t *testing.T) {
	testCases := []struct {
		key  string
		val1 string
		val2 string
	}{
		{
			key:  "foo",
			val1: "foo",
			val2: "bar",
		},
		{
			key:  "bar",
			val1: "bar",
			val2: "qux",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("test overwrite with key %s", tc.key), func(t *testing.T) {
			r := registry.New[string, string]()
			// Set initial key value
			r.MustRegister(tc.key, tc.val1)
			gotVal1, ok := r.Get(tc.key)
			if !ok {
				t.Fatalf("value for key %q not found", tc.key)
			}
			if gotVal1 != tc.val1 {
				t.Fatalf("got value %q for %q, want %q", gotVal1, tc.key, tc.val1)
			}
			// Overwrite value
			r.Overwrite(tc.key, tc.val2)
			gotVal2, ok := r.Get(tc.key)
			if !ok {
				t.Fatalf("value for key %q not found", tc.key)
			}
			if gotVal2 != tc.val2 {
				t.Fatalf("got value %q for %q, want %q", gotVal2, tc.key, tc.val2)
			}
		})
	}
}
