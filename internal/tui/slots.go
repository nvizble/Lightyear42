package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/nvizble/Lightyear42/internal/models"
)

// RenderSlots lists evaluation availability slots.
func RenderSlots(slots []models.Slot) string {
	if len(slots) == 0 {
		return styleLabel.Render("Nenhum slot futuro aberto. Abra com `lightyear slots open`.")
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("ID", "INÍCIO", "FIM", "STATUS").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return styleTableHeader.Padding(0, 1)
			}
			return styleTableCell
		})

	for _, slot := range slots {
		status := styleGood.Render("livre")
		if slot.Booked() {
			status = styleAccent.Render("agendado")
		}
		t.Row(
			fmt.Sprintf("%d", slot.ID),
			slotTime(slot.BeginAt),
			slotTime(slot.EndAt),
			status,
		)
	}

	header := styleTitle.Render(fmt.Sprintf("Slots futuros (%d)", len(slots)))
	return header + "\n" + t.Render()
}

// styleAccent reuses the level/orange color for booked status.
var styleAccent = lipgloss.NewStyle().Foreground(colorAccent)

func slotTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Local().Format("02/01 15:04")
}
