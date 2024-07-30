// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ptr

// Value returns the value referenced by p, if p is non-nil, else it returns the
// default value def.
func Value[T any](p *T, def T) T {
	if p != nil {
		return *p
	}

	return def
}
