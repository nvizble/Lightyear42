package models

import (
	"fmt"
	"strings"
)

// CDNSubjectURL is the Intra CDN pattern for project subject PDFs.
// Example: https://cdn.intra.42.fr/pdf/pdf/189890/en.subject.pdf
const CDNSubjectURL = "https://cdn.intra.42.fr/pdf/pdf/%d/%s.subject.pdf"

// Attachment is a file or link linked to a project / project session
// (subject PDF, resources, etc.).
type Attachment struct {
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	Kind     string         `json:"kind"`
	Type     string         `json:"type"`
	URL      string         `json:"url"`
	BaseID   int            `json:"base_id"`
	Language *Language      `json:"language"`
	PDF      *AttachmentPDF `json:"pdf"`
}

// AttachmentPDF is the nested PDF payload returned by the 42 API.
type AttachmentPDF struct {
	PDF *AttachmentPDFFile `json:"pdf"`
}

// AttachmentPDFFile holds the CDN URL of a PDF attachment.
type AttachmentPDFFile struct {
	URL string `json:"url"`
}

// Language identifies an Intra localization (en, fr, pt, …).
type Language struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

// ProjectSession is a campus/cursus-specific instance of a project.
type ProjectSession struct {
	ID       int  `json:"id"`
	CampusID *int `json:"campus_id"`
	CursusID *int `json:"cursus_id"`
}

// PDFID is the numeric id used in the CDN path (attachment id, else base_id).
func (a Attachment) PDFID() int {
	if a.ID != 0 {
		return a.ID
	}
	return a.BaseID
}

// DownloadURL returns the best available URL for this attachment.
// When the API omits url (common for subjects), synthesizes the CDN path
// https://cdn.intra.42.fr/pdf/pdf/{id}/{lang}.subject.pdf
func (a Attachment) DownloadURL() string {
	if u := strings.TrimSpace(a.URL); u != "" {
		return u
	}
	if a.PDF != nil && a.PDF.PDF != nil {
		if u := strings.TrimSpace(a.PDF.PDF.URL); u != "" {
			return u
		}
	}
	// API often returns id (e.g. 189890 for push_swap) with url=null;
	// the file lives at cdn.intra.42.fr/pdf/pdf/{id}/{lang}.subject.pdf
	if a.PDFID() != 0 {
		return a.CDNSubjectURL("")
	}
	return ""
}

// CDNSubjectURL builds the Intra CDN URL for this attachment's subject PDF.
func (a Attachment) CDNSubjectURL(lang string) string {
	id := a.PDFID()
	if id == 0 {
		return ""
	}
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		lang = a.LangCode()
	}
	if lang == "" {
		lang = "en"
	}
	return fmt.Sprintf(CDNSubjectURL, id, lang)
}

// LangCode returns the language identifier (e.g. "en"), or empty.
func (a Attachment) LangCode() string {
	if a.Language == nil {
		return ""
	}
	return a.Language.Identifier
}
