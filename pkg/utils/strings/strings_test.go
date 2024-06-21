// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package strings

import (
	"testing"
)

func TestStringFromPointer(t *testing.T) {
	emptyString := ""
	nonEmptyString := "abc"
	testCases := []struct {
		in  *string
		out string
	}{
		{nil, ""},
		{&emptyString, ""},
		{&nonEmptyString, nonEmptyString},
	}

	for _, tt := range testCases {
		out := StringFromPointer(tt.in)

		if tt.out != out {
			t.Fatalf(`StringFromPointer(%v) == %q, expected %q.`, tt.in, out, tt.out)
		}
	}
}
