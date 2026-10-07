package services

import (
	"context"
	"errors"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

type stubReleases struct {
	rel  *models.Release
	list []models.Release
	err  error
}

func (s stubReleases) Latest(context.Context) (*models.Release, error) {
	return s.rel, s.err
}

func (s stubReleases) List(context.Context) ([]models.Release, error) {
	return s.list, s.err
}

// release builds a release with a Darwin arm64 archive.
func release(tag string, prerelease bool) models.Release {
	return models.Release{
		TagName:    tag,
		Prerelease: prerelease,
		Assets: []models.ReleaseAsset{
			{Name: "lightyear_" + tag[1:] + "_Darwin_arm64.tar.gz", BrowserDownloadURL: "https://ex/" + tag},
		},
	}
}

type stubInstaller struct {
	path    string
	pathErr error
	install error
	called  bool
	url     string
	name    string
}

func (s *stubInstaller) TargetPath() (string, error) {
	return s.path, s.pathErr
}

func (s *stubInstaller) Install(_ context.Context, url, name string) error {
	s.called = true
	s.url = url
	s.name = name
	return s.install
}

func TestUpdateService_Check_Newer(t *testing.T) {
	t.Parallel()

	svc := NewUpdateService(stubReleases{rel: &models.Release{
		TagName: "v1.0.2",
		Assets: []models.ReleaseAsset{
			{Name: "lightyear_1.0.2_Linux_x86_64.tar.gz", BrowserDownloadURL: "https://ex/a.tar.gz"},
		},
	}}, nil)

	plan, err := svc.Check(context.Background(), UpdateOptions{
		Current: "v1.0.0",
		GOOS:    "linux",
		GOARCH:  "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Newer || plan.Latest != "v1.0.2" {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.Asset.BrowserDownloadURL != "https://ex/a.tar.gz" {
		t.Fatalf("asset = %+v", plan.Asset)
	}
}

func TestUpdateService_Check_UpToDate(t *testing.T) {
	t.Parallel()

	svc := NewUpdateService(stubReleases{rel: &models.Release{
		TagName: "v1.0.2",
		Assets: []models.ReleaseAsset{
			{Name: "lightyear_1.0.2_Darwin_arm64.tar.gz", BrowserDownloadURL: "https://ex/a.tar.gz"},
		},
	}}, nil)

	plan, err := svc.Check(context.Background(), UpdateOptions{
		Current: "v1.0.2",
		GOOS:    "darwin",
		GOARCH:  "arm64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Newer {
		t.Fatal("não deveria haver update")
	}
}

func TestUpdateService_Check_DevRequiresForce(t *testing.T) {
	t.Parallel()

	svc := NewUpdateService(stubReleases{}, nil)
	_, err := svc.Check(context.Background(), UpdateOptions{Current: "dev"})
	if err == nil {
		t.Fatal("esperava erro sem --force")
	}
}

func TestUpdateService_Check_DevWithForce(t *testing.T) {
	t.Parallel()

	svc := NewUpdateService(stubReleases{rel: &models.Release{
		TagName: "v1.0.2",
		Assets: []models.ReleaseAsset{
			{Name: "lightyear_1.0.2_Linux_arm64.tar.gz", BrowserDownloadURL: "https://ex/a.tar.gz"},
		},
	}}, nil)

	plan, err := svc.Check(context.Background(), UpdateOptions{
		Current: "dev",
		Force:   true,
		GOOS:    "linux",
		GOARCH:  "arm64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Newer {
		t.Fatal("force em dev deveria permitir install")
	}
}

func TestUpdateService_Apply(t *testing.T) {
	t.Parallel()

	inst := &stubInstaller{path: "/tmp/lightyear"}
	svc := NewUpdateService(nil, inst)
	err := svc.Apply(context.Background(), &UpdatePlan{
		Asset: models.ReleaseAsset{Name: "a.tar.gz", BrowserDownloadURL: "https://ex/a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !inst.called || inst.url != "https://ex/a" {
		t.Fatalf("installer = %+v", inst)
	}
}

func TestUpdateService_Apply_InstallError(t *testing.T) {
	t.Parallel()

	inst := &stubInstaller{path: "/usr/bin/lightyear", install: errors.New("permission denied")}
	svc := NewUpdateService(nil, inst)
	if err := svc.Apply(context.Background(), &UpdatePlan{
		Asset: models.ReleaseAsset{Name: "a.tar.gz", BrowserDownloadURL: "https://ex/a"},
	}); err == nil {
		t.Fatal("esperava erro")
	}
}

func TestUpdateService_Channels(t *testing.T) {
	t.Parallel()

	stable := release("v1.2.0", false)
	list := []models.Release{
		release("v1.3.0-canary.10", true), // newest first, like the GitHub API
		release("v1.3.0-canary.2", true),
		{TagName: "v1.4.0-canary.1", Prerelease: true, Draft: true}, // drafts never count
		stable,
		release("v1.1.0", false),
	}

	tests := []struct {
		name          string
		current       string
		canary        bool
		stableFlag    bool
		list          []models.Release
		wantLatest    string
		wantChannel   string
		wantNewer     bool
		wantDowngrade bool
	}{
		{name: "estável fica no estável", current: "1.2.0", list: list,
			wantLatest: "v1.2.0", wantChannel: ChannelStable},
		{name: "--canary entra no canal canary", current: "1.2.0", canary: true, list: list,
			wantLatest: "v1.3.0-canary.10", wantChannel: ChannelCanary, wantNewer: true},
		{name: "canary continua recebendo canaries", current: "1.3.0-canary.2", list: list,
			wantLatest: "v1.3.0-canary.10", wantChannel: ChannelCanary, wantNewer: true},
		{name: "canary já na última", current: "1.3.0-canary.10", list: list,
			wantLatest: "v1.3.0-canary.10", wantChannel: ChannelCanary},
		{name: "estável mais novo vence a canary", current: "1.3.0-canary.10",
			list:       append([]models.Release{release("v1.3.0", false)}, list...),
			wantLatest: "v1.3.0", wantChannel: ChannelCanary, wantNewer: true},
		{name: "--stable volta da canary (downgrade)", current: "1.3.0-canary.10", stableFlag: true, list: list,
			wantLatest: "v1.2.0", wantChannel: ChannelStable, wantDowngrade: true},
		{name: "--stable em versão estável não faz nada", current: "1.2.0", stableFlag: true, list: list,
			wantLatest: "v1.2.0", wantChannel: ChannelStable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewUpdateService(stubReleases{rel: &stable, list: tt.list}, nil)
			plan, err := svc.Check(context.Background(), UpdateOptions{
				Current: tt.current, Canary: tt.canary, Stable: tt.stableFlag,
				GOOS: "darwin", GOARCH: "arm64",
			})
			if err != nil {
				t.Fatal(err)
			}
			if plan.Latest != tt.wantLatest || plan.Channel != tt.wantChannel ||
				plan.Newer != tt.wantNewer || plan.Downgrade != tt.wantDowngrade {
				t.Fatalf("plan = %+v", plan)
			}
			if plan.Asset.BrowserDownloadURL != "https://ex/"+tt.wantLatest {
				t.Fatalf("asset errado: %+v", plan.Asset)
			}
		})
	}
}

func TestUpdateService_CanaryAndStableConflict(t *testing.T) {
	t.Parallel()

	svc := NewUpdateService(stubReleases{}, nil)
	if _, err := svc.Check(context.Background(), UpdateOptions{Current: "1.2.0", Canary: true, Stable: true}); err == nil {
		t.Fatal("--canary e --stable juntos deveriam falhar")
	}
}
