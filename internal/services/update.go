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
	// Canary opts into the canary channel: the newest release, pre-releases
	// included. Running a canary build implies it unless Stable is set.
	Canary bool
	// Stable pins the stable channel, allowing a downgrade from a canary build.
	Stable bool
	// GOOS/GOARCH override the runtime platform (tests).
	GOOS   string
	GOARCH string
}

// UpdatePlan is the result of checking GitHub for a newer release.
type UpdatePlan struct {
	Current string
	Latest  string
	// Channel is "stable" or "canary".
	Channel string
	Newer   bool
	// Downgrade is set when leaving the canary channel for an older stable.
	Downgrade bool
	Asset     models.ReleaseAsset
}

// Update channels.
const (
	ChannelStable = "stable"
	ChannelCanary = "canary"
)

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
	if opts.Canary && opts.Stable {
		return nil, fmt.Errorf("escolha um canal só: --canary ou --stable")
	}

	// The channel sticks: a canary build keeps getting canaries until the
	// user asks for --stable.
	channel := ChannelStable
	if opts.Canary || (!opts.Stable && update.IsPrerelease(current)) {
		channel = ChannelCanary
	}

	var rel *models.Release
	var err error
	if channel == ChannelCanary {
		rel, err = s.newestRelease(ctx)
	} else {
		rel, err = s.releases.Latest(ctx)
	}
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
		Channel: channel,
		Asset:   *asset,
	}

	if update.IsReleaseVersion(current) {
		newer, err := update.IsNewer(current, rel.TagName)
		if err != nil {
			return nil, err
		}
		plan.Newer = newer
		// Leaving canary: the latest stable can be older than the running build.
		plan.Downgrade = opts.Stable && update.IsPrerelease(current) && !newer
	} else {
		// Forced update from dev: always treat remote as installable.
		plan.Newer = true
	}

	return plan, nil
}

// newestRelease returns the highest-versioned published release, canary
// builds included. A stable newer than every canary wins, as it should.
func (s *UpdateService) newestRelease(ctx context.Context) (*models.Release, error) {
	releases, err := s.releases.List(ctx)
	if err != nil {
		return nil, err
	}
	var newest *models.Release
	for i := range releases {
		rel := &releases[i]
		if rel.Draft || !update.IsReleaseVersion(rel.TagName) {
			continue
		}
		if newest == nil || update.CompareVersions(rel.TagName, newest.TagName) > 0 {
			newest = rel
		}
	}
	if newest == nil {
		return nil, fmt.Errorf("nenhuma release publicada encontrada")
	}
	return newest, nil
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
