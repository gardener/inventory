// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"

	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
)

// IsValidDomainScope can be used to check the scope fields are set for usage
// on the domain level.
func IsValidDomainScope(scope openstackclients.ClientScope) error {
	if scope.Region == "" {
		return errors.New("missing region")
	}

	if scope.Domain == "" {
		return errors.New("missing domain")
	}

	if scope.NamedCredentials == "" {
		return errors.New("missing named credentials")
	}

	return nil
}

// IsValidProjectScope can be used to check the scope fields are set for usage
// on the project level.
func IsValidProjectScope(scope openstackclients.ClientScope) error {
	err := IsValidDomainScope(scope)
	if err != nil {
		return err
	}

	if scope.Project == "" {
		return errors.New("missing project name")
	}

	return nil
}
