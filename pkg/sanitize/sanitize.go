package sanitize

import "github.com/microcosm-cc/bluemonday"

var (
	strictPolicy = bluemonday.UGCPolicy()
	htmlPolicy   = newMessageHTMLPolicy()
)

func newMessageHTMLPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"b", "strong", "i", "em", "u", "s", "strike",
		"br", "p", "blockquote", "code", "pre",
		"ul", "ol", "li",
	)
	return p
}

func Text(value string) string {
	return strictPolicy.Sanitize(value)
}

func TextPtr(value *string) *string {
	if value == nil {
		return nil
	}
	s := Text(*value)
	return &s
}

func HTML(value string) string {
	return htmlPolicy.Sanitize(value)
}

func HTMLPtr(value *string) *string {
	if value == nil {
		return nil
	}
	s := HTML(*value)
	return &s
}
