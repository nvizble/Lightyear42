package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/nvizble/Lightyear42/internal/models"
)

const levelBarWidth = 20

// RenderUser renders a full profile card for a user.
func RenderUser(user *models.User) string {
	var b strings.Builder

	b.WriteString(styleTitle.Render(user.Displayname))
	b.WriteString(styleLabel.Render(" · " + user.Login))
	if user.Staff {
		b.WriteString(styleLabel.Render(" (staff)"))
	}
	b.WriteString("\n")

	if campus := mainCampusName(user); campus != "" {
		writeField(&b, "Campus", campus)
	}
	if user.Email != "" {
		writeField(&b, "Email", user.Email)
	}

	if cursus := user.MainCursus(); cursus != nil {
		b.WriteString("\n")
		b.WriteString(styleLabel.Render(cursus.Cursus.Name))
		if cursus.Grade != "" {
			b.WriteString(styleLabel.Render(" · " + cursus.Grade))
		}
		b.WriteString("\n")
		b.WriteString(styleLevel.Render(fmt.Sprintf("Level %.2f ", cursus.Level)))
		b.WriteString(levelBar(cursus.Level))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	writeField(&b, "Wallet", fmt.Sprintf("%d ₳", user.Wallet))
	writeField(&b, "Pontos de avaliação", fmt.Sprintf("%d", user.CorrectionPoint))

	if user.Location != "" {
		b.WriteString(styleLabel.Render("Local: "))
		b.WriteString(styleGood.Render(user.Location))
		b.WriteString("\n")
	} else {
		writeField(&b, "Local", "offline")
	}

	return styleCard.Render(strings.TrimRight(b.String(), "\n"))
}

// RenderUserList renders search results as a table.
func RenderUserList(users []models.UserSummary) string {
	if len(users) == 0 {
		return styleLabel.Render("Nenhum usuário encontrado.")
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("LOGIN", "NOME", "LOCAL").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return styleTableHeader.Padding(0, 1)
			}
			return styleTableCell
		})

	for _, user := range users {
		location := user.Location
		if location == "" {
			location = "-"
		}
		t.Row(user.Login, user.Displayname, location)
	}

	return t.Render()
}

// writeField appends a "Label: value" line to the card body.
func writeField(b *strings.Builder, label, value string) {
	b.WriteString(styleLabel.Render(label + ": "))
	b.WriteString(styleValue.Render(value))
	b.WriteString("\n")
}

// levelBar draws the fractional progress towards the next level.
func levelBar(level float64) string {
	fraction := level - math.Floor(level)
	filled := int(math.Round(fraction * levelBarWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", levelBarWidth-filled)
	percent := fmt.Sprintf(" %d%%", int(math.Round(fraction*100)))
	return styleLevel.Render(bar) + styleLabel.Render(percent)
}

// mainCampusName returns the first campus name, if any.
func mainCampusName(user *models.User) string {
	if len(user.Campus) == 0 {
		return ""
	}
	return user.Campus[0].Name
}
