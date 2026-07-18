package models

// Attachment is a file or link linked to a project / project session
// (subject PDF, resources, etc.).
type Attachment struct {
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	Kind     string         `json:"kind"`
	Type     string         `json:"type"`
	URL      string         `json:"url"`
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

// DownloadURL returns the best available URL for this attachment.
func (a Attachment) DownloadURL() string {
	if a.URL != "" {
		return a.URL
	}
	if a.PDF != nil && a.PDF.PDF != nil && a.PDF.PDF.URL != "" {
		return a.PDF.PDF.URL
	}
	return ""
}

// LangCode returns the language identifier (e.g. "en"), or empty.
func (a Attachment) LangCode() string {
	if a.Language == nil {
		return ""
	}
	return a.Language.Identifier
}
