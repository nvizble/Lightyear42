package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
)

func TestRenderEvaluations(t *testing.T) {
	t.Parallel()

	t.Run("vazio", func(t *testing.T) {
		t.Parallel()
		out := RenderEvaluations(nil, "jdiniz", 0)
		if !strings.Contains(out, "Nenhuma avaliação agendada") {
			t.Errorf("out = %q, want estado vazio", out)
		}
	})

	t.Run("lista completa", func(t *testing.T) {
		t.Parallel()
		begin := time.Date(2026, 7, 18, 14, 0, 0, 0, time.Local)
		out := RenderEvaluations([]models.ScaleTeam{
			{
				ID:        42,
				BeginAt:   &begin,
				Corrector: models.ScaleTeamActor{Login: "jdiniz"},
				Team:      models.EvaluationTeam{Name: "malima-m's libft"},
			},
			{
				BeginAt:   &begin,
				Corrector: models.ScaleTeamActor{Login: "someone"},
				Team:      models.EvaluationTeam{Name: "jdiniz's gnl"},
			},
		}, "jdiniz", 0)

		for _, want := range []string{
			"Próximas avaliações",
			"18/07 14:00",
			"você avalia", "malima-m's libft",
			"someone", "avalia você", "jdiniz's gnl",
			"profile.intra.42.fr/scale_teams/42/edit",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("out sem %q:\n%s", want, out)
			}
		}
		if strings.Contains(out, "… e mais") {
			t.Error("lista completa não deveria truncar")
		}
	})

	t.Run("corrector invisível", func(t *testing.T) {
		t.Parallel()
		out := RenderEvaluations([]models.ScaleTeam{
			{Team: models.EvaluationTeam{Name: "secret team"}},
		}, "jdiniz", 0)
		for _, want := range []string{"alguém", "avalia você", "secret team", "--/-- --:--"} {
			if !strings.Contains(out, want) {
				t.Errorf("out sem %q:\n%s", want, out)
			}
		}
	})

	t.Run("limita e resume o excedente", func(t *testing.T) {
		t.Parallel()
		many := make([]models.ScaleTeam, 8)
		out := RenderEvaluations(many, "jdiniz", 5)
		if !strings.Contains(out, "… e mais 3") {
			t.Errorf("out deveria resumir as 3 avaliações excedentes:\n%s", out)
		}
	})
}
