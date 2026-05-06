package sanitize

import "html"

func Text(value string) string {
	return html.EscapeString(value)
}

func TextPtr(value *string) *string {
	if value == nil {
		return nil
	}

	escaped := html.EscapeString(*value)
	return &escaped
}
