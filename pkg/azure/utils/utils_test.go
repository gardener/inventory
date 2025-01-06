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
)

func ptr[T any](t T) *T {
	return &t
}

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
					Code: ptr("PowerState/on"),
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
			output := utils.GetPowerState(tc.input)
			if strings.Compare(output, tc.wanted) != 0 {
				t.Fatalf("got %s wanted %s", output, tc.wanted)
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
		desc          string
		input         error
		isWantedError func(error) bool
	}{
		{
			desc:  "non-Azure error is not skipped",
			input: nonAzureError,
			isWantedError: func(err error) bool {
				return !errors.Is(err, asynq.SkipRetry)
			},
		},
		{
			desc:  "Azure error 'Not found' is skipped",
			input: &azErrorStatusNotFound,
			isWantedError: func(err error) bool {
				return errors.Is(err, asynq.SkipRetry)
			},
		},
		{
			desc:  "Azure error 'Forbidden' is not skipped",
			input: &azErrorStatusForbidden,
			isWantedError: func(err error) bool {
				return !errors.Is(err, asynq.SkipRetry)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			outError := utils.MaybeSkipRetry(tc.input)
			if !tc.isWantedError(outError) {
				// should error message be more explicit?
				t.Fatalf("error incorrectly wrapped")
			}
		})
	}
}
