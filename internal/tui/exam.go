package tui

import (
	"fmt"
	"strings"
	"time"

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
	var b strings.Builder
	rank, level := "", 0
	for _, ex := range exercises {
		if ex.Rank != rank {
			if rank != "" {
				b.WriteString("\n\n")
			}
			b.WriteString(styleTitle.Render("Exam Rank " + ex.Rank))
			rank, level = ex.Rank, 0
		}
		if ex.Level != level {
			b.WriteString("\n" + styleLevel.Render(fmt.Sprintf("  nível %d", ex.Level)) + "\n   ")
			level = ex.Level
		}
		b.WriteString(" " + ex.Name)
	}
	return b.String()
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
