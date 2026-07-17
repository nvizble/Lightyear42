package tui

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

func TestRenderProjects(t *testing.T) {
	t.Parallel()

	out := RenderProjects([]models.ProjectUser{
		{
			Status: models.ProjectStatusFinished, Validated: boolPtr(true), FinalMark: intPtr(115),
			Project: models.Project{Name: "libft"},
		},
		{
			Status: models.ProjectStatusFinished, Validated: boolPtr(false), FinalMark: intPtr(42),
			Project: models.Project{Name: "minishell"},
		},
		{
			Status:  models.ProjectStatusInProgress,
			Project: models.Project{Name: "get_next_line"},
		},
	})

	for _, want := range []string{
		"PROJETO", "libft", "✔ aprovado", "115",
		"minishell", "✘ reprovado", "42",
		"get_next_line", "em andamento",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderProjects_Empty(t *testing.T) {
	t.Parallel()

	if out := RenderProjects(nil); !strings.Contains(out, "Nenhum projeto") {
		t.Errorf("output = %q, want empty-state message", out)
	}
}
