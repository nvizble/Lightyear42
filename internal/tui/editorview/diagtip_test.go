package editorview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
)

func TestDiagnosticUnderTheMouse(t *testing.T) {
	next, _ := NewVim(editor.New("int x\nok")).Update(tea.WindowSizeMsg{Width: 70, Height: 8})
	m := setLSP(next.(Model), &lspState{diags: []lsp.Diagnostic{
		{Start: lsp.Pos{Line: 0, Col: 5}, End: lsp.Pos{Line: 0, Col: 5}, Severity: lsp.Error, Message: "expected ';' after top level declarator\nnota: insira ';'"},
		{Start: lsp.Pos{Line: 0, Col: 4}, End: lsp.Pos{Line: 0, Col: 5}, Severity: lsp.Warning, Message: "unused variable 'x'"},
	}})
	hover := func(x, y int) string {
		t.Helper()
		m, _ = press(t, m, tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
		rows := strings.Split(ansi.Strip(m.View()), "\n")
		return strings.Join(rows[:len(rows)-1], "\n") // not the status line
	}
	// The gutter is "1 │ " (4 cells): "x" is at 8, the missing ";" at 9.
	if s := hover(8, 0); !strings.Contains(s, "aviso: unused variable 'x'") || strings.Contains(s, "expected") {
		t.Fatalf("mouse no x:\n%s", s)
	}
	if s := hover(9, 0); !strings.Contains(s, "erro: expected ';' after top level declarator") || !strings.Contains(s, "nota: insira ';'") {
		t.Fatalf("mouse depois do fim (o ';' que falta):\n%s", s)
	}
	if s := hover(12, 0); strings.Contains(s, "expected") || strings.Contains(s, "unused") {
		t.Fatalf("longe do erro não mostra nada:\n%s", s)
	}
	// On the gutter: everything on the line, the error first.
	s := hover(1, 0)
	if e, w := strings.Index(s, "erro:"), strings.Index(s, "aviso:"); e < 0 || w < 0 || e > w {
		t.Fatalf("na margem, todos da linha (erro antes):\n%s", s)
	}
	if s := hover(8, 1); strings.Contains(s, "erro:") {
		t.Fatalf("a linha 2 não tem erro:\n%s", s)
	}
	hover(8, 0)
	m, _ = press(t, m, runes("j"))
	if strings.Contains(ansi.Strip(m.View()), "aviso: unused") {
		t.Fatal("uma tecla fecha a caixa")
	}
}

func TestHoverKeyShowsTheLineDiagnostics(t *testing.T) {
	m := withFakeLSP(t, "int\tx\nok")
	m.lsp.diags = []lsp.Diagnostic{{Start: lsp.Pos{Line: 0, Col: 5}, End: lsp.Pos{Line: 0, Col: 5}, Severity: lsp.Error, Message: "expected ';'"}}
	m = m.lspReply(hoverMsg{text: "variable x\nType: int"})
	s := screen(m)
	if e, h := strings.Index(s, "erro: expected ';'"), strings.Index(s, "variable x"); e < 0 || h < 0 || e > h {
		t.Fatalf("K mostra o erro da linha e depois o hover:\n%s", s)
	}
	// Nothing from the server, but an error on the line: still shown.
	m, _ = press(t, m, runes("j"))
	m, _ = press(t, m, runes("k"))
	m = m.lspReply(hoverMsg{})
	if !strings.Contains(screen(m), "erro: expected ';'") {
		t.Fatalf("K sem hover do servidor ainda mostra o erro:\n%s", screen(m))
	}
	m, _ = press(t, m, runes("j"))
	if m = m.lspReply(hoverMsg{}); m.message != "nada para mostrar aqui" {
		t.Fatalf("linha sem nada: %q", m.message)
	}
}
