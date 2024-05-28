package utils

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"strings"
)

// FetchTag returns the value of the AWS tag with the key s or an empty string if the tag is not found.
func FetchTag(tags []types.Tag, key string) string {
	for _, t := range tags {
		if strings.Compare(*t.Key, key) == 0 {
			return *t.Value
		}
	}
	return ""
}
