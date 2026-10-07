package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/services"
)

var viewNow = time.Date(2026, 10, 7, 13, 12, 0, 0, time.Local)

func at(day, h, m int) *time.Time {
	v := time.Date(2026, 10, 7+day, h, m, 0, 0, time.Local)
	return &v
}

func TestRenderEvaluationRows(t *testing.T) {
	evals := []models.ScaleTeam{
		{ID: 1, BeginAt: at(0, 14, 30), Corrector: models.ScaleTeamActor{Login: "marvin"}, Correcteds: models.ScaleTeamActors{{Login: "bcosta"}}, Team: models.EvaluationTeam{Name: "ft_printf"}},
		{ID: 2, BeginAt: at(1, 10, 15), Corrector: models.ScaleTeamActor{Login: "rsilva"}, Team: models.EvaluationTeam{Name: "get_next_line"}},
		{ID: 3, BeginAt: at(3, 9, 0), Corrector: models.ScaleTeamActor{Login: "marvin"}, Team: models.EvaluationTeam{Name: "born2beroot"}},
	}
	out := RenderEvaluationRows(evals, "marvin", viewNow, 0)
	for _, want := range []string{"Próximas avaliações", "hoje 14:30", "você avalia", "em 1h18", "com bcosta",
		"amanhã 10:15", "você é avaliado", "em 21h", "por rsilva", "sáb 10/10 09:00", "em 2d"} {
		if !strings.Contains(out, want) {
			t.Errorf("faltou %q:\n%s", want, out)
		}
	}
	// Columns line up: the project column starts at the same width on every row.
	lines := strings.Split(out, "\n")[1:]
	column := func(line, word string) int { return lipgloss.Width(line[:strings.Index(line, word)]) }
	col := column(lines[0], "ft_printf")
	if column(lines[1], "get_next_line") != col || column(lines[2], "born2beroot") != col {
		t.Errorf("colunas desalinhadas:\n%s", out)
	}

	if out := RenderEvaluationRows(evals, "marvin", viewNow, 1); !strings.Contains(out, "… e mais 2") {
		t.Errorf("limite não resumiu o resto:\n%s", out)
	}
	if out := RenderEvaluationRows(nil, "marvin", viewNow, 0); !strings.Contains(out, "Nenhuma avaliação agendada.") {
		t.Errorf("lista vazia sem aviso:\n%s", out)
	}
}

func TestRenderCampusSeats(t *testing.T) {
	locs := []models.Location{
		{Host: "c1r1p1", User: models.UserSummary{Login: "marvin"}},
		{Host: "c1r1p3", User: models.UserSummary{Login: "tlima"}},
		{Host: "c1r2p2", User: models.UserSummary{Login: "ana"}},
		{Host: "lab-mac", User: models.UserSummary{Login: "zeca"}},
	}
	layout := map[int]ClusterGrid{1: {Rows: 2, Posts: 4}}
	out := RenderCampusSeats("São Paulo", locs, layout, []string{"TLima"}, "marvin")

	if !strings.Contains(out, "São Paulo — 4 online") || !strings.Contains(out, "Cluster 1 · 3 online") {
		t.Fatalf("cabeçalhos errados:\n%s", out)
	}
	// One line per row, one seat glyph per post.
	var rows []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, seatGlyph) {
			rows = append(rows, l)
		}
	}
	if len(rows) != 2 {
		t.Fatalf("esperava 2 fileiras, veio %d:\n%s", len(rows), out)
	}
	for _, r := range rows {
		if n := strings.Count(r, seatGlyph); n != 4 {
			t.Errorf("fileira com %d postos, esperava 4: %q", n, r)
		}
	}
	if !strings.Contains(out, "Amigos online  tlima c1r1p3") {
		t.Errorf("amigo online sem posto (a busca ignora maiúsculas):\n%s", out)
	}
	if !strings.Contains(out, "zeca @ lab-mac") {
		t.Errorf("posto fora do padrão sumiu:\n%s", out)
	}
}

func TestBanner(t *testing.T) {
	for _, word := range []string{"SUCCESS", "FAILURE"} {
		lines := strings.Split(banner("✓", word), "\n")
		if len(lines) != 3 {
			t.Fatalf("%s: esperava 3 linhas, veio %d", word, len(lines))
		}
		w := lipgloss.Width(lines[0])
		for _, l := range lines {
			if lipgloss.Width(l) != w {
				t.Errorf("%s: linhas com larguras diferentes:\n%s", word, strings.Join(lines, "\n"))
			}
		}
	}
	if got := strings.Split(banner("✓", "SUCCESS"), "\n")[0]; got != "  ▄▀▀▀ █  █ ▄▀▀▀ ▄▀▀▀ █▀▀▀ ▄▀▀▀ ▄▀▀▀" {
		t.Errorf("primeira linha do SUCCESS mudou: %q", got)
	}
}

func TestRenderGradeResult(t *testing.T) {
	pass := services.GradeReport{
		Result:  exam.Result{Passed: true, Tests: []exam.TestOutcome{{Args: []string{"a"}, Passed: true}, {Passed: true}}},
		Graded:  "first_word",
		Session: exam.Session{Mode: exam.ModeExam, Level: 2, Score: 25},
		Levels:  []int{1, 2, 3, 4},
	}
	out := RenderGradeResult(pass)
	for _, want := range []string{"first_word · 2/2 testes", "nível 1 ✓", "nível 2", "25/100", "subindo de nível…"} {
		if !strings.Contains(out, want) {
			t.Errorf("sucesso sem %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "nível 2 ✓") {
		t.Errorf("nível atual marcado como concluído:\n%s", out)
	}

	fail := services.GradeReport{
		Result: exam.Result{Trace: "esperado: \"a\"", Tests: []exam.TestOutcome{{Args: []string{"x y"}, Passed: true}, {Args: []string{"z"}}}},
		Graded: "first_word", TracePath: "/tmp/t.trace",
	}
	out = RenderGradeResult(fail)
	for _, want := range []string{strings.Split(banner("✗", "FAILURE"), "\n")[1], "✓ first_word \"x y\"", "✗ first_word \"z\"", "esperado: \"a\"", "Trace salvo em /tmp/t.trace"} {
		if !strings.Contains(out, want) {
			t.Errorf("falha sem %q:\n%s", want, out)
		}
	}
}

func TestAppGradingSpinner(t *testing.T) {
	m, fe := newTestApp(t, []AppTab{{Title: "Início"}})
	m = run(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	fe.passed = true

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = next.(AppModel)
	if !m.grading || !strings.Contains(m.View(), "⠋ Corrigindo…") {
		t.Fatalf("correção sem spinner:\n%s", m.View())
	}
	next, _ = m.Update(appSpinMsg{})
	if v := next.(AppModel).View(); !strings.Contains(v, "⠙ Corrigindo…") {
		t.Fatalf("spinner não andou:\n%s", v)
	}
}
