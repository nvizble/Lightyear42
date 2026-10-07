package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
)

const (
	defaultGitHubAPI   = "https://api.github.com"
	defaultHTTPTimeout = 30 * time.Second
	maxReleaseBody     = 1 << 20 // 1 MiB
	maxReleasesBody    = 8 << 20 // 8 MiB: a page of releases with all assets
	releasesPerPage    = 30
)

// GitHubReleases reads public release metadata from GitHub.
type GitHubReleases interface {
	// Latest returns the newest stable release (GitHub skips pre-releases).
	Latest(ctx context.Context) (*models.Release, error)
	// List returns the most recent releases, pre-releases included.
	List(ctx context.Context) ([]models.Release, error)
}

// GitHubReleasesRepository implements GitHubReleases over the GitHub REST API.
type GitHubReleasesRepository struct {
	owner  string
	repo   string
	base   string
	client *http.Client
}

// NewGitHubReleases creates a client for owner/repo (e.g. nvizble/Lightyear42).
func NewGitHubReleases(owner, repo string, httpClient *http.Client) *GitHubReleasesRepository {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &GitHubReleasesRepository{
		owner:  owner,
		repo:   repo,
		base:   defaultGitHubAPI,
		client: httpClient,
	}
}

// WithBaseURL overrides the API root (tests).
func (r *GitHubReleasesRepository) WithBaseURL(base string) *GitHubReleasesRepository {
	r.base = strings.TrimSuffix(base, "/")
	return r
}

// Latest returns the newest published stable release for the repository.
func (r *GitHubReleasesRepository) Latest(ctx context.Context) (*models.Release, error) {
	var release models.Release
	if err := r.get(ctx, "/releases/latest", maxReleaseBody, &release); err != nil {
		return nil, err
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("release sem tag_name")
	}
	return &release, nil
}

// List returns the most recent releases (newest first), pre-releases included.
func (r *GitHubReleasesRepository) List(ctx context.Context) ([]models.Release, error) {
	var releases []models.Release
	path := fmt.Sprintf("/releases?per_page=%d", releasesPerPage)
	if err := r.get(ctx, path, maxReleasesBody, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

// get fetches /repos/{owner}/{repo}{path} and decodes the JSON body into v.
func (r *GitHubReleasesRepository) get(ctx context.Context, path string, limit int64, v any) error {
	url := fmt.Sprintf("%s/repos/%s/%s%s", r.base, r.owner, r.repo, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "lightyear-cli")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("consultar releases: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return fmt.Errorf("ler resposta do GitHub: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub releases: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decodificar release: %w", err)
	}
	return nil
}
