package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/nvizble/Lightyear42/internal/models"
)

// statusLabels translates API project statuses for display.
var statusLabels = map[string]string{
	models.ProjectStatusInProgress: "em andamento",
	"searching_a_group":            "procurando grupo",
	"creating_group":               "criando grupo",
	"waiting_for_correction":       "aguardando correção",
	"waiting_to_start":             "aguardando início",
}

// RenderProjects renders project enrolments as a table.
func RenderProjects(projects []models.ProjectUser) string {
	if len(projects) == 0 {
		return styleLabel.Render("Nenhum projeto encontrado.")
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("PROJETO", "STATUS", "NOTA").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return styleTableHeader.Padding(0, 1)
			}
			return styleTableCell
		})

	for _, pu := range projects {
		t.Row(pu.Project.Name, projectStatus(pu), projectMark(pu))
	}

	return t.Render()
}

// projectStatus formats the status column with result markers.
func projectStatus(pu models.ProjectUser) string {
	if pu.Status == models.ProjectStatusFinished {
		if pu.Passed() {
			return styleGood.Render("✔ aprovado")
		}
		return styleFail.Render("✘ reprovado")
	}

	label, ok := statusLabels[pu.Status]
	if !ok {
		label = strings.ReplaceAll(pu.Status, "_", " ")
	}
	return styleLevel.Render(label)
}

// projectMark formats the final mark column.
func projectMark(pu models.ProjectUser) string {
	if pu.FinalMark == nil {
		return styleLabel.Render("-")
	}
	return fmt.Sprintf("%d", *pu.FinalMark)
}
