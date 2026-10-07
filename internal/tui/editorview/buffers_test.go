package editorview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestBuffers(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	mainc, ftc := write("main.c", "int\tmain(void);\n"), write("ft.c", "int\tft(void);\n")
	ed, err := editor.Open(mainc)
	if err != nil {
		t.Fatal(err)
	}
	m := NewVim(ed).WithLSP()
	defer m.Close()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	m = next.(Model)
	ex := func(cmd string) Model {
		t.Helper()
		m, _ = press(t, m, runes(":"+cmd), tea.KeyMsg{Type: tea.KeyEnter})
		return m
	}
	name := func() string { return filepath.Base(m.Editor().Path()) }

	m = ex("e " + ftc)
	if name() != "ft.c" || !strings.Contains(ansi.Strip(m.statusLine()), "ft.c [2/2]") {
		t.Fatalf(":e abre outro buffer: %s %q", name(), ansi.Strip(m.statusLine()))
	}
	// One server for the project, shared by both buffers.
	if len(m.ses.servers) != 1 || m.ses.bufs[0].lsp.srv != m.ses.bufs[1].lsp.srv {
		t.Fatalf("um servidor por projeto: %d", len(m.ses.servers))
	}
	// Registers work across buffers.
	m, _ = press(t, m, runes("yy"))
	m = ex("bp")
	m, _ = press(t, m, runes("p"))
	if name() != "main.c" || m.Editor().Buffer().Text() != "int\tmain(void);\nint\tft(void);\n" {
		t.Fatalf(":bp e o register: %s %q", name(), m.Editor().Buffer().Text())
	}
	if m = ex("ls"); m.message != "1% main.c [+]  2  ft.c" {
		t.Fatalf(":ls: %q", m.message)
	}
	if m = ex("q"); !m.isError || !strings.Contains(m.message, "E37") {
		t.Fatalf(":q com o buffer atual modificado: %q", m.message)
	}
	m = ex("b 2")
	if m = ex("q"); !m.isError || !strings.Contains(m.message, "E162: main.c") {
		t.Fatalf(":q com outro buffer modificado: %q", m.message)
	}
	if m = ex("b ma"); name() != "main.c" {
		t.Fatalf(":b por nome: %s", name())
	}
	if m = ex("bd"); !m.isError || !strings.Contains(m.message, "E89") {
		t.Fatalf(":bd com alterações: %q", m.message)
	}
	if m = ex("wa"); m.message != "1 arquivo(s) salvo(s)" {
		t.Fatalf(":wa: %q", m.message)
	}
	m = ex("bd")
	if len(m.ses.bufs) != 1 || name() != "ft.c" {
		t.Fatalf(":bd fecha o buffer: %d %s", len(m.ses.bufs), name())
	}
	if m = ex("bd"); !strings.Contains(m.message, "único buffer") {
		t.Fatalf(":bd no último: %q", m.message)
	}
	if m = ex("e"); !strings.Contains(m.message, "E32") {
		t.Fatalf(":e sem nome: %q", m.message)
	}
	m, cmd := press(t, m, runes(":q"), tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Done() || cmd == nil {
		t.Fatal(":q sem nada modificado sai")
	}
	if data, _ := os.ReadFile(mainc); !strings.Contains(string(data), "ft(void)") {
		t.Fatalf(":wa salvou o main.c: %q", data)
	}
}
