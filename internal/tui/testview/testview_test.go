package testview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/tester"
	"github.com/nvizble/Lightyear42/internal/tui/editorview"
)

// project writes a small libft-like folder and returns its root.
func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"Makefile":         "NAME = libft.a\n",
		"libft.h":          "char **ft_split(char const *s, char c);\n",
		"ft_split.c":       "#include \"libft.h\"\n\nstatic int\tcount(void);\n\nchar\t**ft_split(char const *s, char c)\n{\n\treturn (0);\n}\n",
		"utils/helpers.c":  "int\tft_helper(int x)\n{\n\treturn (x);\n}\n",
		"ft_split.o":       "",
		".git/HEAD":        "ref",
		"README.md":        "# libft\n",
		"utils/ft_more.c":  "",
		"utils/.hidden.c":  "",
		"utils/deep/x.txt": "",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func newModel(t *testing.T, root string) Model {
	t.Helper()
	ed, err := editor.Open(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	project, err := tester.Lookup("libft")
	if err != nil {
		t.Fatal(err)
	}
	m := New(editorview.NewVim(ed), project, root, tester.Options{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return next.(Model)
}

func send(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func keys(s string) []tea.Msg {
	var msgs []tea.Msg
	for _, r := range s {
		if r == ' ' {
			msgs = append(msgs, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return msgs
}

func click(x, y int) tea.Msg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

func screen(m Model) string { return ansi.Strip(m.View()) }

func TestScreenHasBarTreeAndEditor(t *testing.T) {
	m := newModel(t, project(t))
	s := screen(m)
	for _, want := range []string{"lightyear test · libft", "▶ Rodar testes", "Space e arquivos", "▸ utils", "ft_split.c", "README.md", "NAME = libft.a"} {
		if !strings.Contains(s, want) {
			t.Errorf("a tela não mostra %q:\n%s", want, s)
		}
	}
	for _, hidden := range []string{"ft_split.o", ".git"} {
		if strings.Contains(s, hidden) {
			t.Errorf("a árvore mostra %q", hidden)
		}
	}
	if lines := strings.Split(m.View(), "\n"); len(lines) != 30 {
		t.Errorf("a tela tem %d linhas, esperado 30", len(lines))
	}
}

func TestTreeKeysOpenFilesAndFolders(t *testing.T) {
	root := project(t)
	m := newModel(t, root)
	if m.focus != focusTree {
		t.Fatal("a árvore deveria começar com o foco")
	}
	// Rows: utils/, ft_split.c, libft.h, Makefile, README.md.
	m = send(t, m, keys("gl")...)
	if !m.tree.expanded[filepath.Join(root, "utils")] {
		t.Fatal("l numa pasta deveria abri-la")
	}
	if !strings.Contains(screen(m), "helpers.c") {
		t.Errorf("a pasta aberta não mostra os arquivos:\n%s", screen(m))
	}
	m = send(t, m, keys("h")...)
	if m.tree.expanded[filepath.Join(root, "utils")] {
		t.Error("h deveria fechar a pasta")
	}
	m = send(t, m, keys("j")...)
	m = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.ed.Editor().Path(); got != filepath.Join(root, "ft_split.c") {
		t.Errorf("enter abriu %s", got)
	}
	if m.focus != focusEditor {
		t.Error("abrir um arquivo deveria passar o foco para o editor")
	}
}

func TestSpaceEToggleTree(t *testing.T) {
	m := newModel(t, project(t))
	m = send(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.focus != focusEditor {
		t.Fatal("esc na árvore deveria ir para o editor")
	}
	m = send(t, m, keys(" e")...) // editor → tree focus
	if m.focus != focusTree || !m.treeOpen {
		t.Fatalf("Space e no editor deveria ir para a árvore (focus %d)", m.focus)
	}
	m = send(t, m, keys(" e")...) // tree → closed
	if m.treeOpen || m.focus != focusEditor {
		t.Fatal("Space e na árvore deveria fechá-la")
	}
	if strings.Contains(screen(m), "README.md") {
		t.Errorf("a árvore fechada ainda aparece:\n%s", screen(m))
	}
	m = send(t, m, keys(" e")...) // closed → open
	if !m.treeOpen || m.focus != focusTree {
		t.Fatal("Space e com a árvore fechada deveria abri-la")
	}
	// Space in Insert mode is just a space.
	m = send(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = send(t, m, keys("i e")...)
	if !m.treeOpen || m.focus != focusEditor || !strings.Contains(m.ed.Editor().Buffer().Line(0), " e") {
		t.Errorf("Space no INSERT deveria escrever um espaço: %q", m.ed.Editor().Buffer().Line(0))
	}
}

func TestClickTreeOpensFile(t *testing.T) {
	root := project(t)
	m := newModel(t, root)
	// Row 2 of the tree (screen line 3) is libft.h.
	m = send(t, m, click(3, 3))
	if got := m.ed.Editor().Path(); got != filepath.Join(root, "libft.h") {
		t.Errorf("o clique abriu %s", got)
	}
	// A click on the editor focuses it.
	m = send(t, m, keys(" e")...)
	m = send(t, m, click(60, 5))
	if m.focus != focusEditor {
		t.Error("clicar no editor deveria passar o foco para ele")
	}
}

func TestButtonStartsRun(t *testing.T) {
	m := newModel(t, project(t))
	next, cmd := m.Update(click(m.buttonX()+2, 0))
	m = next.(Model)
	if !m.running || cmd == nil {
		t.Fatal("o clique no botão deveria rodar os testes")
	}
	if !strings.Contains(screen(m), "preparando") {
		t.Errorf("o botão deveria mostrar o progresso:\n%s", screen(m))
	}
	// A second click while running does nothing.
	if _, cmd := m.Update(click(m.buttonX()+2, 0)); cmd != nil {
		t.Error("um segundo clique não deveria rodar de novo")
	}
	m = send(t, m, stepMsg("make"))
	if !strings.Contains(screen(m), "make…") {
		t.Errorf("o progresso não mostra o passo:\n%s", screen(m))
	}
}

func report() tester.Report {
	return tester.Report{Project: "libft", Cases: []tester.Case{
		{Group: "Makefile", Name: "regra re", Status: tester.OK},
		{Group: "ft_split", Name: "ft_split(\"a b\", ' ')", Status: tester.KO, Detail: "palavra 1: esperado \"b\""},
		{Group: "ft_split", Name: "1000 palavras", Status: tester.OK},
		{Group: "norminette", Name: "ft_split.c", Status: tester.KO, Detail: "Error: TOO_MANY_LINES (line: 7, col: 1): Function has more than 25 lines"},
	}}
}

func TestResultsPanel(t *testing.T) {
	root := project(t)
	m := newModel(t, root)
	m = send(t, m, doneMsg{report: report()})
	s := screen(m)
	for _, want := range []string{"✓ 2", "✗ 2", "Resultados", "2 passaram · 2 falharam", "▸ ✓ Makefile", "▾ ✗ ft_split", "palavra 1: esperado", "TOO_MANY_LINES"} {
		if !strings.Contains(s, want) {
			t.Errorf("a tela não mostra %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "regra re") {
		t.Error("um grupo que passou deveria vir fechado")
	}

	// Clicking the failing case opens ft_split.c on the function.
	bodyEnd := 1 + m.bodyHeight()
	row := -1
	for i, l := range m.panel.lines {
		if l.c != nil && l.c.Name == "ft_split(\"a b\", ' ')" {
			row = i
			break
		}
	}
	m = send(t, m, click(10, bodyEnd+1+row))
	if got, line := m.ed.Editor().Path(), m.ed.Editor().Cursor().Line; got != filepath.Join(root, "ft_split.c") || line != 4 {
		t.Errorf("abriu %s na linha %d, esperado ft_split.c na linha 5", got, line+1)
	}

	// Space t goes to the results; enter on a group opens it; Space t closes.
	m = send(t, m, keys(" t")...)
	if m.focus != focusPanel {
		t.Fatal("Space t deveria ir para os resultados")
	}
	m.panel.sel = 0
	m = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(screen(m), "regra re") {
		t.Errorf("enter no grupo deveria abri-lo:\n%s", screen(m))
	}
	m = send(t, m, keys(" t")...)
	if m.panelOpen {
		t.Error("Space t nos resultados deveria fechá-los")
	}
}

func TestLocate(t *testing.T) {
	root := project(t)
	tests := []struct {
		c    tester.Case
		file string
		line int
	}{
		{tester.Case{Group: "norminette", Name: "ft_split.c", Detail: "Error: X (line: 12, col: 4): y"}, "ft_split.c", 12},
		{tester.Case{Group: "Makefile", Name: "regra re"}, "Makefile", 0},
		{tester.Case{Group: "README.md", Name: "seção Resources"}, "README.md", 0},
		{tester.Case{Group: "ft_split", Name: "ft_split(\"\")"}, "ft_split.c", 5},
		{tester.Case{Group: "ft_helper", Name: "x"}, "utils/helpers.c", 1},
		{tester.Case{Group: "ft_split", Name: "compila", Detail: "ft_split.c:8:3: error: x"}, "ft_split.c", 8},
		{tester.Case{Group: "funções permitidas", Name: "x"}, "", 0},
		{tester.Case{Group: "norminette", Name: "ft_split.c", Detail: "Error: A (line: 1, col: 1): a\nError: B (line: 7, col: 2): b"}, "ft_split.c", 1},
		{tester.Case{Group: "ft_nothing", Name: "x"}, "", 0},
	}
	if path, line := locate(root, tests[0].c, "Error: B (line: 7, col: 2): b"); line != 7 || filepath.Base(path) != "ft_split.c" {
		t.Errorf("a linha clicada deveria valer: %s:%d", path, line)
	}
	for _, tt := range tests {
		path, line := locate(root, tt.c, "")
		want := ""
		if tt.file != "" {
			want = filepath.Join(root, tt.file)
		}
		if path != want || line != tt.line {
			t.Errorf("locate(%s / %s) = %s:%d, esperado %s:%d", tt.c.Group, tt.c.Name, path, line, want, tt.line)
		}
	}
}
