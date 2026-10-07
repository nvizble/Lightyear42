package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/services"
)

// RenderExamSession renders the exam (or practice) in progress.
func RenderExamSession(s exam.Session, now time.Time) string {
	title := fmt.Sprintf("Exam Rank %s", s.Rank)
	if s.Mode == exam.ModePractice {
		title = "Prática"
	}

	rows := [][2]string{
		{"Exercício", styleLevel.Render(s.Exercise) + styleLabel.Render(fmt.Sprintf("  (nível %d)", s.Level))},
	}
	if s.Mode == exam.ModeExam {
		rows = append(rows,
			[2]string{"Nota", fmt.Sprintf("%d/100", s.Score)},
			[2]string{"Tempo", renderRemaining(s, now)},
		)
	}
	rows = append(rows,
		[2]string{"Tentativas", fmt.Sprint(s.Attempts)},
		[2]string{"Subject", filepath.Join(s.Workspace, "subjects", s.Exercise) + string(filepath.Separator)},
		[2]string{"Entrega", filepath.Join(s.Workspace, "rendu", s.Exercise) + string(filepath.Separator)},
	)

	var b strings.Builder
	b.WriteString(styleTitle.Render(title))
	for _, r := range rows {
		b.WriteString("\n")
		b.WriteString(styleLabel.Render(fmt.Sprintf("%-11s", r[0])))
		b.WriteString(r[1])
	}
	b.WriteString("\n\n")
	b.WriteString(styleLabel.Render("Quando terminar, rode `lightyear exam grademe`."))
	return styleCard.Render(b.String())
}

// RenderGradeReport renders the outcome of `lightyear exam grademe`.
func RenderGradeReport(r services.GradeReport, now time.Time) string {
	var b strings.Builder
	if !r.Passed {
		b.WriteString(styleFail.Render("✗ FAILURE") + styleLabel.Render("  "+r.Graded) + "\n\n")
		b.WriteString(r.Trace)
		b.WriteString("\n\n")
		b.WriteString(styleLabel.Render("Trace salvo em " + r.TracePath))
		return b.String()
	}

	b.WriteString(styleGood.Render("✓ SUCCESS") + styleLabel.Render("  "+r.Graded) + "\n\n")
	switch {
	case r.Completed && r.Session.Mode == exam.ModePractice:
		b.WriteString("Exercício concluído. Pratique outro com `lightyear exam practice <exercício>`.")
	case r.Completed:
		b.WriteString(styleGood.Render(fmt.Sprintf("Prova concluída: %d/100!", r.Session.Score)))
	default:
		b.WriteString(RenderExamSession(r.Session, now))
	}
	return b.String()
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
