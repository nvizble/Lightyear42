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

// setLSP gives the current buffer a language server side, with no server.
func setLSP(m Model, st *lspState) Model {
	if st.srv == nil {
		st.srv = &lspServer{}
	}
	m.ses.current().lsp = st
	return m.synced()
}

func TestDiagnosticsOnScreen(t *testing.T) {
	colors(t)
	next, _ := NewVim(editor.New("int x\nok")).Update(tea.WindowSizeMsg{Width: 70, Height: 6})
	m := next.(Model)
	m = setLSP(m, &lspState{diags: []lsp.Diagnostic{
		{Start: lsp.Pos{Line: 0, Col: 5}, End: lsp.Pos{Line: 0, Col: 5}, Severity: lsp.Error, Message: "expected ';'\nnota"},
		{Start: lsp.Pos{Line: 0, Col: 4}, End: lsp.Pos{Line: 0, Col: 5}, Severity: lsp.Warning, Message: "unused variable 'x'"},
	}})
	rows := strings.Split(m.View(), "\n")
	// The worst severity colors the gutter; the code is underlined, past
	// the end too (the missing ";").
	if !strings.HasPrefix(rows[0], severityStyles[lsp.Error].Render("1 ● ")) {
		t.Fatalf("gutter: %q", rows[0])
	}
	if !strings.Contains(rows[0], markStyles[lsp.Warning].Render("x")+markStyles[lsp.Error].Render(" ")) {
		t.Fatalf("marcas no código: %q", rows[0])
	}
	if strings.Contains(rows[1], "●") {
		t.Fatalf("a linha 2 não tem diagnóstico: %q", rows[1])
	}
	status := ansi.Strip(rows[len(rows)-1])
	if !strings.Contains(status, "erro: expected ';' ") || strings.Contains(status, "nota") {
		t.Fatalf("a statusline deveria mostrar o erro da linha do cursor (só a 1ª linha): %q", status)
	}
	m, _ = press(t, m, runes("j"))
	if status := ansi.Strip(m.statusLine()); strings.Contains(status, "erro:") {
		t.Fatalf("fora da linha, voltam as dicas: %q", status)
	}
}

// End to end with the real clangd, when installed: the unused variable is
// an error (42 flags), and deleting it clears the diagnostics.
func TestLSPWithClangd(t *testing.T) {
	if _, err := exec.LookPath("clangd"); err != nil {
		t.Skip("clangd não instalado")
	}
	path := filepath.Join(t.TempDir(), "main.c")
	if err := os.WriteFile(path, []byte("int\tmain(void)\n{\n\tint\tx;\n\n\treturn (0);\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	m := NewVim(ed).WithLSP()
	defer m.Close()
	start := m.Init() // Bubble Tea calls Init before any Update
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = next.(Model)
	if status := ansi.Strip(m.statusLine()); !strings.Contains(status, "C · clangd…") {
		t.Fatalf("deveria mostrar o servidor iniciando: %q", status)
	}

	// run feeds a command's message to the model, failing on a stuck one.
	run := func(cmd tea.Cmd) tea.Cmd {
		t.Helper()
		got := make(chan tea.Msg, 1)
		go func() { got <- cmd() }()
		select {
		case msg := <-got:
			next, cmd := m.Update(msg)
			m = next.(Model)
			return cmd
		case <-time.After(70 * time.Second):
			t.Fatalf("o servidor não respondeu: diagnostics %+v, mensagem %q", m.lsp.diags, m.message)
			return nil
		}
	}
	cmd := run(start)
	if m.lsp.srv.client == nil {
		t.Skipf("clangd não iniciou aqui: %s", m.message)
	}
	for m.worst(2) == nil {
		cmd = run(cmd)
	}
	m, _ = press(t, m, runes("3G"))
	if status := ansi.Strip(m.statusLine()); !strings.Contains(strings.ToLower(status), "erro: unused variable 'x'") || !strings.Contains(status, "clangd ✖ 1") {
		t.Fatalf("statusline: %q", status)
	}

	m, _ = press(t, m, runes("dd"))
	for len(m.lsp.diags) != 0 {
		cmd = run(cmd)
	}
}

// A server lightyear downloads first (clangd on Linux) says so while it
// starts.
func TestStatusWhileDownloadingServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.c")
	if err := os.WriteFile(path, []byte("int\tmain(void);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	m := NewVim(ed).WithLSP() // the start command is never run
	defer m.Close()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 10})
	m = next.(Model)
	m.lsp.srv.server.Download, m.lsp.srv.fetching = &lsp.Download{MB: 118}, true
	if status := ansi.Strip(m.statusLine()); !strings.Contains(status, "C · baixando o clangd (só na 1ª vez, 118 MB)…") {
		t.Fatalf("deveria avisar do download: %q", status)
	}
}
