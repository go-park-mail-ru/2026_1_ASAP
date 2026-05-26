package sanitize

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func TestText(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: ""},
		{name: "plain", value: "Hello World", want: "Hello World"},
		{name: "ampersand", value: "A & B", want: "A &amp; B"},
		{name: "less than", value: "x < y", want: "x &lt; y"},
		{name: "script tag", value: "<script>alert('xss')</script>", want: ""},
		{name: "bold tag kept", value: "<b>Bold text</b>", want: "<b>Bold text</b>"},
		{name: "img tag sanitized", value: `<img src=x onerror=alert(1)>`, want: `<img src="x">`},
		{name: "russian", value: "Привет мир", want: "Привет мир"},
		{name: "newlines", value: "Line1\nLine2", want: "Line1\nLine2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Text(tt.value))
		})
	}
}

func TestHTML(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: ""},
		{name: "plain", value: "Hello World", want: "Hello World"},
		{name: "ampersand", value: "a & b", want: "a &amp; b"},
		{name: "bold kept", value: "<b>Bold text</b>", want: "<b>Bold text</b>"},
		{name: "script stripped", value: "<script>alert('xss')</script>", want: ""},
		{name: "img stripped", value: `<img src=x onerror=alert(1)>`, want: ""},
		{name: "mixed", value: "edited <b>text</b>", want: "edited <b>text</b>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, HTML(tt.value))
		})
	}
}

func TestTextPtr(t *testing.T) {
	require.Nil(t, TextPtr(nil))
	got := TextPtr(strPtr("A & B"))
	require.NotNil(t, got)
	require.Equal(t, "A &amp; B", *got)
}

func TestHTMLPtr(t *testing.T) {
	require.Nil(t, HTMLPtr(nil))
	got := HTMLPtr(strPtr("edited <b>x</b>"))
	require.NotNil(t, got)
	require.Equal(t, "edited <b>x</b>", *got)
}

func TestTextAndHTML_differOnImg(t *testing.T) {
	raw := `<img src=x onerror=alert(1)>`
	require.Equal(t, `<img src="x">`, Text(raw))
	require.Equal(t, "", HTML(raw))
}
