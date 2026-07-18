package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/repository"
	"github.com/nvizble/Lightyear42/internal/subjects"
)

// ErrSubjectPDFUnknown means the CDN PDF id could not be resolved
// (API attachments are 403 for students; HTML scrape failed; no --pdf-id).
var ErrSubjectPDFUnknown = errors.New("subject PDF id desconhecido")

// SubjectDownloader fetches a remote URL into a writer (API/CDN).
type SubjectDownloader interface {
	Download(ctx context.Context, url string, w io.Writer) error
}

// MeProjects loads the authenticated user's project enrolments.
type MeProjects interface {
	Me(ctx context.Context) (*models.User, error)
}

// SubjectOptions controls subject resolution and download.
type SubjectOptions struct {
	Query string
	Lang  string
	Force bool
	Dir   string
	// PDFID forces the CDN document id (e.g. 189890 for push_swap).
	// When set, it is stored in the local slug→id index for later runs.
	PDFID int
}

// SubjectResult is the local path of the subject PDF.
type SubjectResult struct {
	Project  models.Project
	Path     string
	Cached   bool
	Language string
	URL      string
}

// ImportResult summarizes a local index merge.
type ImportResult struct {
	Path    string
	Added   int
	Updated int
	Total   int
}

// SetIDResult is the outcome of updating one slug→pdf-id mapping.
type SetIDResult struct {
	Slug     string
	ID       int
	Previous int
	Path     string
}

// SubjectService resolves, caches and opens project subject PDFs.
//
// Note: GET /v2/projects/:id/attachments returns 403 for student tokens
// (X-Application-Roles: None). Subject PDFs are served from the public CDN
// (cdn.intra.42.fr/pdf/pdf/{id}/{lang}.subject.pdf); the numeric id is
// resolved via --pdf-id, local index, embedded catalog, or Intra HTML.
type SubjectService struct {
	projects repository.Projects
	me       MeProjects
	dl       SubjectDownloader
	http     *http.Client
}

// NewSubjectService wires project lookup, /me enrolments and binary download.
func NewSubjectService(projects repository.Projects, me MeProjects, dl SubjectDownloader) *SubjectService {
	return &SubjectService{
		projects: projects,
		me:       me,
		dl:       dl,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

var cdnSubjectRe = regexp.MustCompile(`https://cdn\.intra\.42\.fr/pdf/pdf/(\d+)/([a-z]+)\.subject\.pdf`)

// EnsureSubject returns a local PDF path, downloading when missing.
func (s *SubjectService) EnsureSubject(ctx context.Context, opts SubjectOptions) (*SubjectResult, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return nil, fmt.Errorf("informe o nome ou slug do projeto")
	}
	if opts.Dir == "" {
		return nil, fmt.Errorf("diretório de subjects não configurado")
	}
	if err := os.MkdirAll(opts.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("criar diretório de subjects: %w", err)
	}
	// Ship the embedded catalog into the local index (fill missing keys only).
	_ = seedLocalIndex(opts.Dir)

	project, err := s.resolveProject(ctx, opts.Query)
	if err != nil {
		return nil, err
	}

	lang := strings.ToLower(strings.TrimSpace(opts.Lang))
	if lang == "" {
		lang = "en"
	}

	local := loadPDFIndex(opts.Dir)
	pdfID := opts.PDFID
	if pdfID == 0 {
		pdfID = local[project.Slug]
	}
	if pdfID == 0 {
		pdfID = subjects.Lookup(project.Slug)
	}
	if pdfID == 0 {
		pdfID, lang = s.discoverPDFID(ctx, project.Slug, lang)
	}
	if pdfID == 0 {
		intra := "https://projects.intra.42.fr/projects/" + project.Slug
		return &SubjectResult{Project: *project}, fmt.Errorf(
			"%w: a API 42 não expõe o PDF (attachments → 403 para alunos).\n"+
				"Abra o projeto na Intra, copie o id da URL do PDF\n"+
				"  (ex.: cdn.intra.42.fr/pdf/pdf/193464/en.subject.pdf → 193464)\n"+
				"e rode:\n"+
				"  lightyear subject set-id %s 193464\n"+
				"ou: lightyear subject %s --pdf-id 193464\n"+
				"Página do projeto: %s",
			ErrSubjectPDFUnknown, opts.Query, opts.Query, intra,
		)
	}

	if opts.PDFID != 0 || local[project.Slug] != pdfID {
		_ = savePDFIndex(opts.Dir, project.Slug, pdfID)
	}

	url := fmt.Sprintf(models.CDNSubjectURL, pdfID, lang)
	filename := sanitizeFilename(project.Slug) + "_" + lang + ".pdf"
	path := filepath.Join(opts.Dir, filename)

	if !opts.Force {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return &SubjectResult{
				Project: *project, Path: path, Cached: true, Language: lang, URL: url,
			}, nil
		}
	}

	tmp := path + ".partial"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("criar arquivo temporário: %w", err)
	}
	if err := s.dl.Download(ctx, url, f); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("baixar subject da CDN: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("gravar subject: %w", err)
	}

	return &SubjectResult{
		Project: *project, Path: path, Cached: false, Language: lang, URL: url,
	}, nil
}

// ImportPDFIndex merges a JSON catalog (slug→pdf-id) into the local index.
// Does not require API authentication.
func ImportPDFIndex(dir, path string) (*ImportResult, error) {
	if dir == "" {
		return nil, fmt.Errorf("diretório de subjects não configurado")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ler catálogo: %w", err)
	}
	src, err := subjects.Parse(data)
	if err != nil {
		return nil, err
	}
	if len(src) == 0 {
		return nil, fmt.Errorf("catálogo vazio ou sem entradas válidas")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("criar diretório de subjects: %w", err)
	}
	_ = seedLocalIndex(dir)

	dst := loadPDFIndex(dir)
	added, updated := subjects.Merge(dst, src)
	if err := writePDFIndex(dir, dst); err != nil {
		return nil, err
	}
	return &ImportResult{
		Path:    indexPath(dir),
		Added:   added,
		Updated: updated,
		Total:   len(dst),
	}, nil
}

// SetPDFID stores or updates the CDN pdf-id for a project in the local index.
// Resolves the project via /me when possible; otherwise matches the embedded catalog
// or accepts the query as a raw slug. Does not download the PDF.
func (s *SubjectService) SetPDFID(ctx context.Context, dir, query string, id int) (*SetIDResult, error) {
	if id <= 0 {
		return nil, fmt.Errorf("pdf-id deve ser um inteiro positivo")
	}
	if dir == "" {
		return nil, fmt.Errorf("diretório de subjects não configurado")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("criar diretório de subjects: %w", err)
	}
	_ = seedLocalIndex(dir)

	slug := ""
	if s != nil {
		if p, err := s.resolveProject(ctx, query); err == nil && p != nil && p.Slug != "" {
			slug = p.Slug
		}
	}
	if slug == "" {
		slug = subjects.MatchSlug(query)
	}
	if slug == "" {
		slug = strings.TrimSpace(query)
	}
	if slug == "" {
		return nil, fmt.Errorf("informe o nome ou slug do projeto")
	}

	local := loadPDFIndex(dir)
	prev := local[slug]
	local[slug] = id
	if err := writePDFIndex(dir, local); err != nil {
		return nil, err
	}
	return &SetIDResult{
		Slug:     slug,
		ID:       id,
		Previous: prev,
		Path:     indexPath(dir),
	}, nil
}

// seedLocalIndex copies missing entries from the embedded catalog into the local index.
func seedLocalIndex(dir string) error {
	if dir == "" {
		return fmt.Errorf("diretório de subjects não configurado")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	dst := loadPDFIndex(dir)
	if len(dst) == 0 {
		emb := subjects.Embedded()
		if len(emb) == 0 {
			return nil
		}
		return writePDFIndex(dir, emb)
	}
	added := subjects.MergeAbsent(dst, subjects.Embedded())
	if added == 0 {
		return nil
	}
	return writePDFIndex(dir, dst)
}

func (s *SubjectService) resolveProject(ctx context.Context, query string) (*models.Project, error) {
	query = strings.TrimSpace(query)
	if s != nil && s.me != nil {
		if me, err := s.me.Me(ctx); err == nil && me != nil {
			if p := matchEnrolled(me.ProjectsUsers, query); p != nil {
				return p, nil
			}
		}
	}
	if s != nil && s.projects != nil {
		return s.projects.BySlugOrName(ctx, query)
	}
	return nil, fmt.Errorf("projeto %q não resolvido (sem API); use o slug completo", query)
}

func matchEnrolled(list []models.ProjectUser, query string) *models.Project {
	lower := strings.ToLower(query)
	for i := range list {
		p := list[i].Project
		if strings.EqualFold(p.Name, query) || strings.EqualFold(p.Slug, query) {
			return &p
		}
	}
	for i := range list {
		p := list[i].Project
		if strings.Contains(strings.ToLower(p.Slug), lower) ||
			strings.Contains(strings.ToLower(p.Name), lower) {
			return &p
		}
	}
	return nil
}

// discoverPDFID tries to extract a CDN subject id from the Intra project page HTML.
func (s *SubjectService) discoverPDFID(ctx context.Context, slug, preferLang string) (int, string) {
	client := s.http
	if client == nil {
		client = http.DefaultClient
	}
	page := "https://projects.intra.42.fr/projects/" + slug
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, page, nil)
	if err != nil {
		return 0, preferLang
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "lightyear-cli")

	resp, err := client.Do(req)
	if err != nil {
		return 0, preferLang
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, preferLang
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return 0, preferLang
	}

	matches := cdnSubjectRe.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return 0, preferLang
	}
	for _, m := range matches {
		if preferLang != "" && m[2] == preferLang {
			var id int
			_, _ = fmt.Sscanf(m[1], "%d", &id)
			return id, m[2]
		}
	}
	var id int
	_, _ = fmt.Sscanf(matches[0][1], "%d", &id)
	return id, matches[0][2]
}

type pdfIndex = subjects.Index

func indexPath(dir string) string {
	return filepath.Join(dir, "index.json")
}

func loadPDFIndex(dir string) pdfIndex {
	data, err := os.ReadFile(indexPath(dir))
	if err != nil {
		return pdfIndex{}
	}
	idx, err := subjects.Parse(data)
	if err != nil {
		return pdfIndex{}
	}
	return idx
}

func savePDFIndex(dir string, slug string, id int) error {
	idx := loadPDFIndex(dir)
	if idx == nil {
		idx = pdfIndex{}
	}
	idx[slug] = id
	return writePDFIndex(dir, idx)
}

func writePDFIndex(dir string, idx pdfIndex) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath(dir), data, 0o600)
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "project"
	}
	return out
}
