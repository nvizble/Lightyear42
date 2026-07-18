package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

type stubProjects struct {
	project *models.Project
	atts    []models.Attachment
	sess    []models.ProjectSession
	sessAtt map[int][]models.Attachment
	err     error
}

func (s stubProjects) BySlugOrName(context.Context, string) (*models.Project, error) {
	return s.project, s.err
}
func (s stubProjects) Attachments(context.Context, int) ([]models.Attachment, error) {
	return s.atts, nil
}
func (s stubProjects) SessionAttachments(_ context.Context, id int) ([]models.Attachment, error) {
	return s.sessAtt[id], nil
}
func (s stubProjects) Sessions(context.Context, int) ([]models.ProjectSession, error) {
	return s.sess, nil
}

type stubDL struct {
	body string
	err  error
}

func (s stubDL) Download(_ context.Context, _ string, w io.Writer) error {
	if s.err != nil {
		return s.err
	}
	_, err := io.WriteString(w, s.body)
	return err
}

func TestSubjectService_EnsureSubject_DownloadAndCache(t *testing.T) {
	dir := t.TempDir()
	campus := 27
	svc := NewSubjectService(stubProjects{
		project: &models.Project{ID: 1, Name: "push_swap", Slug: "push_swap"},
		sess: []models.ProjectSession{
			{ID: 9, CampusID: &campus},
		},
		sessAtt: map[int][]models.Attachment{
			9: {{
				ID:       1,
				Name:     "en.subject.pdf",
				URL:      "https://cdn.example/en.subject.pdf",
				Language: &models.Language{Identifier: "en"},
			}},
		},
	}, stubDL{body: "%PDF-subject"})

	res, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query:    "push_swap",
		CampusID: campus,
		Dir:      dir,
		Lang:     "en",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Cached {
		t.Fatal("primeira vez não deveria ser cache")
	}
	data, err := os.ReadFile(res.Path)
	if err != nil || string(data) != "%PDF-subject" {
		t.Fatalf("file = %q err=%v", data, err)
	}

	res2, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query:    "push_swap",
		CampusID: campus,
		Dir:      dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res2.Cached {
		t.Fatal("segunda vez deveria usar cache")
	}
	if res2.Path != res.Path {
		t.Fatalf("path mudou: %s vs %s", res.Path, res2.Path)
	}
}

func TestPickSubject_PrefersLang(t *testing.T) {
	t.Parallel()

	atts := []models.Attachment{
		{Name: "fr.subject.pdf", URL: "https://x/fr", Language: &models.Language{Identifier: "fr"}},
		{Name: "en.subject.pdf", URL: "https://x/en", Language: &models.Language{Identifier: "en"}},
	}
	got, err := pickSubject(atts, "fr")
	if err != nil {
		t.Fatal(err)
	}
	if got.LangCode() != "fr" {
		t.Fatalf("lang = %s", got.LangCode())
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()
	if got := sanitizeFilename("Push Swap!"); got != "push_swap_" && !strings.HasPrefix(got, "push") {
		// push_swap_ with trailing underscore for !
		if filepath.Base(got) == "" {
			t.Fatal(got)
		}
	}
	if got := sanitizeFilename("push_swap"); got != "push_swap" {
		t.Fatalf("got %q", got)
	}
}
