package utils_test

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gardener/inventory/pkg/aws/utils"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

func TestFetchTag(t *testing.T) {
	testCases := []struct {
		desc   string
		tags   []types.Tag
		key    string
		wanted string
	}{
		{
			desc: "fetch existing tag",
			tags: []types.Tag{
				{Key: ptr.To("tag1"), Value: ptr.To("value1")},
				{Key: ptr.To("tag2"), Value: ptr.To("value2")},
				{Key: ptr.To("tag3"), Value: ptr.To("value3")},
			},
			key:    "tag1",
			wanted: "value1",
		},
		{
			desc: "fetch missing tag",
			tags: []types.Tag{
				{Key: ptr.To("tag2"), Value: ptr.To("value2")},
				{Key: ptr.To("tag3"), Value: ptr.To("value3")},
			},
			key:    "tag1",
			wanted: "",
		},
		{
			desc: "handle tags with nil key",
			tags: []types.Tag{
				{Key: nil, Value: nil},
				{Key: ptr.To("tag1"), Value: ptr.To("value1")},
			},
			key:    "tag1",
			wanted: "value1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := utils.FetchTag(tc.tags, tc.key)
			if strings.Compare(output, tc.wanted) != 0 {
				t.Fatalf("want %s got %s", tc.wanted, output)
			}
		})
	}
}
