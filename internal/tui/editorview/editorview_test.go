package editorview

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
)

func newTestModel(t *testing.T, ed *editor.Editor) Model {
	t.Helper()
	next, _ := New(ed).Update(tea.WindowSizeMsg{Width: 60, Height: 8})
	return next.(Model)
}

func press(t *testing.T, m Model, msgs ...tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, msg := range msgs {
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m, cmd
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestTypingAndView(t *testing.T) {
	m := newTestModel(t, editor.New(""))
	m, _ = press(t, m, runes("int"), tea.KeyMsg{Type: tea.KeySpace}, runes("main(void)"), tea.KeyMsg{Type: tea.KeyEnter}, runes("{"))
	if got := m.Editor().Buffer().Text(); got != "int main(void)\n{" {
		t.Fatalf("texto: %q", got)
	}
	view := ansi.Strip(m.View())
	for _, want := range []string{"1 │ int main(void)", "2 │ {", "EDIT", "[sem nome] [+]", "2:2"} {
		if !strings.Contains(view, want) {
			t.Errorf("faltou %q na tela:\n%s", want, view)
		}
	}
	if lines := strings.Split(view, "\n"); len(lines) != 8 || !strings.HasPrefix(strings.TrimSpace(lines[2]), "~") {
		t.Errorf("tela deveria ter 8 linhas e ~ depois do fim:\n%s", view)
	}
}

func TestEnterKeepsIndentation(t *testing.T) {
	ed := editor.New("\tif (argc == 2)")
	ed.MoveCursor(ed.Buffer().End())
	m := newTestModel(t, ed)
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEnter}, runes("x"))
	if got := m.Editor().Buffer().Line(1); got != "\tx" {
		t.Fatalf("a nova linha deveria herdar o tab: %q", got)
	}
}

func TestUndoRedoKeys(t *testing.T) {
	m := newTestModel(t, editor.New(""))
	m, _ = press(t, m, runes("abc"), tea.KeyMsg{Type: tea.KeyCtrlZ})
	if m.Editor().Buffer().Text() != "" {
		t.Fatalf("ctrl+z: %q", m.Editor().Buffer().Text())
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlY})
	if m.Editor().Buffer().Text() != "abc" {
		t.Fatalf("ctrl+y: %q", m.Editor().Buffer().Text())
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlY})
	if !strings.Contains(m.View(), "nada para refazer") {
		t.Fatalf("sem redo deveria avisar:\n%s", m.View())
	}
}

func TestSaveAndQuit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "first_word.c")
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	m := newTestModel(t, ed)
	m, _ = press(t, m, runes("x"))

	// Unsaved changes: the first Ctrl-Q only warns.
	m, cmd := press(t, m, tea.KeyMsg{Type: tea.KeyCtrlQ})
	// The 60-column status bar cuts the warning, but never drops it.
	if cmd != nil || m.Done() || !strings.Contains(m.View(), "alterações") {
		t.Fatalf("primeiro ctrl+q deveria só avisar:\n%s", m.View())
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if data, _ := os.ReadFile(path); string(data) != "x" || !strings.Contains(m.View(), "salvo:") {
		t.Fatalf("ctrl+s não salvou: %q\n%s", data, m.View())
	}
	m, cmd = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlQ})
	if !m.Done() || cmd == nil {
		t.Fatal("salvo, ctrl+q deveria sair direto")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+q deveria encerrar o programa")
	}
}

func TestMouseClickPlacesCursor(t *testing.T) {
	m := newTestModel(t, editor.New("\tx = 1;\nab"))
	// Gutter is "1 │ " (4 cells); the tab spans columns 0–3, so x is at
	// screen column 4 → cell 8.
	m, _ = press(t, m, tea.MouseMsg{X: 8, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := m.Editor().Cursor(); got != (editor.Position{Line: 0, Column: 1}) {
		t.Fatalf("clique no x: cursor em %v", got)
	}
	m, _ = press(t, m, tea.MouseMsg{X: 40, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := m.Editor().Cursor(); got != (editor.Position{Line: 1, Column: 2}) {
		t.Fatalf("clique depois do fim da linha: cursor em %v", got)
	}
}

func TestRenderLine(t *testing.T) {
	// Tabs expand to the next stop of 4; the cursor past the end draws a cell.
	if got := ansi.Strip(renderLine("\tx", 4, 0, 20, -1)); got != "    x" {
		t.Fatalf("tab: %q", got)
	}
	if got := ansi.Strip(renderLine("ab\tc", 4, 0, 20, -1)); got != "ab  c" {
		t.Fatalf("tab no meio: %q", got)
	}
	if got := ansi.Strip(renderLine("abc", 4, 0, 20, 5)); got != "abc   " {
		t.Fatalf("cursor depois do fim: %q", got)
	}
	if got := ansi.Strip(renderLine("abcdef", 4, 2, 3, -1)); got != "cde" {
		t.Fatalf("rolagem horizontal: %q", got)
	}
}

// The whole loop with real keyboard bytes: type, Ctrl-S, Ctrl-Q.
func TestProgramTypesSavesAndQuits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.c")
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	in, keys, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close() }()
	p := tea.NewProgram(New(ed), tea.WithInput(in), tea.WithOutput(io.Discard))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()

	p.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	if _, err := keys.Write([]byte("int x;\r\x13\x11")); err != nil { // Enter, Ctrl-S, Ctrl-Q
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("o editor não saiu com ctrl+q")
	}
	if data, _ := os.ReadFile(path); string(data) != "int x;\n" {
		t.Fatalf("arquivo salvo: %q", data)
	}
}

func TestVimModeStatusAndCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.c")
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := NewVim(ed).Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	m := next.(Model)
	status := func() string { lines := strings.Split(ansi.Strip(m.View()), "\n"); return lines[len(lines)-1] }

	if !strings.HasPrefix(strings.TrimSpace(status()), "NORMAL") {
		t.Fatalf("deveria começar no NORMAL: %q", status())
	}
	m, _ = press(t, m, runes("i"))
	if !strings.Contains(status(), "INSERT") {
		t.Fatalf("i deveria entrar no INSERT: %q", status())
	}
	m, _ = press(t, m, runes("int x;"), tea.KeyMsg{Type: tea.KeyEsc}, runes("d"))
	if !strings.Contains(status(), "NORMAL") || !strings.HasSuffix(strings.TrimSpace(status()), "d") {
		t.Fatalf("esc volta ao NORMAL e o d pendente aparece: %q", status())
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEsc}, runes(":wq"))
	if got := strings.TrimSpace(status()); got != ":wq" {
		t.Fatalf("a linha de comando deveria tomar a barra: %q", got)
	}
	m, cmd := press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() || cmd == nil {
		t.Fatal(":wq deveria sair")
	}
	if data, _ := os.ReadFile(path); string(data) != "int x;" {
		t.Fatalf(":wq deveria salvar: %q", data)
	}
}

func TestVimClickRespectsNormalMode(t *testing.T) {
	next, _ := NewVim(editor.New("ab")).Update(tea.WindowSizeMsg{Width: 60, Height: 8})
	m := next.(Model)
	m, _ = press(t, m, tea.MouseMsg{X: 40, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := m.Editor().Cursor(); got != (editor.Position{Line: 0, Column: 1}) {
		t.Fatalf("no NORMAL o clique para no último caractere: %v", got)
	}
}

// Real keyboard bytes through a running program, Vim style.
func TestVimProgramWithRealKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "first_word.c")
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	in, keys, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close() }()
	p := tea.NewProgram(NewVim(ed), tea.WithInput(in), tea.WithOutput(io.Discard))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	p.Send(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Separate writes: an Esc glued to the next key would read as Alt+key.
	for _, chunk := range []string{"iint main(void)\r{\r}", "\x1b", "ggOx", "\x1b", "dd:wq\r"} {
		if _, err := keys.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
		time.Sleep(60 * time.Millisecond)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("o editor não saiu com :wq")
	}
	// gg isn't implemented yet (phase 3): "g g" are no-ops, so O opens above
	// the last line and dd removes that new line again.
	if data, _ := os.ReadFile(path); string(data) != "int main(void)\n{\n}" {
		t.Fatalf("arquivo salvo: %q", data)
	}
}
