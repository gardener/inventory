// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gardener/inventory/pkg/gcp/constants"
)

// ProjectFQN returns the fully-qualified name for the given project id.
func ProjectFQN(s string) string {
	if strings.HasPrefix(s, constants.ProjectsPrefix) {
		return s
	}

	return fmt.Sprintf("%s%s", constants.ProjectsPrefix, s)
}

// ZoneFQN returns the fully-qualified name for the given zone name.
func ZoneFQN(s string) string {
	if strings.HasPrefix(s, constants.ZonesPrefix) {
		return s
	}

	return fmt.Sprintf("%s%s", constants.ZonesPrefix, s)
}

// UnqualifyRegion returns the unqualified name for a region.
func UnqualifyRegion(s string) string {
	return strings.TrimPrefix(s, constants.RegionsPrefix)
}

// UnqualifyZone returns the unqualified name for a zone.
func UnqualifyZone(s string) string {
	return strings.TrimPrefix(s, constants.ZonesPrefix)
}

// RegionFromZone returns the region name from a given zone according to the
// [GCP Naming Convention]. If the provided zone name is invalid or empty, the
// function returns an empty string.
//
// [GCP Naming Convention]: https://cloud.google.com/compute/docs/regions-zones#identifying_a_region_or_zone
func RegionFromZone(zone string) string {
	zone = UnqualifyZone(zone)
	if zone == "" {
		return ""
	}
	idx := strings.LastIndex(zone, "-")
	if idx == -1 {
		return ""
	}

	return zone[:idx]
}

// ResourceNameFromURL returns the name of a resource from the specified URL.
//
// See [Resource Names] for more details.
//
// [Resource Names]: https://cloud.google.com/apis/design/resource_names
func ResourceNameFromURL(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}

	parts := strings.Split(u.Path, "/")
	return parts[len(parts)-1]
}
