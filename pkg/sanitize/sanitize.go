package sanitize

import "html"

func Text(value string) string {
	return html.EscapeString(value)
}

func TextPtr(value *string) *string {
	if value == nil {
		return nil
	}
	s := Text(*value)
	return &s
}

func HTML(value string) string {
	return html.EscapeString(value)
}

func HTMLPtr(value *string) *string {
	if value == nil {
		return nil
	}
	s := HTML(*value)
	return &s
}
