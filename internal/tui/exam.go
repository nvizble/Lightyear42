package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/services"
)

// RenderExamSession renders the exam (or practice) in progress.
func RenderExamSession(s exam.Session, now time.Time) string {
	return examCard(s, now, true, "Quando terminar, rode `lightyear exam grademe`.")
}

// RenderGradeReport renders the outcome of `lightyear exam grademe`: the
// result and, when it passed, the next exercise.
func RenderGradeReport(r services.GradeReport, now time.Time) string {
	out := RenderGradeResult(r)
	if r.Passed && !r.Completed {
		out += "\n\n" + RenderExamSession(r.Session, now)
	}
	return out
}

// RenderExamCatalog lists the exercises grouped by rank and level.
func RenderExamCatalog(exercises []exam.Exercise) string {
	return ExamCatalogView(exercises, 0).Content
}

// ExamCatalogView is the catalog with each exercise's name as a hotspot
// (its ID is the name), the names of a level wrapped to width cells (0: one
// line per level).
func ExamCatalogView(exercises []exam.Exercise, width int) AppView {
	var lines []string
	var spots []Hotspot
	rank, level := "", 0
	for _, ex := range exercises {
		if ex.Rank != rank {
			if rank != "" {
				lines = append(lines, "")
			}
			lines = append(lines, styleTitle.Render("Exam Rank "+ex.Rank))
			rank, level = ex.Rank, 0
		}
		if ex.Level != level {
			lines = append(lines, styleLevel.Render(fmt.Sprintf("  nível %d", ex.Level)), "   ")
			level = ex.Level
		}
		last := len(lines) - 1
		col := lipgloss.Width(lines[last]) + 1
		if width > 0 && col > 4 && col+lipgloss.Width(ex.Name) > width {
			lines = append(lines, "   ")
			last, col = last+1, 4
		}
		lines[last] += " " + ex.Name
		spots = append(spots, Hotspot{Line: last, Col: col, Width: lipgloss.Width(ex.Name), Search: ex.Name,
			Info: fmt.Sprintf("treinar %s (nível %d, sem tempo)", ex.Name, ex.Level)})
	}
	return AppView{Content: strings.Join(lines, "\n"), Hotspots: spots}
}

func renderRemaining(s exam.Session, now time.Time) string {
	if s.TimeUp(now) {
		return styleFail.Render("esgotado")
	}
	left := s.Remaining(now).Round(time.Second)
	text := fmt.Sprintf("%02dh%02dm restantes", int(left.Hours()), int(left.Minutes())%60)
	if left < 15*time.Minute {
		return styleFail.Render(text)
	}
	return text
}
