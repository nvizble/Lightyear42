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
)

// GitHubReleases reads public release metadata from GitHub.
type GitHubReleases interface {
	Latest(ctx context.Context) (*models.Release, error)
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

// Latest returns the newest published release for the repository.
func (r *GitHubReleasesRepository) Latest(ctx context.Context) (*models.Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", r.base, r.owner, r.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "lightyear-cli")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultar releases: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseBody))
	if err != nil {
		return nil, fmt.Errorf("ler resposta do GitHub: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub releases: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release models.Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("decodificar release: %w", err)
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("release sem tag_name")
	}
	return &release, nil
}
