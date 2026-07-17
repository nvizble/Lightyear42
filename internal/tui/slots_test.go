package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
)

func TestRenderSlots(t *testing.T) {
	t.Parallel()

	if out := RenderSlots(nil); !strings.Contains(out, "Nenhum slot futuro") {
		t.Errorf("empty = %q", out)
	}

	begin := time.Date(2026, 7, 18, 14, 0, 0, 0, time.Local)
	end := begin.Add(30 * time.Minute)
	out := RenderSlots([]models.Slot{
		{ID: 10, BeginAt: &begin, EndAt: &end},
		{ID: 11, BeginAt: &begin, EndAt: &end, ScaleTeam: &models.SlotScaleTeam{ID: 1}},
	})
	for _, want := range []string{"Slots futuros (2)", "10", "11", "livre", "agendado", "18/07 14:00"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}
