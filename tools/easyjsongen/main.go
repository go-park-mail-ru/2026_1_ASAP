package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var jsonPackages = []string{
	"internal/auth/dto/auth",
	"internal/auth/dto/session",
	"internal/auth/dto/vkid",
	"internal/chat/dto/chat",
	"internal/chat/dto/message",
	"internal/chat/dto/sticker",
	"internal/chat/dto/ws",
	"internal/gateway/chat",
	"internal/gateway/complaint",
	"internal/gateway/payment",
	"internal/complaint/dto/analytic",
	"internal/complaint/dto/complaint",
	"internal/media/dto",
	"internal/payment/dto",
	"internal/profile/dto/contact",
	"internal/profile/dto/media",
	"internal/profile/dto/profile",
	"internal/search/dto/search",
	"internal/subscription/dto",
}

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fatal(err)
	}

	for _, pkg := range jsonPackages {
		files, err := jsonFiles(filepath.Join(repoRoot, pkg))
		if err != nil {
			fatal(err)
		}
		if len(files) == 0 {
			continue
		}
		if err = runEasyJSON(repoRoot, filepath.Join(repoRoot, pkg), files); err != nil {
			fatal(fmt.Errorf("%s: %w", pkg, err))
		}
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err = os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func jsonFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() ||
			!strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") ||
			strings.HasSuffix(name, "_easyjson.go") ||
			strings.HasSuffix(name, "_mock.go") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if bytes.Contains(data, []byte("`json:\"")) && supportedByEasyJSON(data) {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files, nil
}

func supportedByEasyJSON(data []byte) bool {
	unsupported := [][]byte{
		[]byte("io.Reader"),
		[]byte("multipart.File"),
		[]byte("FileInput"),
		[]byte("type WsResponse["),
	}
	for _, marker := range unsupported {
		if bytes.Contains(data, marker) {
			return false
		}
	}
	return true
}

func runEasyJSON(repoRoot, dir string, files []string) error {
	args := append([]string{"-all"}, files...)
	cmdName := easyjsonCommand()
	cmd := exec.Command(cmdName, args...)
	if cmdName == "" {
		args = append([]string{"run", "github.com/mailru/easyjson/easyjson", "-all"}, files...)
		cmd = exec.Command("go", args...)
	}

	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %w\n%s", cmd.Args, err, string(out))
	}

	relDir, err := filepath.Rel(repoRoot, dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		fmt.Printf("easyjson %s\n", filepath.ToSlash(filepath.Join(relDir, file)))
	}
	return nil
}

func easyjsonCommand() string {
	if path, err := exec.LookPath("easyjson"); err == nil {
		return path
	}

	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return ""
	}
	path := filepath.Join(strings.TrimSpace(string(out)), "bin", "easyjson")
	if _, err = os.Stat(path); err == nil {
		return path
	}
	return ""
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
