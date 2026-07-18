package repository

import (
	"context"
	"net/url"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

type projectsFakeAPI struct {
	byPath map[string]any
	err    error
}

func (f *projectsFakeAPI) Get(_ context.Context, path string, _ url.Values, out any) error {
	if f.err != nil {
		return f.err
	}
	val, ok := f.byPath[path]
	if !ok {
		return errNotFound{path: path}
	}
	switch dst := out.(type) {
	case *models.Project:
		*dst = val.(models.Project)
	case *[]models.Project:
		*dst = val.([]models.Project)
	case *[]models.Attachment:
		*dst = val.([]models.Attachment)
	case *[]models.ProjectSession:
		*dst = val.([]models.ProjectSession)
	default:
		tpanic("tipo não suportado")
	}
	return nil
}

type errNotFound struct{ path string }

func (e errNotFound) Error() string { return "not found: " + e.path }

func tpanic(msg string) { panic(msg) }

func TestProjectsRepository_BySlugOrName(t *testing.T) {
	t.Parallel()

	api := &projectsFakeAPI{byPath: map[string]any{
		"/projects/push_swap": models.Project{ID: 42, Name: "push_swap", Slug: "push_swap"},
	}}
	repo := NewProjectsRepository(api, NoopCache{})

	p, err := repo.BySlugOrName(context.Background(), "push_swap")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != 42 || p.Slug != "push_swap" {
		t.Fatalf("project = %+v", p)
	}
}

func TestProjectsRepository_Attachments(t *testing.T) {
	t.Parallel()

	api := &projectsFakeAPI{byPath: map[string]any{
		"/projects/42/attachments": []models.Attachment{
			{ID: 1, Name: "en.subject.pdf", URL: "https://cdn.example/en.subject.pdf"},
		},
	}}
	repo := NewProjectsRepository(api, NoopCache{})
	atts, err := repo.Attachments(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 1 || atts[0].DownloadURL() == "" {
		t.Fatalf("atts = %+v", atts)
	}
}
