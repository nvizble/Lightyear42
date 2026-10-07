package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/services"
)

type fakeExam struct {
	sess   *exam.Session
	passed bool
}

func (f *fakeExam) Status() (exam.Session, error) {
	if f.sess == nil {
		return exam.Session{}, services.ErrNoExamSession
	}
	return *f.sess, nil
}

func (f *fakeExam) Start(now time.Time, rank string, d time.Duration) (exam.Session, error) {
	f.sess = &exam.Session{Mode: exam.ModeExam, Rank: rank, Level: 1, Exercise: "first_word", Deadline: now.Add(d)}
	return *f.sess, nil
}

func (f *fakeExam) Grade(context.Context, time.Time) (services.GradeReport, error) {
	return services.GradeReport{Result: exam.Result{Passed: f.passed}, Graded: f.sess.Exercise, Session: *f.sess}, nil
}

func (f *fakeExam) Finish() (exam.Session, error) {
	s := *f.sess
	f.sess = nil
	return s, nil
}

func (f *fakeExam) Exercises() []exam.Exercise {
	return []exam.Exercise{{Rank: "02", Level: 1, Name: "first_word"}}
}

var appNow = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func newTestApp(t *testing.T, tabs []AppTab) (AppModel, *fakeExam) {
	t.Helper()
	fe := &fakeExam{}
	m := NewApp(AppOptions{Tabs: tabs, Exam: fe, Unavailable: "faça login"}, appNow)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	return next.(AppModel), fe
}

// run applies a message and, if it returns a command, feeds its result back
// (one level deep, enough for loads and grading).
func run(t *testing.T, m AppModel, msg tea.Msg) AppModel {
	t.Helper()
	next, cmd := m.Update(msg)
	m = next.(AppModel)
	if cmd != nil {
		if out := cmd(); out != nil {
			if _, batch := out.(tea.BatchMsg); !batch {
				next, _ = m.Update(out)
				m = next.(AppModel)
			}
		}
	}
	return m
}

func click(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

func TestAppTabsByKeyAndClick(t *testing.T) {
	loads := 0
	tabs := []AppTab{
		{Title: "Início", Load: func(context.Context) (string, error) { return "home", nil }},
		{Title: "Projetos", Load: func(context.Context) (string, error) { loads++; return "libft 125", nil }},
	}
	m, _ := newTestApp(t, tabs)

	m = run(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if m.active != 1 || !strings.Contains(m.View(), "libft 125") {
		t.Fatalf("tecla 2 deveria abrir Projetos:\n%s", m.View())
	}

	// Clicking the Exam tab label in the tab bar (row 0).
	examSpan := m.tabSpans()[2]
	m = run(t, m, click(examSpan.from+1, 0))
	if m.active != m.examTab() || !strings.Contains(m.View(), "Simulador de provas") {
		t.Fatalf("clique na aba Exam falhou:\n%s", m.View())
	}

	// Coming back to a loaded tab must not refetch.
	m = run(t, m, click(m.tabSpans()[1].from, 0))
	if loads != 1 {
		t.Fatalf("aba recarregada sem pedir: %d loads", loads)
	}
}

func TestAppExamFlowWithFooterButtons(t *testing.T) {
	m, fe := newTestApp(t, []AppTab{{Title: "Início"}})
	if m.active != m.examTab() {
		t.Fatal("sem login, o app deveria abrir na aba Exam")
	}

	clickButton := func(m AppModel, key string) AppModel {
		for _, b := range m.buttons() {
			if b.key == key {
				return run(t, m, click(b.from, m.height-1))
			}
		}
		t.Fatalf("botão %q não está no rodapé", key)
		return m
	}

	m = clickButton(m, "s")
	if fe.sess == nil || !strings.Contains(m.View(), "first_word") {
		t.Fatalf("botão começar não iniciou a prova:\n%s", m.View())
	}

	fe.passed = true
	m = clickButton(m, "g")
	if m.grading || !strings.Contains(m.View(), "SUCCESS") {
		t.Fatalf("grademe não mostrou o resultado:\n%s", m.View())
	}

	m = clickButton(m, "f")
	if fe.sess == nil || !m.confirmFinish {
		t.Fatal("encerrar precisa de confirmação")
	}
	m = clickButton(m, "f")
	if fe.sess != nil || m.examSess != nil {
		t.Fatal("segundo f deveria encerrar a prova")
	}
}

func TestAppScrollAndUnavailable(t *testing.T) {
	long := strings.Repeat("linha\n", 100)
	tabs := []AppTab{
		{Title: "Início", Load: func(context.Context) (string, error) { return long, nil }},
		{Title: "Slots"},
	}
	m, _ := newTestApp(t, tabs)
	m = run(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	m = run(t, m, tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	if m.scroll != 3 {
		t.Fatalf("roda do mouse deveria rolar 3 linhas, rolou %d", m.scroll)
	}
	m = run(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	m = run(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	m = run(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	m = run(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	if maxScroll := len(m.bodyLines()) - m.bodyHeight(); m.scroll != maxScroll {
		t.Fatalf("rolagem passou do fim: %d (máx %d)", m.scroll, maxScroll)
	}
	if got := strings.Count(m.View(), "\n") + 1; got != m.height {
		t.Fatalf("a tela deveria ter exatamente %d linhas, tem %d", m.height, got)
	}

	m = run(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if !strings.Contains(m.View(), "faça login") {
		t.Fatalf("aba sem Load deveria explicar o motivo:\n%s", m.View())
	}
}
