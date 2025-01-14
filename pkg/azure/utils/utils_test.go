// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/azure/constants"
	"github.com/gardener/inventory/pkg/azure/utils"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

func TestGetPowerState(t *testing.T) {
	testCases := []struct {
		desc   string
		input  []*armcompute.InstanceViewStatus
		wanted string
	}{
		{
			desc: "power state prefix exists",
			input: []*armcompute.InstanceViewStatus{
				{
					Code: ptr.To("PowerState/on"),
				},
			},
			wanted: "on",
		},
		{
			desc:   "nil states",
			input:  nil,
			wanted: constants.PowerStateUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := utils.GetPowerState(tc.input)
			if strings.Compare(got, tc.wanted) != 0 {
				t.Fatalf("got %s wanted %s", got, tc.wanted)
			}
		})
	}
}

func TestMaybeSkipRetry(t *testing.T) {
	nonAzureError := errors.New("test error")
	azErrorStatusNotFound := azcore.ResponseError{
		StatusCode: http.StatusNotFound,
	}
	azErrorStatusForbidden := azcore.ResponseError{
		StatusCode: http.StatusForbidden,
	}

	testCases := []struct {
		desc       string
		err        error
		shouldSkip bool
	}{
		{
			desc:       "non-Azure error is not skipped",
			err:        nonAzureError,
			shouldSkip: false,
		},
		{
			desc:       "Azure error 'Not found' is skipped",
			err:        &azErrorStatusNotFound,
			shouldSkip: true,
		},
		{
			desc:       "Azure error 'Forbidden' is not skipped",
			err:        &azErrorStatusForbidden,
			shouldSkip: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			gotError := utils.MaybeSkipRetry(tc.err)
			if errors.Is(gotError, asynq.SkipRetry) != tc.shouldSkip {
				t.Fatalf("got %v wanted %v", gotError, tc.shouldSkip)
			}
		})
	}
}
