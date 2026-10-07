package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestColorscheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if name, err := Colorscheme(); err != nil || name != "" {
		t.Fatalf("sem config: %q %v", name, err)
	}
	file := filepath.Join(dir, AppName, "config.yaml")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("client_id: abc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveColorscheme("dracula"); err != nil {
		t.Fatal(err)
	}
	if name, err := Colorscheme(); err != nil || name != "dracula" {
		t.Fatalf("salvo: %q %v", name, err)
	}
	data, _ := os.ReadFile(file)
	info, _ := os.Stat(file)
	if !strings.Contains(string(data), "client_id: abc") || info.Mode().Perm() != 0o600 {
		t.Fatalf("as outras chaves e o 0600 ficam: %s %v", data, info.Mode().Perm())
	}
}
