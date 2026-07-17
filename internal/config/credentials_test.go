package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCredentials_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	paths, err := SaveCredentials("uid-abc", "secret-xyz")
	if err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	info, err := os.Stat(paths.ConfigFile)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Errorf("perms = %o, want no group/other access", info.Mode().Perm())
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "uid-abc" || cfg.ClientSecret != "secret-xyz" {
		t.Errorf("creds = %q/%q", cfg.ClientID, cfg.ClientSecret)
	}
	if cfg.APIBaseURL == "" || cfg.RedirectURI == "" {
		t.Error("defaults de api_base_url/redirect_uri deveriam estar preenchidos")
	}
}

func TestSaveCredentials_PreservesFriends(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, AppName)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	yaml := "friends:\n  - malima-m\nclient_id: old\nclient_secret: old-secret\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := SaveCredentials("new-id", "new-secret"); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	friends, err := NewFriendsFile().Load()
	if err != nil {
		t.Fatalf("Load friends: %v", err)
	}
	if len(friends) != 1 || friends[0] != "malima-m" {
		t.Errorf("friends = %v, want [malima-m]", friends)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ClientID != "new-id" || cfg.ClientSecret != "new-secret" {
		t.Errorf("creds = %q/%q", cfg.ClientID, cfg.ClientSecret)
	}
}

func TestSaveCredentials_RejectsEmpty(t *testing.T) {
	t.Parallel()

	if _, err := SaveCredentials("", "x"); err == nil {
		t.Fatal("deveria rejeitar client_id vazio")
	}
	if _, err := SaveCredentials("x", "  "); err == nil {
		t.Fatal("deveria rejeitar client_secret vazio")
	}
}

func TestHasCredentials(t *testing.T) {
	t.Parallel()

	if HasCredentials(Config{}) {
		t.Error("vazio não tem credenciais")
	}
	if !HasCredentials(Config{ClientID: "a", ClientSecret: "b"}) {
		t.Error("deveria ter credenciais")
	}
}
