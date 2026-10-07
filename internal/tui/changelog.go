package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/changelog"
)

var (
	mdCode = regexp.MustCompile("`([^`]+)`")
	mdBold = regexp.MustCompile(`\*\*([^*]+)\*\*`)
)

// RenderRelease renders what changed in one release: "Novidades da 1.2.1"
// and its notes.
func RenderRelease(r changelog.Release) string {
	return releaseTitle("Novidades da "+r.Version, r.Date) + "\n" + renderNotes(r.Notes)
}

// RenderChangelog renders every release, newest first.
func RenderChangelog(releases []changelog.Release) string {
	sections := make([]string, 0, len(releases))
	for _, r := range releases {
		sections = append(sections, releaseTitle(r.Version, r.Date)+"\n"+renderNotes(r.Notes))
	}
	return strings.Join(sections, "\n\n")
}

func releaseTitle(title, date string) string {
	if date != "" {
		return styleTitle.Render(title) + styleLabel.Render(" · "+date)
	}
	return styleTitle.Render(title)
}

// renderNotes turns the Markdown bullets into terminal text: "- " becomes a
// bullet, `code` is highlighted and **bold** is bold.
func renderNotes(notes string) string {
	bold := lipgloss.NewStyle().Bold(true)
	lines := strings.Split(notes, "\n")
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, "- "):
			l = "  • " + l[2:]
		case strings.HasPrefix(l, "  "):
			l = "    " + strings.TrimLeft(l, " ")
		}
		l = mdCode.ReplaceAllStringFunc(l, func(m string) string { return styleLevel.Render(m[1 : len(m)-1]) })
		l = mdBold.ReplaceAllStringFunc(l, func(m string) string { return bold.Render(m[2 : len(m)-2]) })
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}
