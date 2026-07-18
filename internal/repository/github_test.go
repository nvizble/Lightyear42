package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubReleases_Latest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/nvizble/Lightyear42/releases/latest" {
			t.Errorf("path = %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Accept"); got == "" {
			t.Error("Accept header ausente")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v1.0.2",
			"assets": [
				{"name": "lightyear_1.0.2_Linux_x86_64.tar.gz", "browser_download_url": "https://example.com/a.tar.gz", "size": 100},
				{"name": "lightyear_1.0.2_linux_amd64.deb", "browser_download_url": "https://example.com/a.deb", "size": 200}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	repo := NewGitHubReleases("nvizble", "Lightyear42", srv.Client()).WithBaseURL(srv.URL)
	rel, err := repo.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if rel.TagName != "v1.0.2" {
		t.Fatalf("TagName = %q", rel.TagName)
	}
	if len(rel.Assets) != 2 {
		t.Fatalf("assets = %d", len(rel.Assets))
	}
	if rel.Assets[0].Name != "lightyear_1.0.2_Linux_x86_64.tar.gz" {
		t.Fatalf("asset name = %q", rel.Assets[0].Name)
	}
}

func TestGitHubReleases_Latest_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	repo := NewGitHubReleases("nvizble", "Lightyear42", srv.Client()).WithBaseURL(srv.URL)
	if _, err := repo.Latest(context.Background()); err == nil {
		t.Fatal("esperava erro HTTP")
	}
}
