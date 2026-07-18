package services

import (
	"context"
	"errors"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

type stubReleases struct {
	rel *models.Release
	err error
}

func (s stubReleases) Latest(context.Context) (*models.Release, error) {
	return s.rel, s.err
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
