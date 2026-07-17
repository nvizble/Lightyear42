package tui

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

func TestParseHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		want seat
		ok   bool
	}{
		{"c1r2p3", seat{1, 2, 3}, true},
		{"C10R12P5", seat{10, 12, 5}, true},
		{"c1r2p3.campus.42.fr", seat{1, 2, 3}, true},
		{"made-f0Ar7s6", seat{}, false},
		{"", seat{}, false},
	}

	for _, tt := range tests {
		got, ok := parseHost(tt.host)
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseHost(%q) = %+v, %v; want %+v, %v", tt.host, got, ok, tt.want, tt.ok)
		}
	}
}

func TestRenderCampusMap(t *testing.T) {
	t.Parallel()

	out := RenderCampusMap("São-Paulo", []models.Location{
		{Host: "c1r1p1", User: models.UserSummary{Login: "jdiniz"}},
		{Host: "c1r2p3", User: models.UserSummary{Login: "malima-m"}},
		{Host: "c2r1p1", User: models.UserSummary{Login: "other"}},
		{Host: "weird-host", User: models.UserSummary{Login: "ghost"}},
	}, nil)

	for _, want := range []string{
		"São-Paulo — 4 online",
		"Cluster 1", "Cluster 2",
		"jdiniz", "malima-m", "other",
		"r1", "r2", "p1", "p3",
		"Outros postos:", "ghost", "weird-host",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderCampusMap_ShowsEmptyClustersAndRows(t *testing.T) {
	t.Parallel()

	// Only clusters 2 and 3 have people; cluster 1 must still be drawn,
	// and every cluster gets the campus-wide grid (rows 1..3 × posts 1..2).
	out := RenderCampusMap("Porto", []models.Location{
		{Host: "c2r1p1", User: models.UserSummary{Login: "alpha"}},
		{Host: "c3r3p2", User: models.UserSummary{Login: "beta"}},
	}, nil)

	for _, want := range []string{
		"Cluster 1", "Cluster 2", "Cluster 3",
		"Cluster 1 — 0 online",
		"r1", "r2", "r3",
		"alpha", "beta",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderCampusMap_ConfiguredLayout(t *testing.T) {
	t.Parallel()

	layout := map[int]ClusterGrid{
		1: {Rows: 10, Posts: 4},
		2: {Rows: 2, Posts: 2}, // smaller than the occupied seats below
		3: {Rows: 3, Posts: 6},
	}

	out := RenderCampusMap("São-Paulo", []models.Location{
		{Host: "c2r5p3", User: models.UserSummary{Login: "alpha"}},
	}, layout)

	for _, want := range []string{
		// Cluster 1 uses its configured 10×4 grid even while empty.
		"Cluster 1 — 0 online", "r10", "p4",
		// Occupied seat outside the configured grid must still be drawn.
		"alpha", "r5", "p3",
		// Cluster 3 exists only in the layout but is still rendered.
		"Cluster 3 — 0 online", "p6",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderCampusMap_Empty(t *testing.T) {
	t.Parallel()

	if out := RenderCampusMap("Porto", nil, nil); !strings.Contains(out, "Ninguém online") {
		t.Errorf("output = %q, want empty-state message", out)
	}
}
