package sanitize

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func TestText(t *testing.T) {
	type args struct {
		value string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Empty string",
			args: args{value: ""},
			want: "",
		},
		{
			name: "Plain text without special characters",
			args: args{value: "Hello World"},
			want: "Hello World",
		},
		{
			name: "Text with ampersand",
			args: args{value: "A & B"},
			want: "A &amp; B",
		},
		{
			name: "Text with less than sign",
			args: args{value: "x < y"},
			want: "x &lt; y",
		},
		{
			name: "Text with greater than sign",
			args: args{value: "x > y"},
			want: "x &gt; y",
		},
		{
			name: "Text with double quote",
			args: args{value: `He said "Hello"`},
			want: "He said &#34;Hello&#34;",
		},
		{
			name: "Text with single quote",
			args: args{value: "It's a test"},
			want: "It&#39;s a test",
		},
		{
			name: "Text with HTML script tag",
			args: args{value: "<script>alert('xss')</script>"},
			want: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name: "Text with HTML bold tag",
			args: args{value: "<b>Bold text</b>"},
			want: "&lt;b&gt;Bold text&lt;/b&gt;",
		},
		{
			name: "Text with multiple special characters",
			args: args{value: "A & B < C > D \"E\" F 'G'"},
			want: "A &amp; B &lt; C &gt; D &#34;E&#34; F &#39;G&#39;",
		},
		{
			name: "Text with only special characters",
			args: args{value: "&<>\"'"},
			want: "&amp;&lt;&gt;&#34;&#39;",
		},
		{
			name: "Text with Russian characters",
			args: args{value: "Привет мир"},
			want: "Привет мир",
		},
		{
			name: "Text with numbers",
			args: args{value: "123 456"},
			want: "123 456",
		},
		{
			name: "Text with newlines",
			args: args{value: "Line1\nLine2"},
			want: "Line1\nLine2",
		},
		{
			name: "Text with tabs",
			args: args{value: "Col1\tCol2"},
			want: "Col1\tCol2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Text(tt.args.value)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTextPtr(t *testing.T) {
	type args struct {
		value *string
	}

	tests := []struct {
		name string
		args args
		want *string
	}{
		{
			name: "Nil pointer",
			args: args{value: nil},
			want: nil,
		},
		{
			name: "Empty string pointer",
			args: args{value: strPtr("")},
			want: strPtr(""),
		},
		{
			name: "Plain text without special characters",
			args: args{value: strPtr("Hello World")},
			want: strPtr("Hello World"),
		},
		{
			name: "Text with ampersand",
			args: args{value: strPtr("A & B")},
			want: strPtr("A &amp; B"),
		},
		{
			name: "Text with less than sign",
			args: args{value: strPtr("x < y")},
			want: strPtr("x &lt; y"),
		},
		{
			name: "Text with greater than sign",
			args: args{value: strPtr("x > y")},
			want: strPtr("x &gt; y"),
		},
		{
			name: "Text with double quote",
			args: args{value: strPtr(`He said "Hello"`)},
			want: strPtr("He said &#34;Hello&#34;"),
		},
		{
			name: "Text with single quote",
			args: args{value: strPtr("It's a test")},
			want: strPtr("It&#39;s a test"),
		},
		{
			name: "Text with HTML script tag",
			args: args{value: strPtr("<script>alert('xss')</script>")},
			want: strPtr("&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"),
		},
		{
			name: "Text with HTML bold tag",
			args: args{value: strPtr("<b>Bold text</b>")},
			want: strPtr("&lt;b&gt;Bold text&lt;/b&gt;"),
		},
		{
			name: "Text with multiple special characters",
			args: args{value: strPtr("A & B < C > D \"E\" F 'G'")},
			want: strPtr("A &amp; B &lt; C &gt; D &#34;E&#34; F &#39;G&#39;"),
		},
		{
			name: "Text with only special characters",
			args: args{value: strPtr("&<>\"'")},
			want: strPtr("&amp;&lt;&gt;&#34;&#39;"),
		},
		{
			name: "Text with Russian characters",
			args: args{value: strPtr("Привет мир")},
			want: strPtr("Привет мир"),
		},
		{
			name: "Text with numbers",
			args: args{value: strPtr("123 456")},
			want: strPtr("123 456"),
		},
		{
			name: "Text with newlines",
			args: args{value: strPtr("Line1\nLine2")},
			want: strPtr("Line1\nLine2"),
		},
		{
			name: "Text with tabs",
			args: args{value: strPtr("Col1\tCol2")},
			want: strPtr("Col1\tCol2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextPtr(tt.args.value)
			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, *tt.want, *got)
			}
		})
	}
}
