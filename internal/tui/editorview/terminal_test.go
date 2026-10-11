package editorview

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestKeyBytes(t *testing.T) {
	for _, tt := range []struct {
		msg       tea.KeyMsg
		appCursor bool
		want      string
	}{
		{runes("ls"), false, "ls"},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true}, false, "\x1bb"},
		{tea.KeyMsg{Type: tea.KeyEnter}, false, "\r"},
		{tea.KeyMsg{Type: tea.KeySpace}, false, " "},
		{tea.KeyMsg{Type: tea.KeyBackspace}, false, "\x7f"},
		{tea.KeyMsg{Type: tea.KeyTab}, false, "\t"},
		{tea.KeyMsg{Type: tea.KeyEsc}, false, "\x1b"},
		{tea.KeyMsg{Type: tea.KeyCtrlC}, false, "\x03"},
		{tea.KeyMsg{Type: tea.KeyCtrlD}, false, "\x04"},
		{tea.KeyMsg{Type: tea.KeyUp}, false, "\x1b[A"},
		{tea.KeyMsg{Type: tea.KeyUp}, true, "\x1bOA"},
		{tea.KeyMsg{Type: tea.KeyLeft}, false, "\x1b[D"},
		{tea.KeyMsg{Type: tea.KeyHome}, false, "\x1b[H"},
		{tea.KeyMsg{Type: tea.KeyDelete}, false, "\x1b[3~"},
		{tea.KeyMsg{Type: tea.KeyPgDown}, false, "\x1b[6~"},
		{tea.KeyMsg{Type: tea.KeyCtrlLeft}, false, "\x1b[1;5D"},
		{tea.KeyMsg{Type: tea.KeyF1}, false, "\x1bOP"},
	} {
		if got := string(keyBytes(tt.msg, tt.appCursor)); got != tt.want {
			t.Errorf("%v (app %v): %q, esperado %q", tt.msg, tt.appCursor, got, tt.want)
		}
	}
}

// waitTerminal feeds the terminal's output messages to the model until the
// screen shows want (or the shell ends, with until "exit").
func waitTerminal(t *testing.T, m Model, cmd tea.Cmd, want string) (Model, tea.Cmd) {
	t.Helper()
	return waitScreen(t, m, cmd, want, func(screen string) bool { return strings.Contains(screen, want) })
}

// waitScreen is waitTerminal with any condition on the screen.
func waitScreen(t *testing.T, m Model, cmd tea.Cmd, want string, ok func(screen string) bool) (Model, tea.Cmd) {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for !ok(ansi.Strip(m.View())) {
		if cmd == nil {
			t.Fatalf("o terminal parou sem mostrar %q:\n%s", want, ansi.Strip(m.View()))
		}
		got := make(chan tea.Msg, 1)
		go func(c tea.Cmd) { got <- c() }(cmd)
		select {
		case msg := <-got:
			var next tea.Model
			next, cmd = m.Update(msg)
			m = next.(Model)
		case <-deadline:
			t.Fatalf("o terminal não mostrou %q:\n%s", want, ansi.Strip(m.View()))
		}
	}
	return m, cmd
}

// termModel is a Vim editor on a file in a temp folder, 80×20.
func termModel(t *testing.T) (Model, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("sem pseudo-terminal no Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "main.c")
	if err := os.WriteFile(path, []byte("int main(void)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := NewVim(ed).Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	return next.(Model), dir
}

// exCmd types an ex command (":term printf hi") and returns the commands
// Update gave back.
func exCmd(t *testing.T, m Model, cmd string) (Model, tea.Cmd) {
	t.Helper()
	var out tea.Cmd
	for _, msg := range []tea.Msg{runes(":"), runes(cmd), tea.KeyMsg{Type: tea.KeyEnter}} {
		next, c := m.Update(msg)
		m, out = next.(Model), c
	}
	return m, out
}

func TestTerminalRunsACommand(t *testing.T) {
	m, dir := termModel(t)
	defer m.Close()
	m, cmd := exCmd(t, m, "term printf 'ola-%d\\n' 42; pwd")
	if m.ses.term == nil || !m.ses.termFocus {
		t.Fatal(":term deveria abrir o painel com o foco")
	}
	m, cmd = waitTerminal(t, m, cmd, "ola-42")
	screen := ansi.Strip(m.View())
	real, _ := filepath.EvalSymlinks(dir)
	if !strings.Contains(screen, filepath.Base(real)) {
		t.Errorf("o terminal deveria rodar na pasta do arquivo (%s):\n%s", real, screen)
	}
	for _, want := range []string{"int main(void)", "terminal · printf", "TERMINAL"} {
		if !strings.Contains(screen, want) {
			t.Errorf("a tela não mostra %q:\n%s", want, screen)
		}
	}
	m, _ = waitTerminal(t, m, cmd, "processo terminou")
	// Any key closes the panel of a finished command.
	next, _ := m.Update(runes("x"))
	m = next.(Model)
	if m.ses.term != nil || m.ses.termFocus {
		t.Error("uma tecla deveria fechar o terminal que terminou")
	}
	if got := m.ed.Buffer().Text(); got != "int main(void)\n" {
		t.Errorf("a tecla que fecha não deveria ir para o código: %q", got)
	}
}

func TestTerminalFocusAndKeys(t *testing.T) {
	m, _ := termModel(t)
	defer m.Close()
	m, cmd := exCmd(t, m, "term cat")
	// Typed keys go to the program (cat echoes the line back).
	for _, msg := range []tea.Msg{runes("eco"), tea.KeyMsg{Type: tea.KeyEnter}} {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	m, cmd = waitScreen(t, m, cmd, "eco duas vezes", func(screen string) bool { return strings.Count(screen, "eco") >= 2 })
	_ = cmd

	// Ctrl-\ Ctrl-n goes back to the code; keys edit it again.
	for _, msg := range []tea.Msg{tea.KeyMsg{Type: tea.KeyCtrlBackslash}, tea.KeyMsg{Type: tea.KeyCtrlN}} {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	if m.ses.termFocus {
		t.Fatal("Ctrl-\\ Ctrl-n deveria voltar ao código")
	}
	next, _ := m.Update(runes("x"))
	if got := next.(Model).ed.Buffer().Text(); got != "nt main(void)\n" {
		t.Errorf("x no código: %q", got)
	}
	m = next.(Model)

	// Ctrl-w j goes down to the terminal, Ctrl-w k back up.
	for _, msg := range []tea.Msg{tea.KeyMsg{Type: tea.KeyCtrlW}, runes("j")} {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	if !m.ses.termFocus {
		t.Fatal("Ctrl-w j deveria ir para o terminal")
	}
	for _, msg := range []tea.Msg{tea.KeyMsg{Type: tea.KeyCtrlW}, runes("k")} {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	if m.ses.termFocus {
		t.Fatal("Ctrl-w k deveria voltar ao código")
	}

	// A click on the panel focuses it; on the code, the code.
	next, _ = m.Update(tea.MouseMsg{X: 5, Y: 18, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = next.(Model)
	if !m.ses.termFocus {
		t.Fatal("clicar no terminal deveria dar o foco a ele")
	}
	next, _ = m.Update(tea.MouseMsg{X: 10, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = next.(Model)
	if m.ses.termFocus {
		t.Fatal("clicar no código deveria dar o foco a ele")
	}

	// :term with the panel open only gives it the focus back.
	old := m.ses.term
	m, _ = exCmd(t, m, "term")
	if m.ses.term != old || !m.ses.termFocus {
		t.Error(":term com o painel aberto deveria só focar o terminal")
	}

	// Closing the editor ends the program.
	m.Close()
	select {
	case _, ok := <-old.output:
		for ok {
			_, ok = <-old.output
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fechar o editor deveria encerrar o programa do terminal")
	}
}

func TestTerminalLayout(t *testing.T) {
	m, _ := termModel(t)
	defer m.Close()
	before := m.ed.Viewport().Height
	m, cmd := exCmd(t, m, "term sleep 5")
	_ = cmd
	if got := m.ed.Viewport().Height; got >= before {
		t.Errorf("o código deveria encolher para caber o terminal: %d → %d", before, got)
	}
	rows := strings.Split(m.View(), "\n")
	if len(rows) != 20 {
		t.Fatalf("a tela tem %d linhas, esperado 20", len(rows))
	}
	if !strings.Contains(ansi.Strip(rows[20-1-m.terminalHeight()]), "terminal · sleep") {
		t.Errorf("o título do terminal deveria estar logo acima da área dele:\n%s", ansi.Strip(m.View()))
	}
}
