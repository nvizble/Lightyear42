package services

import (
	"context"
	"encoding/json"
	"errors"
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
	err  error
}

func (s stubMe) Me(context.Context) (*models.User, error) {
	return s.user, s.err
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

func enrolledPushSwap() stubMe {
	return stubMe{
		user: &models.User{ProjectsUsers: []models.ProjectUser{
			{Project: models.Project{ID: 2687, Name: "push_swap", Slug: "42next-push_swap"}},
		}},
	}
}

func TestSubjectService_WithPDFID(t *testing.T) {
	dir := t.TempDir()
	dl := &stubDL{body: "%PDF-push_swap"}
	svc := NewSubjectService(stubProjects{}, enrolledPushSwap(), dl)

	res, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query: "push_swap",
		Dir:   dir,
		Lang:  "en",
		PDFID: 193464,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Project.Slug != "42next-push_swap" {
		t.Fatalf("slug = %s, want enrolled 42next-push_swap", res.Project.Slug)
	}
	wantURL := "https://cdn.intra.42.fr/pdf/pdf/193464/en.subject.pdf"
	if dl.url != wantURL {
		t.Fatalf("download url = %q, want %q", dl.url, wantURL)
	}
	data, _ := os.ReadFile(res.Path)
	if string(data) != "%PDF-push_swap" {
		t.Fatalf("file = %q", data)
	}

	dl2 := &stubDL{body: "x"}
	svc2 := NewSubjectService(stubProjects{}, enrolledPushSwap(), dl2)
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

func TestSubjectService_EmbeddedCatalog(t *testing.T) {
	dir := t.TempDir()
	dl := &stubDL{body: "%PDF"}
	svc := NewSubjectService(stubProjects{}, enrolledPushSwap(), dl)

	res, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query: "push_swap",
		Dir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dl.url != "https://cdn.intra.42.fr/pdf/pdf/193464/en.subject.pdf" {
		t.Fatalf("url = %s", dl.url)
	}
	if res.Cached {
		t.Fatal("primeira descarga")
	}
	// Warm local index from embedded catalog.
	idx := loadPDFIndex(dir)
	if idx["42next-push_swap"] != 193464 {
		t.Fatalf("index = %#v", idx)
	}
}

func TestSubjectService_LocalIndexBeatsEmbedded(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(`{"42next-push_swap": 111}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dl := &stubDL{body: "%PDF"}
	svc := NewSubjectService(stubProjects{}, enrolledPushSwap(), dl)

	_, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query: "push_swap",
		Dir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dl.url != "https://cdn.intra.42.fr/pdf/pdf/111/en.subject.pdf" {
		t.Fatalf("url = %s, want local id 111", dl.url)
	}
}

func TestSubjectService_DiscoverFromHTML(t *testing.T) {
	dir := t.TempDir()
	html := `<a href="https://cdn.intra.42.fr/pdf/pdf/424242/en.subject.pdf">subject</a>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	t.Cleanup(srv.Close)

	dl := &stubDL{body: "%PDF"}
	slug := "not-in-embedded-catalog"
	svc := NewSubjectService(stubProjects{
		project: &models.Project{ID: 1, Name: "discover", Slug: slug},
	}, stubMe{user: &models.User{}}, dl)
	svc.http = srv.Client()
	svc.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = srv.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})

	_, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query: slug,
		Dir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dl.url != "https://cdn.intra.42.fr/pdf/pdf/424242/en.subject.pdf" {
		t.Fatalf("url = %s", dl.url)
	}
}

func TestSubjectService_RequireAuth(t *testing.T) {
	svc := NewSubjectService(nil, nil, nil)
	_, err := svc.EnsureSubject(context.Background(), SubjectOptions{
		Query: "push_swap",
		Dir:   t.TempDir(),
	})
	if !errors.Is(err, ErrSubjectAuthRequired) {
		t.Fatalf("err = %v, want ErrSubjectAuthRequired", err)
	}

	svc2 := NewSubjectService(nil, stubMe{err: errors.New("no token")}, nil)
	_, err = svc2.EnsureSubject(context.Background(), SubjectOptions{
		Query: "push_swap",
		Dir:   t.TempDir(),
	})
	if !errors.Is(err, ErrSubjectAuthRequired) {
		t.Fatalf("err = %v", err)
	}
}

func TestSeedLocalIndex_CopiesEmbedded(t *testing.T) {
	dir := t.TempDir()
	if err := seedLocalIndex(dir); err != nil {
		t.Fatal(err)
	}
	idx := loadPDFIndex(dir)
	if idx["42next-push_swap"] != 193464 {
		t.Fatalf("seed incompleto: %#v", idx["42next-push_swap"])
	}
	if len(idx) < 100 {
		t.Fatalf("esperava catálogo grande, got %d", len(idx))
	}
	// Second seed must not overwrite a local override.
	idx["42next-push_swap"] = 1
	if err := writePDFIndex(dir, idx); err != nil {
		t.Fatal(err)
	}
	if err := seedLocalIndex(dir); err != nil {
		t.Fatal(err)
	}
	if loadPDFIndex(dir)["42next-push_swap"] != 1 {
		t.Fatal("seed sobrescreveu id local")
	}
}

func TestSetPDFID(t *testing.T) {
	dir := t.TempDir()
	svc := NewSubjectService(nil, enrolledPushSwap(), nil)
	res, err := svc.SetPDFID(context.Background(), dir, "push_swap", 193464)
	if err != nil {
		t.Fatal(err)
	}
	if res.Slug != "42next-push_swap" || res.ID != 193464 {
		t.Fatalf("%+v", res)
	}
	res2, err := svc.SetPDFID(context.Background(), dir, "42next-push_swap", 999)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Previous != 193464 || res2.ID != 999 {
		t.Fatalf("%+v", res2)
	}
	if loadPDFIndex(dir)["42next-push_swap"] != 999 {
		t.Fatal(loadPDFIndex(dir))
	}
}

func TestImportPDFIndex(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.json")
	if err := os.WriteFile(src, []byte(`{"zz-test-a": 1, "zz-test-b": 2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := ImportPDFIndex(dir, src)
	if err != nil {
		t.Fatal(err)
	}
	// seedLocalIndex first fills the embedded catalog; then merge adds 2 keys.
	if res.Added != 2 || res.Updated != 0 {
		t.Fatalf("%+v", res)
	}
	if res.Total < 200 {
		t.Fatalf("total=%d, esperava seed + imports", res.Total)
	}

	if err := os.WriteFile(src, []byte(`{"zz-test-b": 9, "zz-test-c": 3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = ImportPDFIndex(dir, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || res.Updated != 1 {
		t.Fatalf("%+v", res)
	}
	var idx map[string]int
	data, _ := os.ReadFile(res.Path)
	_ = json.Unmarshal(data, &idx)
	if idx["zz-test-b"] != 9 || idx["zz-test-a"] != 1 || idx["zz-test-c"] != 3 {
		t.Fatalf("%v", idx)
	}
}

func TestImportPDFIndex_Invalid(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(src, []byte(`[]`), 0o600)
	if _, err := ImportPDFIndex(dir, src); err == nil {
		t.Fatal("esperava erro")
	}
	_ = os.WriteFile(src, []byte(`{}`), 0o600)
	if _, err := ImportPDFIndex(dir, src); err == nil {
		t.Fatal("esperava catálogo vazio")
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
}
