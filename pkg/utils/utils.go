// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

// GroupBy groups the given slice of items using a function which provides a
// key, based on which the items will be grouped.
func GroupBy[K comparable, V any](items []V, keyFunc func(item V) K) map[K][]V {
	result := make(map[K][]V)
	for _, item := range items {
		key := keyFunc(item)
		result[key] = append(result[key], item)
	}

	return result
}
