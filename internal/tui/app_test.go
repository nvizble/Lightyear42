package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
	// subject and files, when set, are what EditTargets returns.
	subject string
	files   []string
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

func (f *fakeExam) EditTargets(exam.Session) (string, []string, error) {
	if f.subject != "" {
		return f.subject, f.files, nil
	}
	return "subject.pt.txt", []string{"rendu/first_word.c"}, nil
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
// (one level deep, enough for loads and grading; batches run every command).
func run(t *testing.T, m AppModel, msg tea.Msg) AppModel {
	t.Helper()
	next, cmd := m.Update(msg)
	m = next.(AppModel)
	if cmd == nil {
		return m
	}
	out := cmd()
	msgs := []tea.Msg{out}
	if batch, ok := out.(tea.BatchMsg); ok {
		msgs = msgs[:0]
		for _, c := range batch {
			if c != nil {
				msgs = append(msgs, c())
			}
		}
	}
	for _, o := range msgs {
		if o != nil {
			next, _ = m.Update(o)
			m = next.(AppModel)
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
	if m.grading || !strings.Contains(m.View(), strings.Split(banner("✓", "SUCCESS"), "\n")[0]) {
		t.Fatalf("grademe não mostrou o resultado:\n%s", m.View())
	}
	if n := strings.Count(m.View(), "Exam Rank 02"); n != 1 {
		t.Fatalf("o cartão da prova deveria aparecer uma vez, apareceu %d:\n%s", n, m.View())
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

func TestAppEditOpensExternalEditor(t *testing.T) {
	m, _ := newTestApp(t, []AppTab{{Title: "Início"}})
	m = run(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")}); cmd == nil {
		t.Fatal("E deveria abrir o $EDITOR")
	}
	m = run(t, m, appEditedMsg{})
	if !strings.Contains(m.View(), "Aperte g para corrigir") {
		t.Fatalf("ao voltar do editor deveria sugerir o grademe:\n%s", m.View())
	}
}

// e opens lightyear's editor over the app: subject read-only on the left,
// the code on the right; keys go to it, and :wq comes back to the app.
func TestAppEmbeddedEditor(t *testing.T) {
	dir := t.TempDir()
	subject, code := filepath.Join(dir, "subject.pt.txt"), filepath.Join(dir, "rendu", "first_word", "first_word.c")
	if err := os.WriteFile(subject, []byte("Assignment name  : first_word\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, fe := newTestApp(t, []AppTab{{Title: "Início"}})
	fe.subject, fe.files = subject, []string{code}
	key := func(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
	m = run(t, m, key("s"))
	m = run(t, m, key("e"))
	if m.editor == nil || !strings.Contains(m.View(), "Assignment name  : first_word") || !strings.Contains(m.View(), "first_word.c") {
		t.Fatalf("e abre o editor com o subject e o código:\n%s", m.View())
	}
	// Keys go to the editor: q isn't the app's quit there.
	for _, msg := range []tea.Msg{key("q"), tea.KeyMsg{Type: tea.KeyEsc}, key("i"), key("int\tmain(void);"), tea.KeyMsg{Type: tea.KeyEsc}, key(":wq")} {
		m = run(t, m, msg)
	}
	if m.editor == nil {
		t.Fatal("o editor deveria continuar aberto até o :wq")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(AppModel)
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("o :wq do editor não pode fechar o app")
		}
	}
	if m.editor != nil || !strings.Contains(m.View(), "De volta do editor") {
		t.Fatalf(":wq volta ao app:\n%s", m.View())
	}
	if data, _ := os.ReadFile(code); string(data) != "int\tmain(void);" {
		t.Fatalf("o código foi salvo: %q", data)
	}

	// The editor comes back with the windows sized as they were left.
	m = run(t, m, key("e"))
	before := m.editor.Shares()
	for _, msg := range []tea.Msg{key("2"), key("0"), tea.KeyMsg{Type: tea.KeyCtrlW}, key(">")} {
		m = run(t, m, msg)
	}
	resized := m.editor.Shares()
	if fmt.Sprint(resized) == fmt.Sprint(before) {
		t.Fatalf("20 ctrl+w > deveria mudar os tamanhos: %v", resized)
	}
	m = run(t, m, key(":q"))
	m = run(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, key("e"))
	if got := m.editor.Shares(); fmt.Sprint(got) != fmt.Sprint(resized) {
		t.Fatalf("o editor deveria reabrir com os tamanhos de antes: %v, esperado %v", got, resized)
	}
}

func TestEditorCommand(t *testing.T) {
	tests := []struct {
		visual, editor string
		want           []string
	}{
		{"", "", []string{"vim", "-O", "-c", "setlocal nomodifiable readonly", "-c", "wincmd l", "s.txt", "a.c"}},
		{"", "nvim", []string{"nvim", "-O", "-c", "setlocal nomodifiable readonly", "-c", "wincmd l", "s.txt", "a.c"}},
		{"", "/usr/local/bin/vim", []string{"/usr/local/bin/vim", "-O", "-c", "setlocal nomodifiable readonly", "-c", "wincmd l", "s.txt", "a.c"}},
		{"code -w", "vim", []string{"code", "-w", "s.txt", "a.c"}},
		{"", "nano", []string{"nano", "s.txt", "a.c"}},
	}
	for _, tt := range tests {
		t.Setenv("VISUAL", tt.visual)
		t.Setenv("EDITOR", tt.editor)
		got := editorCommand("s.txt", []string{"a.c"}).Args
		if strings.Join(got, " ") != strings.Join(tt.want, " ") {
			t.Errorf("VISUAL=%q EDITOR=%q: %v, esperado %v", tt.visual, tt.editor, got, tt.want)
		}
	}
}

// Runs the real program loop: pressing E must suspend the app, run the
// external editor with the subject and the turn-in files, and come back.
func TestAppEditRunsEditorProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("usa um editor falso em shell script")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	editor := filepath.Join(dir, "fake-editor")
	script := "#!/bin/sh\necho \"$@\" > " + argsFile + "\n"
	if err := os.WriteFile(editor, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", editor)

	fe := &fakeExam{sess: &exam.Session{Mode: exam.ModeExam, Rank: "02", Level: 1, Exercise: "first_word"}}
	// An *os.File input, like the real os.Stdin, is handed to the editor as is.
	in, keys, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close() }()
	p := tea.NewProgram(NewApp(AppOptions{Tabs: []AppTab{{Title: "Início"}}, Exam: fe}, appNow),
		tea.WithInput(in), tea.WithOutput(io.Discard))

	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()

	p.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if data, err := os.ReadFile(argsFile); err == nil && len(data) > 0 {
			if got := strings.TrimSpace(string(data)); got != "subject.pt.txt rendu/first_word.c" {
				t.Fatalf("editor recebeu %q", got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("o editor não foi executado")
		}
		time.Sleep(20 * time.Millisecond)
	}

	p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	_ = keys.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("o app não voltou do editor")
	}
}
