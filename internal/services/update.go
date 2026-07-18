package services

import (
	"context"
	"fmt"
	"runtime"

	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/repository"
	"github.com/nvizble/Lightyear42/internal/update"
)

// DefaultGitHubOwner and DefaultGitHubRepo identify the public release source.
const (
	DefaultGitHubOwner = "nvizble"
	DefaultGitHubRepo  = "Lightyear42"
)

// UpdateOptions controls Check/Apply behavior.
type UpdateOptions struct {
	// Current is the running binary version (e.g. cmd.Version).
	Current string
	// CheckOnly reports without downloading.
	CheckOnly bool
	// Force allows updating from non-semver builds (e.g. "dev").
	Force bool
	// GOOS/GOARCH override the runtime platform (tests).
	GOOS   string
	GOARCH string
}

// UpdatePlan is the result of checking GitHub for a newer release.
type UpdatePlan struct {
	Current string
	Latest  string
	Newer   bool
	Asset   models.ReleaseAsset
}

// BinaryInstaller installs a release archive over the current executable.
type BinaryInstaller interface {
	TargetPath() (string, error)
	Install(ctx context.Context, archiveURL, archiveName string) error
}

// UpdateService orchestrates self-update from GitHub Releases.
type UpdateService struct {
	releases repository.GitHubReleases
	install  BinaryInstaller
}

// NewUpdateService wires release lookup and binary installer.
func NewUpdateService(releases repository.GitHubReleases, install BinaryInstaller) *UpdateService {
	return &UpdateService{releases: releases, install: install}
}

// Check fetches the latest release and decides whether an update is available.
func (s *UpdateService) Check(ctx context.Context, opts UpdateOptions) (*UpdatePlan, error) {
	goos := opts.GOOS
	goarch := opts.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	current := opts.Current
	if !update.IsReleaseVersion(current) && !opts.Force {
		return nil, fmt.Errorf("versão local %q não é um release (use --force para atualizar builds de desenvolvimento)", current)
	}

	rel, err := s.releases.Latest(ctx)
	if err != nil {
		return nil, err
	}

	asset, err := update.SelectAsset(rel.Assets, goos, goarch)
	if err != nil {
		return nil, err
	}

	plan := &UpdatePlan{
		Current: current,
		Latest:  rel.TagName,
		Asset:   *asset,
	}

	if update.IsReleaseVersion(current) {
		newer, err := update.IsNewer(current, rel.TagName)
		if err != nil {
			return nil, err
		}
		plan.Newer = newer
	} else {
		// Forced update from dev: always treat remote as installable.
		plan.Newer = true
	}

	return plan, nil
}

// Apply downloads and installs the planned release asset.
func (s *UpdateService) Apply(ctx context.Context, plan *UpdatePlan) error {
	if plan == nil {
		return fmt.Errorf("plano de update ausente")
	}
	if s.install == nil {
		return fmt.Errorf("installer não configurado")
	}
	target, err := s.install.TargetPath()
	if err != nil {
		return err
	}
	if err := s.install.Install(ctx, plan.Asset.BrowserDownloadURL, plan.Asset.Name); err != nil {
		return fmt.Errorf("instalar em %s: %w", target, err)
	}
	return nil
}
