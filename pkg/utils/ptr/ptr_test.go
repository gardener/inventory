// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ptr_test

import (
	"testing"

	"github.com/gardener/inventory/pkg/utils/ptr"
)

func TestValue(t *testing.T) {
	testStringValue := "value"

	testCases := []struct {
		desc   string
		input  *string
		def    string
		wanted string
	}{
		{
			desc:   "nil input with empty default",
			input:  nil,
			def:    "",
			wanted: "",
		},
		{
			desc:   "nil input with different default",
			input:  nil,
			def:    "def",
			wanted: "def",
		},
		{
			desc:   "normal value, empty default",
			input:  &testStringValue,
			def:    "",
			wanted: testStringValue,
		},
		{
			desc:   "normal value with default",
			input:  &testStringValue,
			def:    "def",
			wanted: testStringValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := ptr.Value(tc.input, tc.def)
			if output != tc.wanted {
				t.Fatalf("want %s got %s", tc.wanted, output)
			}
		})
	}
}
