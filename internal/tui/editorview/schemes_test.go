package editorview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/syntax"
)

func TestColorschemes(t *testing.T) {
	colors(t)
	path := filepath.Join(t.TempDir(), "main.c")
	if err := os.WriteFile(path, []byte("int\tmain(void)\n{\n\treturn (0);\n}"), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved []string
	m := NewVim(ed).OnColorscheme(func(name string) { saved = append(saved, name) })
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 8})
	m = next.(Model)
	ex := func(cmd string) {
		t.Helper()
		m, _ = press(t, m, runes(":"+cmd), tea.KeyMsg{Type: tea.KeyEnter})
	}
	keyword := func(name string) string { return findScheme(name).styles[syntax.Keyword].Render("return") }

	if m.Colorscheme() != "lightyear" || !strings.Contains(m.View(), keyword("lightyear")) {
		t.Fatalf("o tema padrão é o lightyear: %s", m.Colorscheme())
	}
	if ex("colorscheme"); !strings.Contains(m.message, "[lightyear]") || !strings.Contains(m.message, "dracula") || !strings.Contains(m.message, "off") {
		t.Fatalf(":colorscheme lista os temas: %q", m.message)
	}
	ex("colorscheme dracula")
	if m.Colorscheme() != "dracula" || !strings.Contains(m.View(), keyword("dracula")) || strings.Join(saved, ",") != "dracula" {
		t.Fatalf(":colorscheme dracula: %s salvo %v", m.Colorscheme(), saved)
	}
	if ex("colo nada"); !m.isError || !strings.Contains(m.message, "E185") || m.Colorscheme() != "dracula" {
		t.Fatalf("tema que não existe: %q", m.message)
	}
	ex("colo off")
	if strings.Contains(m.View(), keyword("dracula")) || strings.Contains(m.View(), keyword("lightyear")) || !strings.Contains(m.View(), "return (0);") {
		t.Fatal("off tira as cores do código")
	}
	if got := m.WithColorscheme("monokai").Colorscheme(); got != "monokai" {
		t.Fatalf("WithColorscheme: %s", got)
	}
	if got := m.WithColorscheme("").Colorscheme(); got != "monokai" {
		t.Fatalf("nome vazio mantém o tema: %s", got)
	}
	for _, name := range ColorSchemes() {
		if findScheme(name) == nil {
			t.Fatalf("tema %s não encontrado", name)
		}
	}
}
