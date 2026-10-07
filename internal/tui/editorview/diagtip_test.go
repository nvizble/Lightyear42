package editorview

import (
	"os"
	"os/exec"
	"path/filepath"
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
	if s := hover(8, 0); strings.Contains(s, "gra aplica") {
		t.Fatalf("sem correção, sem dica:\n%s", s)
	}
	m.lsp.diags[0].Message = "Expected ';' after expression (fix available)"
	if s := hover(9, 0); !strings.Contains(s, "▸ clique aqui para corrigir") {
		t.Fatalf("com correção, o botão de corrigir:\n%s", s)
	}
	// The mouse can go onto the box (it stays); a click there asks the
	// server for the fix, from the error's line (no server here: it says so).
	top, left := m.placement(7, m.ses.tip.lines, 0, 9)
	if s := hover(left+3, top+1); !strings.Contains(s, "clique aqui para corrigir") {
		t.Fatalf("a caixa fica com o mouse em cima dela:\n%s", s)
	}
	m, _ = press(t, m, runes("j"))
	hover(9, 0)
	m, _ = press(t, m, tea.MouseMsg{X: left + 3, Y: top + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if m.ses.tip != nil || m.Editor().Cursor().Line != 0 || !strings.Contains(m.message, "sem language server") {
		t.Fatalf("o clique pede a correção a partir da linha do erro: cursor %v, %q", m.Editor().Cursor(), m.message)
	}
	if s := hover(12, 0); strings.Contains(s, "expected") || strings.Contains(s, "unused") {
		t.Fatalf("longe do erro não mostra nada:\n%s", s)
	}
	// On the gutter: everything on the line, the error first.
	s := hover(1, 0)
	if e, w := strings.Index(s, "erro:"), strings.Index(s, "aviso:"); e < 0 || w < 0 || e > w {
		t.Fatalf("na margem, todos da linha (erro antes):\n%s", s)
	}
	hover(60, 5) // off the box, which stays while the mouse is on it
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

func TestFixFromAClick(t *testing.T) {
	m := withFakeLSP(t, "int\tx\nint\ty")
	at := func(l, c int) lsp.Pos { return lsp.Pos{Line: l, Col: c} }
	fix := lsp.Action{Title: "insert ';'", Edits: []lsp.FileEdit{{Path: m.Editor().Path(), Edits: []lsp.TextEdit{{Start: at(0, 5), End: at(0, 5), Text: ";"}}}}}
	// One fix, from a click: applied right away.
	m, _ = m.editReply(actionsMsg{actions: []lsp.Action{fix}, apply: true})
	if m.ses.pick != nil || !strings.HasPrefix(m.Editor().Buffer().Text(), "int\tx;") {
		t.Fatalf("uma correção do clique é aplicada direto: %q", m.Editor().Buffer().Text())
	}
	// Several: the list, to choose.
	other := lsp.Action{Title: "outra"}
	m, _ = m.editReply(actionsMsg{actions: []lsp.Action{fix, other}, apply: true})
	if m.ses.pick == nil || len(m.ses.pick.items) != 2 {
		t.Fatal("várias correções abrem a lista")
	}
}

// The real clangd: hover the missing ";" error, click the box, and the ";"
// is in.
func TestClickFixWithClangd(t *testing.T) {
	if _, err := exec.LookPath("clangd"); err != nil {
		t.Skip("clangd não instalado")
	}
	path := filepath.Join(t.TempDir(), "main.c")
	text := "#include <stdio.h>\n\nint\tmain(void)\n{\n\tprintf(\"hello world\")\n}\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, m: NewVim(ed).WithLSP(), msgs: make(chan tea.Msg, 16)}
	defer h.m.Close()
	h.exec(h.m.Init())
	h.send(tea.WindowSizeMsg{Width: 80, Height: 16})
	h.until("o erro do ';'", func() bool { return h.m.lsp != nil && len(h.m.lsp.diags) > 0 })
	d := h.m.lsp.diags[0]
	if !strings.Contains(d.Message, "fix available") {
		t.Skipf("este clangd não oferece a correção: %q", d.Message)
	}
	// The error is on the "}" line; the gutter is "1 │ " (4 cells).
	x, y := 4+d.Start.Col, d.Start.Line
	h.send(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
	if h.m.ses.tip == nil || !h.m.ses.tip.fix {
		t.Fatalf("o mouse no erro deveria mostrar a caixa com a correção:\n%s", screen(h.m))
	}
	top, left := h.m.placement(15, h.m.ses.tip.lines, y, x)
	h.send(tea.MouseMsg{X: left + 2, Y: top + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	h.until("o ';' entrar", func() bool { return strings.Contains(h.m.Editor().Buffer().Line(4), `printf("hello world");`) })
}
