package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/joaodiniz/42cli/internal/models"
)

// RenderFriends renders the friends list.
func RenderFriends(friends []string) string {
	if len(friends) == 0 {
		return styleLabel.Render("Sua lista de amigos está vazia. Adicione com `42 friends add <login>`.")
	}

	var b strings.Builder
	b.WriteString(styleTitle.Render(fmt.Sprintf("Amigos (%d)", len(friends))))
	for _, login := range friends {
		b.WriteString("\n  ")
		b.WriteString(styleValue.Render(login))
	}
	return b.String()
}

// RenderFriendsOnline renders which friends are online and where.
func RenderFriendsOnline(online []models.Location, total int) string {
	if len(online) == 0 {
		return styleLabel.Render(fmt.Sprintf("Nenhum dos seus %d amigos está online neste campus.", total))
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers("LOGIN", "POSTO", "DESDE").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return styleTableHeader.Padding(0, 1)
			}
			return styleTableCell
		})

	for _, loc := range online {
		since := "-"
		if loc.BeginAt != nil {
			since = loc.BeginAt.Local().Format("02/01 15:04")
		}
		t.Row(styleGood.Render(loc.User.Login), loc.Host, since)
	}

	header := styleTitle.Render(fmt.Sprintf("%d de %d amigos online", len(online), total))
	return header + "\n" + t.Render()
}
