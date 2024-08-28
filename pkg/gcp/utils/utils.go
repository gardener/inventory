// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strings"

	"github.com/gardener/inventory/pkg/gcp/constants"
)

// ProjectFQN returns the full-qualified name for the given project id.
func ProjectFQN(s string) string {
	if strings.HasPrefix(s, constants.ProjectsPrefix) {
		return s
	}

	return fmt.Sprintf("%s%s", constants.ProjectsPrefix, s)
}
