// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package strings

// StringFromPointer returns the string value of a pointer to a string or an empty string if the pointer is nil.
func StringFromPointer(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
