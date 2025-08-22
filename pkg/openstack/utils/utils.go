// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"

	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/openstack/models"
)

// ErrNoProjectMatchingScope is an error which is returned when the task for finding
// [models.Project] project by client scope doesn't find a match.
var ErrNoProjectMatchingScope = errors.New("no project matching scope found")

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

	if scope.ProjectID == "" {
		return errors.New("missing project ID")
	}

	return nil
}

// MatchScopeToProject matches the given scope to an OpenStack project.
func MatchScopeToProject(scope openstackclients.ClientScope, projects []models.Project) (models.Project, error) {
	for _, project := range projects {
		if scope.Project != project.Name {
			continue
		}

		if scope.Domain != project.Domain {
			continue
		}

		if scope.Region != project.Region {
			continue
		}

		return project, nil
	}

	return models.Project{}, ErrNoProjectMatchingScope
}
