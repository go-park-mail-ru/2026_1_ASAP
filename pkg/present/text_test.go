package present

import "testing"

func TestTextForViewer(t *testing.T) {
	tests := []struct {
		name               string
		raw                string
		want               string
		subscriptionActive bool
	}{
		{
			name: "escapes html for free viewer",
			raw:  `<b>Hello</b>`,
			want: `&lt;b&gt;Hello&lt;/b&gt;`,
		},
		{
			name:               "escapes html and masks profanity for active subscription",
			raw:                `<b>блять</b>`,
			want:               `&lt;b&gt;***&lt;/b&gt;`,
			subscriptionActive: true,
		},
		{
			name:               "keeps clean text for active subscription",
			raw:                `Привет`,
			want:               `Привет`,
			subscriptionActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TextForViewer(tt.raw, tt.subscriptionActive); got != tt.want {
				t.Fatalf("TextForViewer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextPtrForViewer(t *testing.T) {
	tests := []struct {
		raw                *string
		want               *string
		name               string
		subscriptionActive bool
	}{
		{
			name: "nil",
			raw:  nil,
			want: nil,
		},
		{
			name: "escapes pointer value",
			raw:  ptrString(`<script>`),
			want: ptrString(`&lt;script&gt;`),
		},
		{
			name:               "masks pointer value",
			raw:                ptrString(`blyat!`),
			want:               ptrString(`***!`),
			subscriptionActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextPtrForViewer(tt.raw, tt.subscriptionActive)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("TextPtrForViewer() = %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != *tt.want {
				t.Fatalf("TextPtrForViewer() = %v, want %q", got, *tt.want)
			}
		})
	}
}

func ptrString(s string) *string {
	return &s
}
