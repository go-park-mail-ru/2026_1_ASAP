package filter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaskProfanity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"clean", "Привет мир", "Привет мир"},
		{"single", "блять", "***"},
		{"in sentence", "ну блять же", "ну *** же"},
		{"latin", "blyat", "***"},
		{"preserve punctuation", "блять!", "***!"},
		{"html escaped kept", "test", "test"},
		{"false positive stem", "стебель", "стебель"},
		{"false positive sky", "небо", "небо"},
		{"profane verb", "ебать", "***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := MaskProfanity(tt.in); got != tt.want {
				t.Fatalf("MaskProfanity(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLoadRootsFromFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "roots.txt")
	content := "# comment\n\nкастом\n\n# tail\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadRootsFromFile(path); err != nil {
		t.Fatal(err)
	}
	if got := MaskProfanity("кастомное"); got != "***" {
		t.Fatalf("custom root: got %q", got)
	}
	if got := MaskProfanity("блять"); got != "блять" {
		t.Fatalf("old root should not match after reload: got %q", got)
	}

	// restore embedded defaults for other tests in package
	if err := SetRootsFromReader(strings.NewReader(string(defaultProfanityRoots))); err != nil {
		t.Fatal(err)
	}
}

func TestParseRoots_SkipsCommentsAndEmpty(t *testing.T) {
	t.Parallel()
	got, err := parseRoots(strings.NewReader("  # x\n\n  AbC  \n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "abc" {
		t.Fatalf("parseRoots = %v", got)
	}
}
