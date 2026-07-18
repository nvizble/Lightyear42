package models

import "testing"

func TestAttachment_CDNSubjectURL_PushSwapExample(t *testing.T) {
	t.Parallel()

	// push_swap subject on CDN: https://cdn.intra.42.fr/pdf/pdf/189890/en.subject.pdf
	a := Attachment{ID: 189890, Language: &Language{Identifier: "en"}}
	got := a.DownloadURL()
	want := "https://cdn.intra.42.fr/pdf/pdf/189890/en.subject.pdf"
	if got != want {
		t.Fatalf("DownloadURL = %q, want %q", got, want)
	}
}

func TestAttachment_DownloadURL_PrefersExplicitURL(t *testing.T) {
	t.Parallel()

	a := Attachment{
		ID:  189890,
		URL: "https://cdn.intra.42.fr/pdf/pdf/189890/fr.subject.pdf",
	}
	if got := a.DownloadURL(); got != a.URL {
		t.Fatalf("got %q", got)
	}
}

func TestAttachment_PDFID_FallsBackToBaseID(t *testing.T) {
	t.Parallel()

	a := Attachment{BaseID: 189890}
	if a.PDFID() != 189890 {
		t.Fatalf("PDFID = %d", a.PDFID())
	}
	want := "https://cdn.intra.42.fr/pdf/pdf/189890/pt.subject.pdf"
	if got := a.CDNSubjectURL("pt"); got != want {
		t.Fatalf("got %q", got)
	}
}
