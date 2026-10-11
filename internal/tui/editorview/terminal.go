package editorview

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// The terminal (:term): a panel under the windows running a shell, or a
// command (:term make), in a pseudo-terminal; vt10x keeps its screen. While
// it has the focus, keys go to the shell; Ctrl-\ Ctrl-n (Vim's way) or
// Ctrl-w and a direction go back to the code, and a click on either side
// moves the focus. When the shell ends, any key closes the panel.

type terminal struct {
	vt    vt10x.Terminal
	pty   *os.File
	cmd   *exec.Cmd
	title string
	// output gets a value when the shell writes (coalesced) and is closed
	// when it ends; Update waits on it (wait).
	output chan struct{}

	mu       sync.Mutex
	exited   bool
	exitText string
}

// termOutputMsg says the terminal has something new to show.
type termOutputMsg struct {
	t      *terminal
	closed bool
}

// startTerminal runs command (the shell when empty) in dir, on a
// pseudo-terminal of cols × rows.
func startTerminal(dir, command string, cols, rows int) (*terminal, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("o terminal ainda não funciona no Windows")
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	title := filepath.Base(shell)
	if command != "" {
		cmd = exec.Command(shell, "-c", command)
		title = command
	}
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	cols, rows = max(cols, 2), max(rows, 1)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, fmt.Errorf("abrir o terminal: %w", err)
	}
	t := &terminal{
		vt:     vt10x.New(vt10x.WithWriter(f), vt10x.WithSize(cols, rows)),
		pty:    f,
		cmd:    cmd,
		title:  title,
		output: make(chan struct{}, 1),
	}
	go t.read()
	return t, nil
}

// read copies the shell's output into the screen until it ends.
func (t *terminal) read() {
	buf := make([]byte, 32*1024)
	for {
		n, err := t.pty.Read(buf)
		if n > 0 {
			_, _ = t.vt.Write(buf[:n])
			select {
			case t.output <- struct{}{}:
			default:
			}
		}
		if err != nil {
			break
		}
	}
	err := t.cmd.Wait()
	t.mu.Lock()
	t.exited = true
	t.exitText = "processo terminou"
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		t.exitText = fmt.Sprintf("processo terminou com o código %d", exitErr.ExitCode())
	}
	t.mu.Unlock()
	close(t.output)
}

// wait is the command that delivers the next termOutputMsg.
func (t *terminal) wait() tea.Cmd {
	return func() tea.Msg {
		_, ok := <-t.output
		return termOutputMsg{t: t, closed: !ok}
	}
}

func (t *terminal) done() (bool, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exited, t.exitText
}

func (t *terminal) resize(cols, rows int) {
	cols, rows = max(cols, 2), max(rows, 1)
	if c, r := t.vt.Size(); c == cols && r == rows {
		return
	}
	t.vt.Resize(cols, rows)
	_ = pty.Setsize(t.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (t *terminal) write(b []byte) {
	if len(b) > 0 {
		_, _ = t.pty.Write(b)
	}
}

// close ends the process and the pseudo-terminal.
func (t *terminal) close() {
	if exited, _ := t.done(); !exited && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	_ = t.pty.Close()
}

// keyBytes is what a terminal sends for a key, like xterm does.
func keyBytes(msg tea.KeyMsg, appCursor bool) []byte {
	var out []byte
	if msg.Alt {
		out = append(out, 0x1b)
	}
	arrow := func(c byte) []byte {
		if appCursor {
			return []byte{0x1b, 'O', c}
		}
		return []byte{0x1b, '[', c}
	}
	switch msg.Type {
	case tea.KeyRunes:
		return append(out, []byte(string(msg.Runes))...)
	case tea.KeySpace:
		return append(out, ' ')
	case tea.KeyEnter:
		return append(out, '\r')
	case tea.KeyBackspace:
		return append(out, 0x7f)
	case tea.KeyUp:
		return append(out, arrow('A')...)
	case tea.KeyDown:
		return append(out, arrow('B')...)
	case tea.KeyRight:
		return append(out, arrow('C')...)
	case tea.KeyLeft:
		return append(out, arrow('D')...)
	case tea.KeyShiftTab:
		return append(out, "\x1b[Z"...)
	case tea.KeyHome:
		return append(out, "\x1b[H"...)
	case tea.KeyEnd:
		return append(out, "\x1b[F"...)
	case tea.KeyPgUp:
		return append(out, "\x1b[5~"...)
	case tea.KeyPgDown:
		return append(out, "\x1b[6~"...)
	case tea.KeyDelete:
		return append(out, "\x1b[3~"...)
	case tea.KeyInsert:
		return append(out, "\x1b[2~"...)
	case tea.KeyCtrlLeft:
		return append(out, "\x1b[1;5D"...)
	case tea.KeyCtrlRight:
		return append(out, "\x1b[1;5C"...)
	case tea.KeyCtrlUp:
		return append(out, "\x1b[1;5A"...)
	case tea.KeyCtrlDown:
		return append(out, "\x1b[1;5B"...)
	}
	if fn := map[tea.KeyType]string{tea.KeyF1: "\x1bOP", tea.KeyF2: "\x1bOQ", tea.KeyF3: "\x1bOR", tea.KeyF4: "\x1bOS",
		tea.KeyF5: "\x1b[15~", tea.KeyF6: "\x1b[17~", tea.KeyF7: "\x1b[18~", tea.KeyF8: "\x1b[19~",
		tea.KeyF9: "\x1b[20~", tea.KeyF10: "\x1b[21~", tea.KeyF11: "\x1b[23~", tea.KeyF12: "\x1b[24~"}[msg.Type]; fn != "" {
		return append(out, fn...)
	}
	// Ctrl-A … Ctrl-_, Tab and Esc are the control bytes themselves.
	if msg.Type >= 0 && msg.Type < 32 {
		return append(out, byte(msg.Type))
	}
	return nil
}

// terminalHeight is the panel's share of the screen: a title line and the
// shell's rows.
func (m Model) terminalHeight() int {
	if m.ses.term == nil {
		return 0
	}
	return min(max(m.ses.height*2/5, 6), max(m.ses.height-4, 2))
}

// openTerminal is :term [command]: a new terminal, in the current file's
// folder; with the panel open and no command, it only takes the focus.
func (m Model) openTerminal(command string) vim.Result {
	ses := m.ses
	if ses.term != nil {
		if exited, _ := ses.term.done(); command == "" && !exited {
			ses.termFocus = true
			return vim.Result{}
		}
		ses.term.close()
		ses.term = nil
	}
	dir := "."
	if p := ses.current().ed.Path(); p != "" {
		dir = filepath.Dir(p)
	}
	ses.term = &terminal{} // reserves the space for the layout below
	cols, rows := max(ses.width, 2), max(m.terminalHeight()-1, 1)
	t, err := startTerminal(dir, command, cols, rows)
	if err != nil {
		ses.term = nil
		return vim.Result{Message: err.Error(), Err: true}
	}
	ses.term, ses.termFocus = t, true
	ses.starts = append(ses.starts, t.wait())
	return vim.Result{}
}

// closeTerminal removes the panel and gives the focus back to the code.
func (m Model) closeTerminal() {
	if m.ses.term != nil {
		m.ses.term.close()
		m.ses.term, m.ses.termFocus = nil, false
	}
}

// terminalKey handles a key while the terminal has the focus.
func (m Model) terminalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ses := m.ses
	t := ses.term
	if exited, _ := t.done(); exited {
		m.closeTerminal()
		return m, nil
	}
	k := msg.String()
	switch ses.termLead {
	case "ctrl+\\":
		ses.termLead = ""
		if k == "ctrl+n" {
			ses.termFocus = false
			return m, nil
		}
		t.write([]byte{0x1c}) // the Ctrl-\ that was held back
	case "ctrl+w":
		ses.termLead = ""
		switch k {
		case "k", "w", "W", "h", "l", "ctrl+w", "up":
			ses.termFocus = false
			return m, nil
		}
		t.write([]byte{0x17})
	}
	if k == "ctrl+\\" || k == "ctrl+w" {
		ses.termLead = k
		return m, nil
	}
	t.write(keyBytes(msg, t.vt.Mode()&vt10x.ModeAppCursor != 0))
	return m, nil
}

// renderTerminal draws the panel: its title line and the shell's screen.
func (m Model) renderTerminal(width, height int) []string {
	t, focused := m.ses.term, m.ses.termFocus
	name := " terminal · " + t.title + " "
	hint := " Ctrl-\\ Ctrl-n volta ao código "
	if exited, text := t.done(); exited {
		hint = " " + text + " · qualquer tecla fecha "
	} else if !focused {
		hint = " :term ou clique para usar "
	}
	style := styleTitle
	if focused {
		style = styleTitleActive
	}
	// A long command gives way: the hint (how to leave, or that it ended) stays.
	title := ansi.Truncate(name, max(width-ansi.StringWidth(hint), 12), "…")
	if room := width - ansi.StringWidth(title) - ansi.StringWidth(hint); room >= 0 {
		title += strings.Repeat(" ", room) + hint
	}
	title = ansi.Truncate(title, width, "")
	rows := []string{style.Render(title + strings.Repeat(" ", max(width-ansi.StringWidth(title), 0)))}

	t.vt.Lock()
	defer t.vt.Unlock()
	cols, vrows := t.vt.Size()
	cur := t.vt.Cursor()
	showCursor := focused && t.vt.CursorVisible()
	for y := 0; y < height-1; y++ {
		if y >= vrows {
			rows = append(rows, "")
			continue
		}
		var b, run strings.Builder
		var style *lipgloss.Style
		flush := func() {
			if style != nil {
				b.WriteString(style.Render(run.String()))
			} else {
				b.WriteString(run.String())
			}
			run.Reset()
		}
		for x := 0; x < min(cols, width); x++ {
			g := t.vt.Cell(x, y)
			s := glyphStyle(g, showCursor && x == cur.X && y == cur.Y)
			if style == nil || !sameStyle(s, *style) {
				flush()
				style = &s
			}
			ch := g.Char
			if ch == 0 {
				ch = ' '
			}
			run.WriteRune(ch)
		}
		flush()
		rows = append(rows, b.String())
	}
	return rows
}

// glyphStyle is how to draw one cell of the terminal.
func glyphStyle(g vt10x.Glyph, cursor bool) lipgloss.Style {
	const reverse, underline, bold, italic = 1, 2, 4, 16
	s := lipgloss.NewStyle()
	fg, bg := termColor(g.FG), termColor(g.BG)
	if fg != "" {
		s = s.Foreground(lipgloss.Color(fg))
	}
	if bg != "" {
		s = s.Background(lipgloss.Color(bg))
	}
	if g.Mode&bold != 0 {
		s = s.Bold(true)
	}
	if g.Mode&underline != 0 {
		s = s.Underline(true)
	}
	if g.Mode&italic != 0 {
		s = s.Italic(true)
	}
	if (g.Mode&reverse != 0) != cursor {
		s = s.Reverse(true)
	}
	return s
}

// sameStyle compares the parts of two cell styles that can differ.
func sameStyle(a, b lipgloss.Style) bool {
	return a.GetForeground() == b.GetForeground() && a.GetBackground() == b.GetBackground() &&
		a.GetBold() == b.GetBold() && a.GetUnderline() == b.GetUnderline() &&
		a.GetItalic() == b.GetItalic() && a.GetReverse() == b.GetReverse()
}

// termColor turns a vt10x color into a lipgloss one ("" for the default).
func termColor(c vt10x.Color) string {
	switch {
	case c >= vt10x.DefaultFG:
		return ""
	case c < 256:
		return strconv.Itoa(int(c))
	}
	return fmt.Sprintf("#%06x", uint32(c)&0xffffff)
}
