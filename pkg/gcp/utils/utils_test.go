// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"strings"
	"testing"

	"github.com/gardener/inventory/pkg/gcp/constants"
	"github.com/gardener/inventory/pkg/gcp/utils"
)

func TestProjectFQN(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		wanted string
	}{
		{
			desc:   "input includes projects/ prefix",
			input:  constants.ProjectsPrefix + "testproject",
			wanted: constants.ProjectsPrefix + "testproject",
		},
		{
			desc:   "input does not include projects/ prefix",
			input:  "testproject",
			wanted: constants.ProjectsPrefix + "testproject",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := utils.ProjectFQN(tc.input)
			if strings.Compare(tc.wanted, output) != 0 {
				t.Fatalf("wanted %s got %s", tc.wanted, output)
			}
		})
	}
}

// identical to the one above, but for ZoneFQN
func TestZoneFQN(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		wanted string
	}{
		{
			desc:   "input includes zones/ prefix",
			input:  constants.ZonesPrefix + "testzone",
			wanted: constants.ZonesPrefix + "testzone",
		},
		{
			desc:   "input does not include zones/ prefix",
			input:  "testzone",
			wanted: constants.ZonesPrefix + "testzone",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := utils.ZoneFQN(tc.input)
			if strings.Compare(tc.wanted, output) != 0 {
				t.Fatalf("wanted %s got %s", tc.wanted, output)
			}
		})
	}
}

func TestUnqualifyRegion(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		wanted string
	}{
		{
			desc:   "input includes region prefix",
			input:  constants.RegionsPrefix + "testregion",
			wanted: "testregion",
		},
		{
			desc:   "input does not include region prefix",
			input:  "testregion",
			wanted: "testregion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := utils.UnqualifyRegion(tc.input)
			if strings.Compare(tc.wanted, output) != 0 {
				t.Fatalf("wanted %s got %s", tc.wanted, output)
			}
		})
	}
}
func TestUnqualifyZone(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		wanted string
	}{
		{
			desc:   "input includes zone prefix",
			input:  constants.ZonesPrefix + "testzone",
			wanted: "testzone",
		},
		{
			desc:   "input does not include zone prefix",
			input:  "testzone",
			wanted: "testzone",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := utils.UnqualifyZone(tc.input)
			if strings.Compare(tc.wanted, output) != 0 {
				t.Fatalf("wanted %s got %s", tc.wanted, output)
			}
		})
	}
}

func TestRegionFromZone(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		wanted string
	}{
		{
			desc:   "input includes region name",
			input:  "testregion1-testzone",
			wanted: "testregion1",
		},
		{
			desc:   "input includes zones prefix",
			input:  constants.ZonesPrefix + "testregion1-testzone",
			wanted: "testregion1",
		},
		{
			desc:   "input only includes zone prefix",
			input:  constants.ZonesPrefix,
			wanted: "",
		},
		{
			desc:   "input does not include region name",
			input:  "testzone",
			wanted: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := utils.RegionFromZone(tc.input)
			if strings.Compare(tc.wanted, output) != 0 {
				t.Fatalf("wanted %s got %s", tc.wanted, output)
			}
		})
	}
}

func TestResourceNameFromURL(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		wanted string
	}{
		{
			desc:   "valid URL",
			input:  "testinstance",
			wanted: "testinstance",
		},
		{
			desc:   "valid URL",
			input:  "instances/testinstance",
			wanted: "testinstance",
		},
		{
			desc:   "with host",
			input:  "https://testhost.com/instances/testinstance",
			wanted: "testinstance",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := utils.ResourceNameFromURL(tc.input)
			if strings.Compare(tc.wanted, output) != 0 {
				t.Fatalf("wanted %s got %s", tc.wanted, output)
			}
		})
	}
}
