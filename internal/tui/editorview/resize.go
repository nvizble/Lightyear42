package editorview

import (
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// Resizing windows: drag the border between them with the mouse (the │
// between side-by-side windows, the title line under a stacked one), or,
// like Vim, Ctrl-w > < (width), + - (height), = (all equal), | _ (as big as
// possible), :vertical resize N and :resize N (N, +N or -N). Sizes are
// shares of the screen, so they hold when the terminal resizes.

// Smallest window: room for the line numbers and some code, or a line of
// text and the title.
const (
	leastWidth  = 12
	leastHeight = 2
)

// drag is a border being dragged: the one after window k.
type drag struct{ k int }

// share is a window's part of the screen; 0 (never set) counts as 1.
func (w *window) share() float64 {
	if w.weight <= 0 {
		return 1
	}
	return w.weight
}

// spread divides total cells by shares (rounding the edges between them),
// giving each at least least cells when there's room.
func spread(total int, shares []float64, least int) []int {
	sum := 0.0
	for _, s := range shares {
		sum += s
	}
	sizes := make([]int, len(shares))
	acc, prev := 0.0, 0
	for i, s := range shares {
		acc += s
		edge := int(math.Round(float64(total) * acc / sum))
		sizes[i], prev = edge-prev, edge
	}
	if total < least*len(sizes) {
		return sizes
	}
	for i := range sizes {
		for sizes[i] < least {
			big := 0
			for j, s := range sizes {
				if s > sizes[big] {
					big = j
				}
			}
			sizes[big]--
			sizes[i]++
		}
	}
	return sizes
}

// sizes are the windows' sizes along the split (widths side by side,
// heights with their title lines stacked), and the least each may have.
func (m Model) sizes() ([]int, int) {
	var out []int
	for _, r := range m.rects() {
		if m.ses.vertical {
			out = append(out, r.w)
		} else {
			out = append(out, r.h+1)
		}
	}
	if m.ses.vertical {
		return out, leastWidth
	}
	return out, leastHeight
}

// setSizes makes the windows' shares match sizes (in cells).
func (m Model) setSizes(sizes []int) {
	for i, w := range m.ses.wins {
		w.weight = float64(sizes[i])
	}
}

// grow makes window i delta cells bigger (smaller when negative), taking
// them from its neighbor: the next window, or the previous for the last.
func (m Model) grow(i, delta int) {
	n := len(m.ses.wins)
	if n < 2 || m.ses.width == 0 {
		return
	}
	j := i + 1
	if j == n {
		j = i - 1
	}
	m.trade(i, j, delta)
}

// trade moves delta cells from window j to window i, keeping both at
// their least.
func (m Model) trade(i, j, delta int) {
	sizes, least := m.sizes()
	delta = max(min(delta, sizes[j]-least), least-sizes[i])
	sizes[i] += delta
	sizes[j] -= delta
	m.setSizes(sizes)
}

// borderAt is the border (after window k) at screen cell (x, y).
func (m Model) borderAt(x, y int) (int, bool) {
	rects := m.rects()
	for k := 0; k < len(rects)-1; k++ {
		r := rects[k]
		if (m.ses.vertical && x == r.x+r.w && y < r.y+r.h+1) || (!m.ses.vertical && y == r.y+r.h) {
			return k, true
		}
	}
	return 0, false
}

// moveBorder puts the border after window k at screen column (or row) pos.
func (m Model) moveBorder(k, pos int) {
	r := m.rects()[k]
	sizes, _ := m.sizes()
	want := pos - r.x
	if !m.ses.vertical {
		want = pos - r.y + 1 // the title line is the border
	}
	m.trade(k, k+1, want-sizes[k])
}

// dragBorder handles the mouse on a border; true when it took the event.
func (m Model) dragBorder(msg tea.MouseMsg) bool {
	if d := m.ses.drag; d != nil {
		if msg.Action == tea.MouseActionMotion && msg.Button == tea.MouseButtonLeft {
			pos := msg.X
			if !m.ses.vertical {
				pos = msg.Y
			}
			m.moveBorder(d.k, pos)
			return true
		}
		m.ses.drag = nil // let go
		return msg.Action == tea.MouseActionRelease
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		if k, ok := m.borderAt(msg.X, msg.Y); ok {
			m.ses.drag = &drag{k}
			return true
		}
	}
	return false
}

// resizeCommands are Ctrl-w > < + - = | _.
func (m Model) resizeCommands() map[string]func(int) vim.Result {
	along := func(vertical bool, d int) func(int) vim.Result {
		return func(n int) vim.Result {
			if m.ses.vertical == vertical {
				m.grow(m.ses.win, d*n)
			}
			return vim.Result{}
		}
	}
	biggest := func(vertical bool) func(int) vim.Result {
		return func(int) vim.Result {
			if m.ses.vertical == vertical {
				for i := range m.ses.wins {
					if i != m.ses.win {
						m.trade(m.ses.win, i, m.ses.width+m.ses.height) // trade keeps i at its least
					}
				}
			}
			return vim.Result{}
		}
	}
	return map[string]func(int) vim.Result{
		"ctrl+w>": along(true, 1), "ctrl+w<": along(true, -1),
		"ctrl+w+": along(false, 1), "ctrl+w-": along(false, -1),
		"ctrl+w|": biggest(true), "ctrl+w_": biggest(false),
		"ctrl+w=": func(int) vim.Result {
			for _, w := range m.ses.wins {
				w.weight = 0
			}
			return vim.Result{}
		},
	}
}

// resizeTo is :resize (height) and :vertical resize (width): N cells, or
// +N / -N more or less.
func (m Model) resizeTo(vertical bool, arg string) vim.Result {
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "+"))
	if err != nil {
		return vim.Result{Message: "uso: :resize N, +N ou -N (:vertical resize para a largura)", Err: true}
	}
	if len(m.ses.wins) < 2 || m.ses.vertical != vertical {
		return vim.Result{}
	}
	delta := n
	if arg[0] != '+' && arg[0] != '-' {
		sizes, _ := m.sizes()
		if !vertical {
			n++ // the title line
		}
		delta = n - sizes[m.ses.win]
	}
	m.grow(m.ses.win, delta)
	return vim.Result{}
}

// Shares are the windows' sizes, to give a later editor the same layout
// (WithShares).
func (m Model) Shares() []float64 {
	out := make([]float64, len(m.ses.wins))
	for i, w := range m.ses.wins {
		out[i] = w.share()
	}
	return out
}

// WithShares sizes the windows like shares (from Shares), when there are as
// many.
func (m Model) WithShares(shares []float64) Model {
	if len(shares) == len(m.ses.wins) {
		for i, w := range m.ses.wins {
			w.weight = shares[i]
		}
	}
	return m
}
