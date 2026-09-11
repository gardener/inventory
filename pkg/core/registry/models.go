// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package registry

// ModelRegistry is the default registry for models.
var ModelRegistry = New[string, any]()
