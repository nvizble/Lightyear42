package editorview

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
)

// withFakeLSP is a Vim editor with the language server side but no server:
// tests hand it the server's answers.
func withFakeLSP(t *testing.T, text string) Model {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.c")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	m := NewVim(ed).WithLSP()
	m.lsp.srv.starting = false
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	return next.(Model)
}

func screen(m Model) string { return ansi.Strip(m.View()) }

func TestCompletionList(t *testing.T) {
	m := withFakeLSP(t, "int\tcounter;\nint\tmain(void) { return (cou")
	m, _ = press(t, m, runes("jA")) // Insert after "cou"
	items := []lsp.Item{{Label: "counter", Detail: "int", Text: "counter"}, {Label: "count(int n)", Detail: "int", Text: "count"}, {Label: "cos", Text: "cos"}}
	m = m.lspReply(completionMsg{seq: m.lsp.seq, items: items})
	if s := screen(m); !strings.Contains(s, "counter       int") || !strings.Contains(s, "count(int n)") || strings.Contains(s, " cos ") {
		t.Fatalf("a lista mostra só o que começa com \"cou\":\n%s", s)
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlN})
	if m.lsp.comp.selected != 1 {
		t.Fatalf("ctrl+n desce: %d", m.lsp.comp.selected)
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyUp}, runes("n"))
	if got := len(m.lsp.comp.shown); got != 2 || m.lsp.comp.selected != 0 {
		t.Fatalf("\"coun\" ainda mostra 2: %d (sel %d)", got, m.lsp.comp.selected)
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if line := m.Editor().Buffer().Line(1); line != "int\tmain(void) { return (counter" || m.lsp.comp != nil {
		t.Fatalf("tab aceita: %q", line)
	}
	if strings.Contains(screen(m), "count(int n)") {
		t.Fatal("a lista deveria fechar")
	}
	// Esc closes the list and leaves Insert; one u undoes the whole session.
	m = m.lspReply(completionMsg{seq: m.lsp.seq, items: items})
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.lsp.comp != nil || m.vim.Mode().String() != "NORMAL" {
		t.Fatalf("esc: lista %v modo %v", m.lsp.comp, m.vim.Mode())
	}
	m, _ = press(t, m, runes("u"))
	if line := m.Editor().Buffer().Line(1); line != "int\tmain(void) { return (cou" {
		t.Fatalf("u deveria desfazer o insert com a completion: %q", line)
	}
	// "." repeats the insert with the completion: the typed "cou" and the
	// "nter" that completed it.
	m = withFakeLSP(t, "int\tcounter;\n\nx")
	m, _ = press(t, m, runes("jA"), runes("c"), runes("o"), runes("u"))
	m = m.lspReply(completionMsg{seq: m.lsp.seq, items: items})
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyEsc}, runes("j."))
	if got := m.Editor().Buffer().Text(); got != "int\tcounter;\ncounter\nxcounter" {
		t.Fatalf(". depois da completion: %q", got)
	}
	// An answer to an older request is dropped.
	m, _ = press(t, m, runes("A"))
	m = m.lspReply(completionMsg{seq: m.lsp.seq - 1, items: items})
	if m.lsp.comp != nil {
		t.Fatal("resposta velha não abre a lista")
	}
}

func TestHoverDefinitionAndDiagnosticJumps(t *testing.T) {
	m := withFakeLSP(t, "int\tadd(int a, int b);\n\nint\tmain(void)\n{\n\treturn (add(1, 2));\n}")
	m = m.lspReply(hoverMsg{text: "function add\n→ int\nint add(int a, int b)"})
	if s := screen(m); !strings.Contains(s, "│ int add(int a, int b) │") || !strings.Contains(s, "╭") {
		t.Fatalf("hover numa caixa:\n%s", s)
	}
	m, _ = press(t, m, runes("j"))
	if strings.Contains(screen(m), "int add(int a, int b) │") {
		t.Fatal("qualquer tecla fecha o hover")
	}

	abs, _ := filepath.Abs(m.Editor().Path())
	m = m.lspReply(definitionMsg{locs: []lsp.Location{{Path: abs, Pos: lsp.Pos{Line: 0, Col: 4}}}})
	if got := m.Editor().Cursor(); got != (editor.Position{Line: 0, Column: 4}) {
		t.Fatalf("gd no mesmo arquivo move o cursor: %v", got)
	}
	// In another file: it opens in a buffer; Ctrl-o comes back.
	other := filepath.Join(filepath.Dir(abs), "ft_add.c")
	if err := os.WriteFile(other, []byte("#include \"ft.h\"\n\tint\tft_add(int a, int b);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = m.lspReply(definitionMsg{locs: []lsp.Location{{Path: other, Pos: lsp.Pos{Line: 1, Col: 1}}}})
	if filepath.Base(m.Editor().Path()) != "ft_add.c" || m.Editor().Cursor() != (editor.Position{Line: 1, Column: 1}) || !strings.Contains(ansi.Strip(m.statusLine()), "ft_add.c [2/2]") {
		t.Fatalf("gd em outro arquivo: %s %v %q", m.Editor().Path(), m.Editor().Cursor(), ansi.Strip(m.statusLine()))
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlO})
	if filepath.Base(m.Editor().Path()) != "main.c" || m.Editor().Cursor() != (editor.Position{Line: 0, Column: 4}) {
		t.Fatalf("ctrl+o volta: %s %v", m.Editor().Path(), m.Editor().Cursor())
	}
	m = m.lspReply(definitionMsg{})
	if m.message != "definição não encontrada" {
		t.Fatalf("sem definição: %q", m.message)
	}

	m.lsp.diags = []lsp.Diagnostic{
		{Start: lsp.Pos{Line: 4, Col: 9}, Severity: lsp.Error, Message: "b"},
		{Start: lsp.Pos{Line: 0, Col: 0}, Severity: lsp.Warning, Message: "a"},
	}
	m, _ = press(t, m, runes("gg]d"))
	if got := m.Editor().Cursor(); got != (editor.Position{Line: 4, Column: 9}) {
		t.Fatalf("]d vai ao próximo: %v", got)
	}
	m, _ = press(t, m, runes("]d"))
	if got := m.Editor().Cursor(); got != (editor.Position{}) {
		t.Fatalf("]d dá a volta: %v", got)
	}
	m, _ = press(t, m, runes("[d"))
	if got := m.Editor().Cursor(); got != (editor.Position{Line: 4, Column: 9}) {
		t.Fatalf("[d volta (dando a volta): %v", got)
	}
	m.lsp.srv.client, m.lsp.srv.starting = nil, true
	if m, _ = press(t, m, runes("K")); !strings.Contains(m.message, "ainda está iniciando") {
		t.Fatalf("K sem servidor pronto: %q", m.message)
	}
}

// harness runs a model like Bubble Tea does: commands in the background,
// their messages fed back to Update one at a time.
type harness struct {
	t    *testing.T
	m    Model
	msgs chan tea.Msg
}

func (h *harness) exec(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		switch msg := cmd().(type) {
		case nil:
		case tea.BatchMsg:
			for _, c := range msg {
				h.exec(c)
			}
		default:
			h.msgs <- msg
		}
	}()
}

func (h *harness) send(msgs ...tea.Msg) {
	for _, msg := range msgs {
		next, cmd := h.m.Update(msg)
		h.m = next.(Model)
		h.exec(cmd)
	}
}

// until feeds messages until cond holds.
func (h *harness) until(what string, cond func() bool) {
	h.t.Helper()
	for !cond() {
		select {
		case msg := <-h.msgs:
			h.send(msg)
		case <-time.After(70 * time.Second):
			h.t.Fatalf("%s: não aconteceu (mensagem %q)\n%s", what, h.m.message, screen(h.m))
		}
	}
}

// The real clangd: completion while typing, K and gd.
func TestLSPFeaturesWithClangd(t *testing.T) {
	if _, err := exec.LookPath("clangd"); err != nil {
		t.Skip("clangd não instalado")
	}
	path := filepath.Join(t.TempDir(), "main.c")
	text := "int\tadd(int a, int b)\n{\n\treturn (a + b);\n}\n\nint\tcounter;\n\nint\tmain(void)\n{\n\treturn (add(1, 2));\n}\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, m: NewVim(ed).WithLSP(), msgs: make(chan tea.Msg, 16)}
	defer h.m.Close()
	h.send(tea.WindowSizeMsg{Width: 80, Height: 14})
	h.exec(h.m.Init())
	h.until("o clangd iniciar", func() bool { return !h.m.lsp.srv.starting })
	if h.m.lsp.srv.client == nil {
		t.Skipf("clangd não iniciou aqui: %s", h.m.message)
	}

	// Line 9 is "\treturn (add(1, 2));": K on "add" shows its signature.
	h.send(runes("10Gww"), runes("K"))
	h.until("o hover", func() bool { return h.m.lsp.hover != nil })
	if s := screen(h.m); !strings.Contains(s, "int add(int a, int b)") {
		t.Fatalf("hover:\n%s", s)
	}
	h.send(runes("gd"))
	h.until("o gd", func() bool { return h.m.Editor().Cursor().Line == 0 })

	// Typing "cou" in a new line of main opens the list with counter; tab
	// takes it.
	h.send(runes("10G"), runes("O"), runes("c"), runes("o"), runes("u"))
	h.until("a completion", func() bool { return h.m.lsp.comp != nil && len(h.m.lsp.comp.shown) > 0 })
	for h.m.lsp.comp.shown[h.m.lsp.comp.selected].Text != "counter" {
		h.send(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	h.send(tea.KeyMsg{Type: tea.KeyTab})
	if line := h.m.Editor().Buffer().Line(h.m.Editor().Cursor().Line); strings.TrimSpace(line) != "counter" {
		t.Fatalf("tab deveria completar: %q", line)
	}
}
