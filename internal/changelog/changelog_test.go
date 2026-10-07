package changelog

import (
	"strings"
	"testing"

	"golang.org/x/mod/semver"
)

func TestParse(t *testing.T) {
	md := `# Changelog

intro ignorada

## [1.3.0-canary.2] — 2026-10-06

- novidade A
  continuação

## v1.2.1 - 2026-10-05
- correção B

## [0.1.0]
- primeira`
	got := Parse(md)
	if len(got) != 3 {
		t.Fatalf("esperava 3 versões, veio %d: %+v", len(got), got)
	}
	if got[0].Version != "1.3.0-canary.2" || got[0].Date != "2026-10-06" || got[0].Notes != "- novidade A\n  continuação" {
		t.Errorf("primeira seção: %+v", got[0])
	}
	if got[1].Version != "1.2.1" || got[1].Date != "2026-10-05" || got[1].Notes != "- correção B" {
		t.Errorf("seção sem colchetes: %+v", got[1])
	}
	if got[2].Version != "0.1.0" || got[2].Date != "" {
		t.Errorf("seção sem data: %+v", got[2])
	}
	if r, ok := Find(got, "v1.2.1"); !ok || r.Notes != "- correção B" {
		t.Errorf("Find com v: %+v %v", r, ok)
	}
	if _, ok := Find(got, "9.9.9"); ok {
		t.Error("versão inexistente não deveria ser achada")
	}
}

// The embedded CHANGELOG.md must stay well formed: every section is a valid
// semver version with a date and some notes, and no version repeats.
func TestEmbeddedChangelog(t *testing.T) {
	releases := Embedded()
	if len(releases) < 10 {
		t.Fatalf("changelog embutido com poucas versões: %d", len(releases))
	}
	seen := map[string]bool{}
	for _, r := range releases {
		if !semver.IsValid("v" + r.Version) {
			t.Errorf("versão inválida no changelog: %q", r.Version)
		}
		if r.Date == "" || strings.TrimSpace(r.Notes) == "" {
			t.Errorf("%s sem data ou sem notas", r.Version)
		}
		if seen[r.Version] {
			t.Errorf("%s repetida", r.Version)
		}
		seen[r.Version] = true
	}
}
