package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/repository"
)

// SubjectDownloader fetches a remote URL into a writer (API/CDN).
type SubjectDownloader interface {
	Download(ctx context.Context, url string, w io.Writer) error
}

// SubjectOptions controls subject resolution and download.
type SubjectOptions struct {
	// Query is the project slug or name (e.g. "push_swap").
	Query string
	// Lang prefers this language code (en, fr, pt, …). Empty = best effort.
	Lang string
	// CampusID prefers attachments from this campus' project session (0 = ignore).
	CampusID int
	// Force re-downloads even when a cached file exists.
	Force bool
	// Dir is the local cache directory for PDF files.
	Dir string
}

// SubjectResult is the local path of the subject PDF.
type SubjectResult struct {
	Project  models.Project
	Path     string
	Cached   bool
	Language string
	URL      string
}

// SubjectService resolves, caches and opens project subject PDFs.
type SubjectService struct {
	projects repository.Projects
	dl       SubjectDownloader
}

// NewSubjectService wires project lookup and binary download.
func NewSubjectService(projects repository.Projects, dl SubjectDownloader) *SubjectService {
	return &SubjectService{projects: projects, dl: dl}
}

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

	project, err := s.projects.BySlugOrName(ctx, opts.Query)
	if err != nil {
		return nil, err
	}

	atts, err := s.collectAttachments(ctx, project.ID, opts.CampusID)
	if err != nil {
		return nil, err
	}

	att, err := pickSubject(atts, opts.Lang)
	if err != nil {
		return nil, fmt.Errorf("projeto %s: %w", project.Slug, err)
	}

	lang := att.LangCode()
	if lang == "" {
		lang = "und"
	}
	filename := sanitizeFilename(project.Slug) + "_" + lang + ".pdf"
	path := filepath.Join(opts.Dir, filename)

	if !opts.Force {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return &SubjectResult{
				Project:  *project,
				Path:     path,
				Cached:   true,
				Language: lang,
				URL:      att.DownloadURL(),
			}, nil
		}
	}

	url := att.DownloadURL()
	if url == "" {
		return nil, fmt.Errorf("attachment %q sem URL de download", att.Name)
	}

	tmp := path + ".partial"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("criar arquivo temporário: %w", err)
	}
	if err := s.dl.Download(ctx, url, f); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("baixar subject: %w", err)
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
		Project:  *project,
		Path:     path,
		Cached:   false,
		Language: lang,
		URL:      url,
	}, nil
}

func (s *SubjectService) collectAttachments(ctx context.Context, projectID, campusID int) ([]models.Attachment, error) {
	var all []models.Attachment

	if campusID > 0 {
		sessions, err := s.projects.Sessions(ctx, projectID)
		if err == nil {
			for _, sess := range sessions {
				if sess.CampusID != nil && *sess.CampusID == campusID {
					atts, err := s.projects.SessionAttachments(ctx, sess.ID)
					if err == nil {
						all = append(all, atts...)
					}
				}
			}
			// Fallback: sessions without campus (global) when campus-specific empty.
			if len(all) == 0 {
				for _, sess := range sessions {
					if sess.CampusID == nil {
						atts, err := s.projects.SessionAttachments(ctx, sess.ID)
						if err == nil {
							all = append(all, atts...)
						}
					}
				}
			}
		}
	}

	projectAtts, err := s.projects.Attachments(ctx, projectID)
	if err != nil && len(all) == 0 {
		return nil, err
	}
	all = append(all, projectAtts...)

	if len(all) == 0 {
		return nil, fmt.Errorf("nenhum attachment encontrado")
	}
	return all, nil
}

func pickSubject(atts []models.Attachment, preferLang string) (*models.Attachment, error) {
	preferLang = strings.ToLower(strings.TrimSpace(preferLang))

	var pdfs []models.Attachment
	for _, a := range atts {
		if a.DownloadURL() == "" {
			continue
		}
		if looksLikeSubject(a) {
			pdfs = append(pdfs, a)
		}
	}
	if len(pdfs) == 0 {
		// Any downloadable PDF-like attachment.
		for _, a := range atts {
			if a.DownloadURL() == "" {
				continue
			}
			if strings.EqualFold(a.Type, "Pdf") || strings.HasSuffix(strings.ToLower(a.Name), ".pdf") ||
				strings.Contains(strings.ToLower(a.URL), ".pdf") {
				pdfs = append(pdfs, a)
			}
		}
	}
	if len(pdfs) == 0 {
		return nil, fmt.Errorf("nenhum PDF de subject disponível")
	}

	if preferLang != "" {
		for i := range pdfs {
			if strings.EqualFold(pdfs[i].LangCode(), preferLang) {
				return &pdfs[i], nil
			}
		}
	}
	// Prefer English, then first.
	for i := range pdfs {
		if strings.EqualFold(pdfs[i].LangCode(), "en") {
			return &pdfs[i], nil
		}
	}
	return &pdfs[0], nil
}

func looksLikeSubject(a models.Attachment) bool {
	blob := strings.ToLower(a.Name + " " + a.Slug + " " + a.Kind + " " + a.Type + " " + a.DownloadURL())
	return strings.Contains(blob, "subject")
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
