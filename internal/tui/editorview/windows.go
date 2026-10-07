package editorview

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// Windows (splits): :sp and :vsp (Ctrl-w s / v) show another window, all
// stacked or all side by side; Ctrl-w w/h/j/k/l move between them, :close
// (Ctrl-w c) and :only (Ctrl-w o) close them, and :q closes the window
// while there is more than one. Each window shows a buffer; two windows on
// the same buffer share its cursor (it lives in the editor). Resizing is in
// resize.go.

var (
	styleTitle       = lipgloss.NewStyle().Foreground(colorMuted).Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"})
	styleTitleActive = styleTitle.Foreground(colorAccent).Bold(true)
)

type window struct {
	buf    *buffer
	weight float64 // its share of the screen (see resize.go)
}

// rect is where a window's text goes on the screen; its title line (when
// there's more than one window) is right below.
type rect struct{ x, y, w, h int }

// rects lays the windows out over the screen above the status line.
func (m Model) rects() []rect {
	n, width, height := len(m.ses.wins), m.ses.width, max(m.ses.height-1, 1)
	if n == 1 {
		return []rect{{0, 0, width, height}}
	}
	shares := make([]float64, n)
	for i, w := range m.ses.wins {
		shares[i] = w.share()
	}
	out := make([]rect, 0, n)
	if m.ses.vertical {
		x := 0
		for _, w := range spread(width-(n-1), shares, leastWidth) { // separators between them
			out = append(out, rect{x, 0, max(w, 1), max(height-1, 1)})
			x += w + 1
		}
		return out
	}
	y := 0
	for _, h := range spread(height, shares, leastHeight) {
		out = append(out, rect{0, y, width, max(h-1, 1)})
		y += h
	}
	return out
}

// layout sizes each buffer's viewport to its window.
func (m Model) layout() {
	if m.width == 0 {
		return
	}
	for i, r := range m.rects() {
		ed := m.ses.wins[i].buf.ed
		ed.SetViewportSize(max(r.w-gutterWidth(ed.Buffer().LineCount()), 1), r.h)
	}
}

// screen draws every window: the rows above the status line.
func (m Model) screen() []string {
	rects := m.rects()
	rows := make([]string, max(m.height-1, 1))
	several := len(m.ses.wins) > 1
	for i, w := range m.ses.wins {
		r, active := rects[i], i == m.ses.win
		lines := m.renderWindow(w.buf, r, active)
		if several {
			lines = append(lines, title(w.buf, r.w, active))
		}
		for j, line := range lines {
			row := r.y + j
			if row >= len(rows) {
				break
			}
			if pad := r.w - ansi.StringWidth(line); pad > 0 && several {
				line += strings.Repeat(" ", pad)
			}
			if m.ses.vertical && i > 0 {
				border := styleGutter
				if d := m.ses.drag; d != nil && d.k == i-1 {
					border = styleGutterHere // being dragged
				}
				rows[row] += border.Render("│")
			}
			rows[row] += line
		}
	}
	return rows
}

// title is the line under a window: its file, highlighted when active.
func title(b *buffer, width int, active bool) string {
	name := "[sem nome]"
	if p := b.ed.Path(); p != "" {
		name = filepath.Base(p)
	}
	if b.ed.Dirty() {
		name += " [+]"
	}
	if b.ed.ReadOnly() {
		name += " [só leitura]"
	}
	style := styleTitle
	if active {
		style = styleTitleActive
	}
	text := ansi.Truncate(" "+name+" ", width, "…")
	return style.Render(text + strings.Repeat(" ", max(width-ansi.StringWidth(text), 0)))
}

// windowAt is the window whose text is at screen cell (x, y), or -1.
func (m Model) windowAt(x, y int) (int, rect) {
	for i, r := range m.rects() {
		if x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h {
			return i, r
		}
	}
	return -1, rect{}
}

// focus makes window i the current one.
func (m Model) focus(i int) {
	m.ses.win = i
	b := m.ses.wins[i].buf
	m.ses.cur = slices.Index(m.ses.bufs, b)
	if m.vim != nil {
		m.vim.SetEditor(b.ed)
	}
}

// split opens a window on the current buffer (or on file), above or left
// of the current one, and moves into it.
func (m Model) split(vertical bool, file string) vim.Result {
	ses := m.ses
	if len(ses.wins) > 1 && ses.vertical != vertical {
		return vim.Result{Message: "misturar :sp e :vsp ainda não dá (:only volta a uma janela)", Err: true}
	}
	ses.vertical = vertical
	// The new window takes half of the current one, like Vim.
	cur := ses.wins[ses.win]
	half := cur.share() / 2
	cur.weight = half
	ses.wins = slices.Insert(ses.wins, ses.win, &window{buf: ses.current(), weight: half})
	m.focus(ses.win)
	if file != "" {
		if err := m.open(file); err != nil {
			return vim.Result{Message: err.Error(), Err: true}
		}
	}
	return vim.Result{}
}

// closeWindow closes the current window (never the last one).
func (m Model) closeWindow() vim.Result {
	ses := m.ses
	if len(ses.wins) == 1 {
		return vim.Result{Message: "E444: é a última janela (:q sai)", Err: true}
	}
	// Its space goes to a neighbor.
	gone := ses.wins[ses.win].share()
	ses.wins = slices.Delete(ses.wins, ses.win, ses.win+1)
	next := min(ses.win, len(ses.wins)-1)
	ses.wins[next].weight = ses.wins[next].share() + gone
	m.focus(next)
	return vim.Result{}
}

// windowCommands are Ctrl-w followed by a key.
func (m Model) windowCommands() map[string]func(int) vim.Result {
	move := func(d int, vertical bool) func(int) vim.Result {
		return func(n int) vim.Result {
			if len(m.ses.wins) > 1 && m.ses.vertical == vertical {
				m.focus(max(0, min(len(m.ses.wins)-1, m.ses.win+d*n)))
			}
			return vim.Result{}
		}
	}
	cycle := func(d int) func(int) vim.Result {
		return func(int) vim.Result {
			m.focus((m.ses.win + d + len(m.ses.wins)) % len(m.ses.wins))
			return vim.Result{}
		}
	}
	return map[string]func(int) vim.Result{
		"ctrl+ww": cycle(1), "ctrl+wctrl+w": cycle(1), "ctrl+wW": cycle(-1),
		"ctrl+wh": move(-1, true), "ctrl+wl": move(1, true),
		"ctrl+wk": move(-1, false), "ctrl+wj": move(1, false),
		"ctrl+ws": func(int) vim.Result { return m.split(false, "") },
		"ctrl+wv": func(int) vim.Result { return m.split(true, "") },
		"ctrl+wc": func(int) vim.Result { return m.closeWindow() },
		"ctrl+wq": func(int) vim.Result { return m.vim.HandleText(":q\r") },
		"ctrl+wo": func(int) vim.Result { return m.only() },
	}
}

func (m Model) only() vim.Result {
	m.ses.wins = []*window{m.ses.wins[m.ses.win]}
	m.ses.win = 0
	return vim.Result{}
}
