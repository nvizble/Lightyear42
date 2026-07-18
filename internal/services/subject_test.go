package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

type stubProjects struct {
	project *models.Project
	err     error
}

func (s stubProjects) BySlugOrName(context.Context, string) (*models.Project, error) {
	return s.project, s.err
}
func (s stubProjects) Attachments(context.Context, int) ([]models.Attachment, error) {
	return nil, nil
}
func (s stubProjects) SessionAttachments(context.Context, int) ([]models.Attachment, error) {
	return nil, nil
}
func (s stubProjects) Sessions(context.Context, int) ([]models.ProjectSession, error) {
	return nil, nil
}

type stubMe struct {
	user *models.User
}

func (s stubMe) Me(context.Context) (*models.User, error) {
	return s.user, nil
}

type stubDL struct {
	body string
	url  string
	err  error
}

func (s *stubDL) Download(_ context.Context, url string, w io.Writer) error {
	s.url = url
	if s.err != nil {
		return s.err
	}
	_, err := io.WriteString(w, s.body)
	return err
}

func TestSubjectService_WithPDFID(t *testing.T) {
	dir := t.TempDir()
	dl := &stubDL{body: "%PDF-push_swap"}
	svc := NewSubjectService(stubProjects{}, stubMe{
		user: &models.User{ProjectsUsers: []models.ProjectUser{
			{Project: models.Project{ID: 2687, Name: "push_swap", Slug: "42next-push_swap"}},
		}},
	}, dl)

	res, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query: "push_swap",
		Dir:   dir,
		Lang:  "en",
		PDFID: 189890,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Project.Slug != "42next-push_swap" {
		t.Fatalf("slug = %s, want enrolled 42next-push_swap", res.Project.Slug)
	}
	wantURL := "https://cdn.intra.42.fr/pdf/pdf/189890/en.subject.pdf"
	if dl.url != wantURL {
		t.Fatalf("download url = %q, want %q", dl.url, wantURL)
	}
	data, _ := os.ReadFile(res.Path)
	if string(data) != "%PDF-push_swap" {
		t.Fatalf("file = %q", data)
	}

	// Second run uses index.json without --pdf-id
	dl2 := &stubDL{body: "x"}
	svc2 := NewSubjectService(stubProjects{}, stubMe{
		user: &models.User{ProjectsUsers: []models.ProjectUser{
			{Project: models.Project{ID: 2687, Name: "push_swap", Slug: "42next-push_swap"}},
		}},
	}, dl2)
	res2, err := svc2.EnsureSubject(context.Background(), SubjectOptions{
		Query: "push_swap",
		Dir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res2.Cached {
		t.Fatal("deveria usar PDF em cache")
	}
}

func TestSubjectService_DiscoverFromHTML(t *testing.T) {
	dir := t.TempDir()
	html := `<a href="https://cdn.intra.42.fr/pdf/pdf/189890/en.subject.pdf">subject</a>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	t.Cleanup(srv.Close)

	dl := &stubDL{body: "%PDF"}
	svc := NewSubjectService(stubProjects{
		project: &models.Project{ID: 1, Name: "push_swap", Slug: "42next-push_swap"},
	}, stubMe{user: &models.User{}}, dl)
	svc.http = srv.Client()
	// Point discover at test server by temporarily patching — use custom discover via HTML server
	// Override: monkey by setting slug page — discoverPDFID uses fixed host.
	// So test discoverPDFID indirectly via rewriting: call discover with transport that maps host.
	svc.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = srv.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})

	res, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query: "42next-push_swap",
		Dir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dl.url != "https://cdn.intra.42.fr/pdf/pdf/189890/en.subject.pdf" {
		t.Fatalf("url = %s", dl.url)
	}
	if res.Cached {
		t.Fatal("primeira descarga")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMatchEnrolled(t *testing.T) {
	t.Parallel()
	list := []models.ProjectUser{
		{Project: models.Project{Name: "Libft", Slug: "42cursus-libft"}},
		{Project: models.Project{Name: "push_swap", Slug: "42next-push_swap"}},
	}
	p := matchEnrolled(list, "push_swap")
	if p == nil || p.Slug != "42next-push_swap" {
		t.Fatalf("got %+v", p)
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()
	if got := sanitizeFilename("push_swap"); got != "push_swap" {
		t.Fatalf("got %q", got)
	}
	_ = filepath.Separator
}
