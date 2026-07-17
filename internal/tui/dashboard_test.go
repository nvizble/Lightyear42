package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joaodiniz/42cli/internal/models"
	"github.com/joaodiniz/42cli/internal/services"
)

type stubProvider struct {
	snap *services.DashboardSnapshot
	err  error
}

func (s stubProvider) Snapshot(context.Context, int) (*services.DashboardSnapshot, error) {
	return s.snap, s.err
}

func sampleSnapshot() *services.DashboardSnapshot {
	begin := time.Date(2026, 7, 17, 9, 0, 0, 0, time.Local)
	return &services.DashboardSnapshot{
		Me: &models.User{
			Login:           "jdiniz",
			Wallet:          100,
			CorrectionPoint: 5,
		},
		CampusID:   28,
		CampusName: "São-Paulo",
		Locations: []models.Location{
			{Host: "c2r1p1", User: models.UserSummary{Login: "malima-m"}, BeginAt: &begin},
			{Host: "c2r1p2", User: models.UserSummary{Login: "other"}},
		},
		Friends:       []string{"malima-m"},
		FriendsOnline: []models.Location{{Host: "c2r1p1", User: models.UserSummary{Login: "malima-m"}, BeginAt: &begin}},
		Evaluations: []models.ScaleTeam{
			{
				BeginAt:   timePtr(time.Date(2026, 7, 18, 14, 0, 0, 0, time.Local)),
				Corrector: models.ScaleTeamActor{Login: "jdiniz"},
				Team:      models.EvaluationTeam{Name: "malima-m's libft"},
			},
			{
				BeginAt:   timePtr(time.Date(2026, 7, 19, 10, 0, 0, 0, time.Local)),
				Corrector: models.ScaleTeamActor{Login: "someone"},
				Team:      models.EvaluationTeam{Name: "jdiniz's get_next_line"},
			},
		},
		Slots: []models.Slot{
			{
				ID:      1,
				BeginAt: timePtr(time.Date(2026, 7, 17, 14, 0, 0, 0, time.Local)),
				EndAt:   timePtr(time.Date(2026, 7, 17, 15, 0, 0, 0, time.Local)),
			},
		},
		TakenAt: time.Date(2026, 7, 17, 12, 30, 0, 0, time.Local),
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func TestDashboard_QuitKeys(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"q", "esc", "ctrl+c"} {
		m := NewDashboard(stubProvider{}, DashboardOptions{})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if key != "q" {
			// esc/ctrl+c are special keys, not runes.
			switch key {
			case "esc":
				_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			case "ctrl+c":
				_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			}
		}
		if cmd == nil {
			t.Fatalf("tecla %q deveria produzir comando de saída", key)
		}
		if msg := cmd(); msg != tea.Quit() {
			t.Fatalf("tecla %q produziu %v, want tea.Quit", key, msg)
		}
	}
}

func TestDashboard_SnapshotUpdatesView(t *testing.T) {
	t.Parallel()

	m := NewDashboard(stubProvider{}, DashboardOptions{
		Layout: map[int]ClusterGrid{1: {Rows: 10, Posts: 4}, 2: {Rows: 12, Posts: 6}},
	})

	updated, _ := m.Update(snapshotMsg{snap: sampleSnapshot()})
	view := updated.(DashboardModel).View()

	for _, want := range []string{
		"São-Paulo", "2 online",
		"jdiniz", "100 ₳", "5 pontos",
		"Ocupação por cluster", "Cluster 1", "Cluster 2", "2/72",
		"Próximas avaliações",
		"18/07 14:00", "você avalia", "malima-m's libft",
		"19/07 10:00", "someone", "avalia você", "jdiniz's get_next_line",
		"Meus slots", "sex 17/07",
		"1 de 1 amigos online", "malima-m",
		"12:30:00",
		"r atualizar",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view sem %q:\n%s", want, view)
		}
	}
}

func TestDashboard_ErrorKeepsPreviousData(t *testing.T) {
	t.Parallel()

	m := NewDashboard(stubProvider{}, DashboardOptions{})
	updated, _ := m.Update(snapshotMsg{snap: sampleSnapshot()})
	updated, _ = updated.(DashboardModel).Update(snapshotMsg{err: errors.New("api fora do ar")})

	view := updated.(DashboardModel).View()
	if !strings.Contains(view, "jdiniz") {
		t.Error("view deveria manter os dados anteriores após erro")
	}
	if !strings.Contains(view, "api fora do ar") {
		t.Error("view deveria mostrar o erro de atualização")
	}
}

func TestDashboard_ErrorBeforeFirstSnapshot(t *testing.T) {
	t.Parallel()

	m := NewDashboard(stubProvider{}, DashboardOptions{})
	updated, _ := m.Update(snapshotMsg{err: errors.New("sem sessão")})

	view := updated.(DashboardModel).View()
	if !strings.Contains(view, "sem sessão") {
		t.Errorf("view = %q, want mensagem de erro", view)
	}
}

func TestDashboard_TickTriggersFetch(t *testing.T) {
	t.Parallel()

	m := NewDashboard(stubProvider{snap: sampleSnapshot()}, DashboardOptions{})
	// Simulate first load done so the tick actually refreshes.
	updated, _ := m.Update(snapshotMsg{snap: sampleSnapshot()})

	_, cmd := updated.(DashboardModel).Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("tick deveria agendar refresh + próximo tick")
	}
}

func TestClusterGridCapacity(t *testing.T) {
	t.Parallel()

	if got := (ClusterGrid{Rows: 12, Posts: 6}).Capacity(); got != 72 {
		t.Errorf("Capacity = %d, want 72", got)
	}
	if got := (ClusterGrid{Rows: 13, Posts: 6, Seats: 64}).Capacity(); got != 64 {
		t.Errorf("Capacity com Seats = %d, want 64", got)
	}
}

func TestOccupancyLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                     string
		cluster, count, capacity int
		want                     []string
	}{
		{"com capacidade", 2, 57, 72, []string{"Cluster 2", "57/72", "79%"}},
		{"sem layout", 4, 3, 0, []string{"Cluster 4", "3 online"}},
		{"acima da capacidade", 1, 50, 40, []string{"50/40", "100%"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := occupancyLine(tt.cluster, tt.count, tt.capacity)
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("occupancyLine = %q, falta %q", out, want)
				}
			}
		})
	}
}
