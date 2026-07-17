package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFriendsFile_RoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store := NewFriendsFile()

	// Missing file must read as an empty list.
	friends, err := store.Load()
	if err != nil {
		t.Fatalf("Load (missing file): %v", err)
	}
	if len(friends) != 0 {
		t.Fatalf("friends = %v, want empty", friends)
	}

	if err := store.Save([]string{"malima-m", "jdiniz"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	friends, err = store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(friends) != 2 || friends[0] != "malima-m" || friends[1] != "jdiniz" {
		t.Fatalf("friends = %v, want [malima-m jdiniz]", friends)
	}
}

func TestFriendsFile_PreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, AppName)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("client_id: my-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := NewFriendsFile().Save([]string{"malima-m"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "my-id") {
		t.Errorf("client_id lost after Save:\n%s", content)
	}
	if !strings.Contains(content, "malima-m") {
		t.Errorf("friends missing after Save:\n%s", content)
	}
}
