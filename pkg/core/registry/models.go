// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

// ModelRegistry is the default registry for models.
var ModelRegistry = New[string, any]()
