package present

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ptrString(s string) *string {
	return &s
}

func TestTextForViewer(t *testing.T) {
	tests := []struct {
		name               string
		raw                string
		want               string
		subscriptionActive bool
	}{
		{
			name:               "plain text for free viewer",
			raw:                "Hello World",
			want:               "Hello World",
			subscriptionActive: false,
		},
		{
			name:               "plain text for active subscription",
			raw:                "Hello World",
			want:               "Hello World",
			subscriptionActive: true,
		},
		{
			name:               "html escaped for free viewer",
			raw:                "<b>Hello</b>",
			want:               "&lt;b&gt;Hello&lt;/b&gt;",
			subscriptionActive: false,
		},
		{
			name:               "html escaped for active subscription",
			raw:                "<b>Hello</b>",
			want:               "&lt;b&gt;Hello&lt;/b&gt;",
			subscriptionActive: true,
		},
		{
			name:               "profanity masked for free viewer",
			raw:                "блять",
			want:               "блять",
			subscriptionActive: false,
		},
		{
			name:               "profanity masked for active subscription",
			raw:                "блять",
			want:               "***",
			subscriptionActive: true,
		},
		{
			name:               "html with profanity for active subscription",
			raw:                "<b>блять</b>",
			want:               "&lt;b&gt;***&lt;/b&gt;",
			subscriptionActive: true,
		},
		{
			name:               "empty string",
			raw:                "",
			want:               "",
			subscriptionActive: false,
		},
		{
			name:               "script tag escaped",
			raw:                "<script>alert('xss')</script>",
			want:               "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
			subscriptionActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextForViewer(tt.raw, tt.subscriptionActive)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTextPtrForViewer(t *testing.T) {
	tests := []struct {
		name               string
		raw                *string
		want               *string
		subscriptionActive bool
	}{
		{
			name:               "nil pointer",
			raw:                nil,
			want:               nil,
			subscriptionActive: false,
		},
		{
			name:               "plain text pointer for free viewer",
			raw:                ptrString("Hello"),
			want:               ptrString("Hello"),
			subscriptionActive: false,
		},
		{
			name:               "plain text pointer for active subscription",
			raw:                ptrString("Hello"),
			want:               ptrString("Hello"),
			subscriptionActive: true,
		},
		{
			name:               "html escaped from pointer",
			raw:                ptrString("<b>Hello</b>"),
			want:               ptrString("&lt;b&gt;Hello&lt;/b&gt;"),
			subscriptionActive: false,
		},
		{
			name:               "profanity masked from pointer for active subscription",
			raw:                ptrString("блять"),
			want:               ptrString("***"),
			subscriptionActive: true,
		},
		{
			name:               "empty string pointer",
			raw:                ptrString(""),
			want:               ptrString(""),
			subscriptionActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextPtrForViewer(tt.raw, tt.subscriptionActive)
			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, *tt.want, *got)
			}
		})
	}
}

func TestPlainTextForViewer(t *testing.T) {
	tests := []struct {
		name               string
		raw                string
		want               string
		subscriptionActive bool
	}{
		{
			name:               "plain text for free viewer",
			raw:                "Hello World",
			want:               "Hello World",
			subscriptionActive: false,
		},
		{
			name:               "plain text for active subscription",
			raw:                "Hello World",
			want:               "Hello World",
			subscriptionActive: true,
		},
		{
			name:               "html escaped for free viewer",
			raw:                "<b>Hello</b>",
			want:               "&lt;b&gt;Hello&lt;/b&gt;",
			subscriptionActive: false,
		},
		{
			name:               "html escaped and profanity masked for active subscription",
			raw:                "<b>блять</b>",
			want:               "&lt;b&gt;***&lt;/b&gt;",
			subscriptionActive: true,
		},
		{
			name:               "profanity masked for active subscription",
			raw:                "блять",
			want:               "***",
			subscriptionActive: true,
		},
		{
			name:               "profanity not masked for free viewer",
			raw:                "блять",
			want:               "блять",
			subscriptionActive: false,
		},
		{
			name:               "empty string",
			raw:                "",
			want:               "",
			subscriptionActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PlainTextForViewer(tt.raw, tt.subscriptionActive)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPlainTextPtrForViewer(t *testing.T) {
	tests := []struct {
		name               string
		raw                *string
		want               *string
		subscriptionActive bool
	}{
		{
			name:               "nil pointer",
			raw:                nil,
			want:               nil,
			subscriptionActive: false,
		},
		{
			name:               "plain text pointer for free viewer",
			raw:                ptrString("Hello"),
			want:               ptrString("Hello"),
			subscriptionActive: false,
		},
		{
			name:               "html escaped from pointer",
			raw:                ptrString("<b>Hello</b>"),
			want:               ptrString("&lt;b&gt;Hello&lt;/b&gt;"),
			subscriptionActive: false,
		},
		{
			name:               "html escaped and profanity masked for active subscription",
			raw:                ptrString("<b>блять</b>"),
			want:               ptrString("&lt;b&gt;***&lt;/b&gt;"),
			subscriptionActive: true,
		},
		{
			name:               "empty string pointer",
			raw:                ptrString(""),
			want:               ptrString(""),
			subscriptionActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PlainTextPtrForViewer(tt.raw, tt.subscriptionActive)
			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, *tt.want, *got)
			}
		})
	}
}
