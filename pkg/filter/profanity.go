package filter

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

//go:embed profanity_roots.txt
var defaultProfanityRoots []byte

var (
	rootsMu sync.RWMutex
	roots   []string
)

func init() {
	if err := SetRootsFromReader(bytes.NewReader(defaultProfanityRoots)); err != nil {
		panic(fmt.Sprintf("filter: load embedded profanity roots: %v", err))
	}
}

// Init loads roots from path when set; otherwise keeps the embedded list.
func Init(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return LoadRootsFromFile(path)
}

// LoadRootsFromFile replaces the active root list from a text file (one root per line).
func LoadRootsFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open profanity roots %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return SetRootsFromReader(f)
}

// SetRootsFromReader parses roots from r and installs them for MaskProfanity.
func SetRootsFromReader(r interface {
	Read([]byte) (int, error)
}) error {
	parsed, err := parseRoots(r)
	if err != nil {
		return err
	}
	if len(parsed) == 0 {
		return fmt.Errorf("profanity roots list is empty")
	}
	rootsMu.Lock()
	roots = parsed
	rootsMu.Unlock()
	return nil
}

func parseRoots(r interface {
	Read([]byte) (int, error)
}) ([]string, error) {
	scanner := bufio.NewScanner(r)
	out := make([]string, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.ToLower(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read profanity roots: %w", err)
	}
	return out, nil
}

func currentRoots() []string {
	rootsMu.RLock()
	defer rootsMu.RUnlock()
	return roots
}

// MaskProfanity replaces profane words with ***.
func MaskProfanity(text string) string {
	if text == "" {
		return text
	}
	active := currentRoots()
	var b strings.Builder
	b.Grow(len(text))
	i := 0
	for i < len(text) {
		r, size := utf8.DecodeRuneInString(text[i:])
		if !isWordRune(r) {
			b.WriteRune(r)
			i += size
			continue
		}
		j := i
		for j < len(text) {
			r2, sz := utf8.DecodeRuneInString(text[j:])
			if !isWordRune(r2) {
				break
			}
			j += sz
		}
		word := text[i:j]
		if matched, masked := maskWord(word, active); matched {
			b.WriteString(masked)
		} else {
			b.WriteString(word)
		}
		i = j
	}
	return b.String()
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\''
}

func maskWord(word string, active []string) (bool, string) {
	norm := normalizeForMatch(word)
	if norm == "" {
		return false, word
	}
	for _, root := range active {
		if strings.Contains(norm, root) {
			return true, strings.Repeat("*", 3)
		}
	}
	return false, word
}

func normalizeForMatch(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'ё':
			b.WriteRune('е')
		case '0':
			b.WriteRune('о')
		case '1', '|':
			continue
		case '@':
			b.WriteRune('а')
		case '$':
			b.WriteRune('с')
		case '3':
			b.WriteRune('з')
		case '4':
			b.WriteRune('ч')
		case '6':
			b.WriteRune('б')
		case '9':
			b.WriteRune('д')
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
