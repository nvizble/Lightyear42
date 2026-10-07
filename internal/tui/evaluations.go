package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
)

// RenderEvaluations lists upcoming evaluations for the CLI command.
// limit <= 0 shows every entry; a positive limit truncates with a summary.
func RenderEvaluations(evaluations []models.ScaleTeam, meLogin string, limit int) string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Próximas avaliações"))

	if len(evaluations) == 0 {
		b.WriteString("\n")
		b.WriteString(styleLabel.Render("Nenhuma avaliação agendada."))
		return styleCard.Render(b.String())
	}

	shown := evaluations
	if limit > 0 && len(shown) > limit {
		shown = shown[:limit]
	}
	for _, st := range shown {
		b.WriteString("\n")
		b.WriteString(styleLevel.Render(evaluationWhen(st.BeginAt)))
		b.WriteString("  ")
		b.WriteString(evaluationLine(st, meLogin))
		if st.IsCorrector(meLogin) {
			if url := st.FillURL(); url != "" {
				b.WriteString("\n")
				b.WriteString(styleLabel.Render("  → " + url))
			}
		}
	}
	if hidden := len(evaluations) - len(shown); hidden > 0 {
		b.WriteString("\n")
		b.WriteString(styleLabel.Render(fmt.Sprintf("… e mais %d", hidden)))
	}
	return styleCard.Render(b.String())
}

// evaluationWhen formats the slot start in local time.
func evaluationWhen(beginAt *time.Time) string {
	if beginAt == nil {
		return "--/-- --:--"
	}
	return beginAt.Local().Format("02/01 15:04")
}

// evaluationLine phrases one evaluation from the user's point of view.
func evaluationLine(st models.ScaleTeam, meLogin string) string {
	if st.IsCorrector(meLogin) {
		return styleValue.Render("você avalia ") + styleGood.Render(st.Team.Name)
	}

	corrector := st.Corrector.Login
	if corrector == "" {
		corrector = "alguém"
	}
	return styleGood.Render(corrector) +
		styleValue.Render(" avalia você") +
		styleLabel.Render(" ("+st.Team.Name+")")
}
