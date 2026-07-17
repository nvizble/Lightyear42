package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

func TestRenderSlotsCalendar_Empty(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local)
	out := RenderSlotsCalendar(nil, "", now)
	if !strings.Contains(out, "Nenhum slot aberto") {
		t.Errorf("out = %q", out)
	}
}

func TestRenderSlotsCalendar_Error(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local)
	out := RenderSlotsCalendar(nil, "Insufficient scope", now)
	for _, want := range []string{"Meus slots", "indisponível", "projects"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestRenderSlotsCalendar_Grid(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local) // sexta
	begin := time.Date(2026, 7, 17, 14, 0, 0, 0, time.Local)
	end := time.Date(2026, 7, 17, 16, 0, 0, 0, time.Local)
	bookedBegin := time.Date(2026, 7, 18, 10, 0, 0, 0, time.Local)
	bookedEnd := time.Date(2026, 7, 18, 10, 30, 0, 0, time.Local)

	out := RenderSlotsCalendar([]models.Slot{
		{ID: 1, BeginAt: &begin, EndAt: &end},
		{ID: 2, BeginAt: &bookedBegin, EndAt: &bookedEnd, ScaleTeam: &models.SlotScaleTeam{ID: 9}},
	}, "", now)

	for _, want := range []string{
		"Meus slots",
		"sex 17/07", "sáb 18/07",
		"14", "15", "10",
		"██ livre", "▓▓ agendado",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}
