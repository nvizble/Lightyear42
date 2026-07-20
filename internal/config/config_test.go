package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if cfg.APIBaseURL != "https://api.intra.42.fr/v2" {
		t.Fatalf("APIBaseURL = %q, want api.intra.42.fr/v2", cfg.APIBaseURL)
	}
	if cfg.RedirectURI == "" {
		t.Fatal("RedirectURI must not be empty")
	}
}

func TestResolvePaths_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}

	if filepath.Base(paths.ConfigDir) != AppName {
		t.Fatalf("ConfigDir base = %q, want %q", filepath.Base(paths.ConfigDir), AppName)
	}
	if filepath.Base(paths.ConfigFile) != "config.yaml" {
		t.Fatalf("ConfigFile base = %q, want config.yaml", filepath.Base(paths.ConfigFile))
	}
	if filepath.Base(paths.CacheDir) != AppName {
		t.Fatalf("CacheDir base = %q, want %q", filepath.Base(paths.CacheDir), AppName)
	}
	if filepath.Base(paths.DataDir) != AppName {
		t.Fatalf("DataDir base = %q, want %q", filepath.Base(paths.DataDir), AppName)
	}
}

func TestLoad_MissingFileUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("FORTYTWO_CLIENT_ID", "")
	t.Setenv("FORTYTWO_CLIENT_SECRET", "")
	t.Setenv("FORTYTWO_API_BASE_URL", "")
	t.Setenv("FORTYTWO_REDIRECT_URI", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := Default()
	if cfg.APIBaseURL != want.APIBaseURL {
		t.Fatalf("APIBaseURL = %q, want %q", cfg.APIBaseURL, want.APIBaseURL)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("FORTYTWO_CLIENT_ID", "test-client-id")
	t.Setenv("FORTYTWO_API_BASE_URL", "https://example.test/v2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.ClientID != "test-client-id" {
		t.Fatalf("ClientID = %q, want test-client-id", cfg.ClientID)
	}
	if cfg.APIBaseURL != "https://example.test/v2" {
		t.Fatalf("APIBaseURL = %q, want https://example.test/v2", cfg.APIBaseURL)
	}
}

func TestLoad_CampusLayoutFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, AppName)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	yaml := "campus_layout:\n  \"1\":\n    rows: 10\n    posts: 4\n  \"3\":\n    rows: 13\n    posts: 6\n    seats: 64\n    natural_posts: true\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.CampusLayout["1"]; got != (ClusterLayout{Rows: 10, Posts: 4}) {
		t.Fatalf("CampusLayout[1] = %+v, want {Rows:10 Posts:4}", got)
	}
	if got := cfg.CampusLayout["3"]; got != (ClusterLayout{Rows: 13, Posts: 6, Seats: 64, NaturalPosts: true}) {
		t.Fatalf("CampusLayout[3] = %+v, want {Rows:13 Posts:6 Seats:64 NaturalPosts:true}", got)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir: %v", err)
	}

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}

	info, err := os.Stat(paths.ConfigDir)
	if err != nil {
		t.Fatalf("Stat config dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("config path is not a directory")
	}
}
