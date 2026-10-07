package editorview

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/syntax"
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
	none := lineLook{cursor: -1, selFrom: -1, selTo: -1}
	// Tabs expand to the next stop of 4; the cursor past the end draws a cell.
	if got := ansi.Strip(renderLine("\tx", 4, 0, 20, none)); got != "    x" {
		t.Fatalf("tab: %q", got)
	}
	if got := ansi.Strip(renderLine("ab\tc", 4, 0, 20, none)); got != "ab  c" {
		t.Fatalf("tab no meio: %q", got)
	}
	if got := ansi.Strip(renderLine("abc", 4, 0, 20, lineLook{cursor: 5, selFrom: -1, selTo: -1})); got != "abc   " {
		t.Fatalf("cursor depois do fim: %q", got)
	}
	if got := ansi.Strip(renderLine("abcdef", 4, 2, 3, none)); got != "cde" {
		t.Fatalf("rolagem horizontal: %q", got)
	}

	colors(t)
	sel, cur := styleSelection.Render, styleCursor.Render
	tests := []struct {
		name                                string
		line                                string
		left, width, cursor, selFrom, selTo int
		want                                string
	}{
		{"seleção com o cursor na ponta", "abcd", 0, 20, 2, 0, 3, sel("ab") + cur("c") + "d"},
		{"tab selecionado inteiro", "a\tb", 0, 20, 0, 0, 5, cur("a") + sel("   b")},
		{"linha vazia selecionada", "", 0, 20, -1, 0, 1, sel(" ")},
		{"seleção cortada pela rolagem", "abcdef", 1, 3, -1, 0, 3, sel("bc") + "d"},
	}
	for _, tt := range tests {
		if got := renderLine(tt.line, 4, tt.left, tt.width, lineLook{cursor: tt.cursor, selFrom: tt.selFrom, selTo: tt.selTo}); got != tt.want {
			t.Errorf("%s: %q, esperado %q", tt.name, got, tt.want)
		}
	}
}

// colors makes lipgloss emit styles (tests have no terminal), for checks
// that look at highlighting.
func colors(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
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

func TestVimVisualSelection(t *testing.T) {
	colors(t)
	next, _ := NewVim(editor.New("abc\n\tdef")).Update(tea.WindowSizeMsg{Width: 60, Height: 8})
	m := next.(Model)
	view := func() []string { return strings.Split(m.View(), "\n") }
	status := func() string { v := view(); return ansi.Strip(v[len(v)-1]) }

	m, _ = press(t, m, runes("vj"))
	if !strings.Contains(status(), "VISUAL") || !strings.Contains(status(), "d apaga") {
		t.Fatalf("v deveria mostrar VISUAL e as dicas: %q", status())
	}
	// Line 0 is selected from "a"; line 1 up to the cursor, on the tab.
	if v := view(); !strings.HasSuffix(v[0], styleSelection.Render("abc")) ||
		!strings.HasSuffix(v[1], styleCursor.Render(" ")+styleSelection.Render("   ")+"def") {
		t.Fatalf("seleção na tela:\n%q\n%q", v[0], v[1])
	}
	m, _ = press(t, m, runes("V"))
	if !strings.Contains(status(), "V-LINE") || !strings.HasSuffix(view()[1], styleSelection.Render("   def")) {
		t.Fatalf("V deveria selecionar as linhas inteiras: %q\n%q", status(), view()[1])
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if !strings.Contains(status(), "NORMAL") || strings.Contains(m.View(), styleSelection.Render("abc")) {
		t.Fatalf("esc deveria limpar a seleção: %q", status())
	}

	// Drag from "b" to "e": press, then motion with the button held. The
	// gutter is "1 │ " (4 cells); "e" is after the tab, at screen column 5.
	left := tea.MouseButtonLeft
	m, _ = press(t, m,
		tea.MouseMsg{X: 5, Y: 0, Action: tea.MouseActionPress, Button: left},
		tea.MouseMsg{X: 9, Y: 1, Action: tea.MouseActionMotion, Button: left},
		tea.MouseMsg{X: 9, Y: 1, Action: tea.MouseActionRelease, Button: left})
	if r, _, ok := m.Editor().SelectedRange(); !ok || m.Editor().Buffer().Slice(r) != "bc\n\tde" || !strings.Contains(status(), "VISUAL") {
		t.Fatalf("arrastar deveria selecionar: %q %q", m.Editor().Buffer().Slice(r), status())
	}
	m, _ = press(t, m, runes("d"))
	if got := m.Editor().Buffer().Text(); got != "af" {
		t.Fatalf("d apaga o que o mouse selecionou: %q", got)
	}
	m, _ = press(t, m, runes("vl"), tea.MouseMsg{X: 4, Y: 0, Action: tea.MouseActionPress, Button: left})
	if _, _, ok := m.Editor().SelectedRange(); ok || !strings.Contains(status(), "NORMAL") {
		t.Fatalf("um clique deveria encerrar a seleção: %q", status())
	}
}

func TestSyntaxColors(t *testing.T) {
	colors(t)
	path := filepath.Join(t.TempDir(), "main.c")
	if err := os.WriteFile(path, []byte("int\tmain(void)\n{\n\treturn (0); // ok\n}"), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	m := NewVim(ed)
	defer m.Close()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 8})
	m = next.(Model)
	style := func(c syntax.Class, s string) string { return schemes[0].styles[c].Render(s) }
	view := m.View()
	for _, want := range []string{style(syntax.Function, "main"), style(syntax.Keyword, "return"), style(syntax.Number, "0"), style(syntax.Comment, "// ok")} {
		if !strings.Contains(view, want) {
			t.Errorf("faltou %q na tela:\n%s", want, view)
		}
	}
	// Typing keeps the tree in step: the new line is colored right away.
	m, _ = press(t, m, runes("Goint x;"), tea.KeyMsg{Type: tea.KeyEsc})
	for _, row := range strings.Split(m.View(), "\n") {
		if strings.Contains(ansi.Strip(row), "5 │ int x;") {
			if !strings.Contains(row, style(syntax.Type, "int")) {
				t.Fatalf("a linha digitada deveria ganhar cor: %q", row)
			}
			return
		}
	}
	t.Fatalf("a linha digitada não apareceu:\n%s", ansi.Strip(m.View()))
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
	// gg goes to the first line, O opens a line above it and dd removes that
	// new line again: the file ends up as typed.
	if data, _ := os.ReadFile(path); string(data) != "int main(void)\n{\n}" {
		t.Fatalf("arquivo salvo: %q", data)
	}
}

func TestSearchPromptAndHighlight(t *testing.T) {
	colors(t)
	next, _ := NewVim(editor.New("foo bar\nbar")).Update(tea.WindowSizeMsg{Width: 60, Height: 6})
	m := next.(Model)
	m, _ = press(t, m, runes("/bar"))
	lines := strings.Split(m.View(), "\n")
	if got := ansi.Strip(lines[len(lines)-1]); strings.TrimSpace(got) != "/bar" {
		t.Fatalf("o prompt da busca toma a barra: %q", got)
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	rows := strings.Split(m.View(), "\n")
	// The cursor sits on the first match; the other one is highlighted.
	if !strings.Contains(rows[1], styleSearchHit.Render("bar")) || !strings.Contains(rows[0], styleSearchHit.Render("ar")) {
		t.Fatalf("destaques:\n%q\n%q", rows[0], rows[1])
	}
	m, _ = press(t, m, runes(":noh"), tea.KeyMsg{Type: tea.KeyEnter})
	if strings.Contains(m.View(), styleSearchHit.Render("bar")) {
		t.Fatal(":noh apaga os destaques")
	}
}

func TestRecordingShown(t *testing.T) {
	next, _ := NewVim(editor.New("a")).Update(tea.WindowSizeMsg{Width: 60, Height: 4})
	m := next.(Model)
	m, _ = press(t, m, runes("qa"))
	if !strings.Contains(ansi.Strip(m.statusLine()), "NORMAL  gravando @a") {
		t.Fatalf("statusline gravando: %q", ansi.Strip(m.statusLine()))
	}
	m, _ = press(t, m, runes("q"))
	if strings.Contains(ansi.Strip(m.statusLine()), "gravando") {
		t.Fatal("q para a gravação")
	}
}
