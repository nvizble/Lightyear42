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

func TestWindows(t *testing.T) {
	dir := t.TempDir()
	mainc, ftc := filepath.Join(dir, "main.c"), filepath.Join(dir, "ft.c")
	for path, text := range map[string]string{mainc: "int\tmain(void);\n", ftc: "int\tft(void);\n"} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ed, err := editor.Open(mainc)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := NewVim(ed).Update(tea.WindowSizeMsg{Width: 60, Height: 8})
	m := next.(Model)
	ex := func(cmd string) {
		t.Helper()
		m, _ = press(t, m, runes(":"+cmd), tea.KeyMsg{Type: tea.KeyEnter})
	}
	screen := func() []string { return strings.Split(ansi.Strip(m.View()), "\n") }
	name := func() string { return filepath.Base(m.Editor().Path()) }

	// :vsp ft.c: side by side, the new window on the left and active.
	ex("vsp " + ftc)
	rows := screen()
	if len(m.ses.wins) != 2 || name() != "ft.c" || !strings.Contains(rows[0], "1 │ int ft(void);") || !strings.Contains(rows[0], "│1 │ int main(void);") {
		t.Fatalf(":vsp lado a lado:\n%s", strings.Join(rows, "\n"))
	}
	if !strings.Contains(rows[6], " ft.c ") || !strings.Contains(rows[6], " main.c ") {
		t.Fatalf("títulos das janelas: %q", rows[6])
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlW}, runes("l"))
	if name() != "main.c" || m.ses.win != 1 {
		t.Fatalf("ctrl+w l: %s %d", name(), m.ses.win)
	}
	ex("sp")
	if !m.isError || !strings.Contains(m.message, "misturar") {
		t.Fatalf(":sp com :vsp aberto: %q", m.message)
	}
	// The same buffer in two windows shows the same text.
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlW}, runes("w"), runes("ix"), tea.KeyMsg{Type: tea.KeyEsc})
	if name() != "ft.c" || !strings.Contains(screen()[0], "1 │ xint") {
		t.Fatalf("ctrl+w w e editar: %s\n%s", name(), strings.Join(screen(), "\n"))
	}
	// A click in the other window moves there.
	m, _ = press(t, m, tea.MouseMsg{X: 40, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if name() != "main.c" {
		t.Fatalf("clique na outra janela: %s", name())
	}
	// :q closes the window (the buffer stays), with changes elsewhere too.
	ex("q")
	if len(m.ses.wins) != 1 || len(m.ses.bufs) != 2 || m.Done() {
		t.Fatalf(":q fecha a janela: %d janelas, %d buffers", len(m.ses.wins), len(m.ses.bufs))
	}
	// :sp stacks them, each with its title; :only leaves one.
	ex("sp")
	rows = screen()
	if len(m.ses.wins) != 2 || !strings.Contains(rows[2], " ft.c [+] ") || !strings.Contains(rows[6], " ft.c [+] ") {
		t.Fatalf(":sp empilha:\n%s", strings.Join(rows, "\n"))
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlW}, runes("o"))
	if len(m.ses.wins) != 1 {
		t.Fatal("ctrl+w o deixa uma janela")
	}
	ex("close")
	if !strings.Contains(m.message, "E444") {
		t.Fatalf(":close na última: %q", m.message)
	}
}
