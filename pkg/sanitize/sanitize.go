package sanitize

import "github.com/microcosm-cc/bluemonday"

var policy = bluemonday.UGCPolicy()

func Text(value string) string {
	return policy.Sanitize(value)
}

func TextPtr(value *string) *string {
	if value == nil {
		return nil
	}

	sanitized := policy.Sanitize(*value)
	return &sanitized
}
