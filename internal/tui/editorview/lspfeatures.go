package editorview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// What the editor asks the language server: K shows the hover, gd goes to
// the definition, ]d and [d move between diagnostics, and typing (or
// Ctrl-n / Ctrl-Space) opens the completion list.

var (
	stylePopup     = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"})
	stylePopupSel  = styleSelection.Bold(true)
	stylePopupInfo = stylePopup.Foreground(colorMuted)
	styleHover     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorMuted).Padding(0, 1)
)

const popupRows = 8 // completion items shown at once

// completion is the open completion list.
type completion struct {
	items    []lsp.Item      // as the server sent them, best first
	start    editor.Position // where the word being completed starts
	shown    []lsp.Item      // the items matching what is typed
	selected int
}

type hoverMsg struct {
	text string
	err  error
}

type definitionMsg struct {
	locs []lsp.Location
	err  error
}

type completionMsg struct {
	seq   int
	items []lsp.Item
	err   error
}

// lspCommands are the Normal-mode keys the language server answers.
func (m Model) lspCommands() map[string]func(int) vim.Result {
	st, ed := m.lsp, m.ed
	at := func() lsp.Pos { c := ed.Cursor(); return lsp.Pos{Line: c.Line, Col: c.Column} }
	return map[string]func(int) vim.Result{
		"K": func(int) vim.Result {
			p := at()
			return st.request(func(ctx context.Context, c *lsp.Client) tea.Msg {
				text, err := c.Hover(ctx, p)
				return hoverMsg{text, err}
			})
		},
		"gd": func(int) vim.Result {
			p := at()
			return st.request(func(ctx context.Context, c *lsp.Client) tea.Msg {
				locs, err := c.Definition(ctx, p)
				return definitionMsg{locs, err}
			})
		},
		"]d": func(n int) vim.Result { return m.jumpDiagnostic(n) },
		"[d": func(n int) vim.Result { return m.jumpDiagnostic(-n) },
	}
}

// request runs f against the server in the background; Update returns it
// after the key.
func (st *lspState) request(f func(context.Context, *lsp.Client) tea.Msg) vim.Result {
	c := st.client
	if c == nil {
		if st.starting {
			return vim.Result{Message: "o " + st.server.Name + " ainda está iniciando"}
		}
		return vim.Result{Message: "sem language server", Err: true}
	}
	st.pending = func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return f(ctx, c)
	}
	return vim.Result{}
}

// jumpDiagnostic moves to the n-th diagnostic after the cursor (before it
// when n < 0), wrapping around the file.
func (m Model) jumpDiagnostic(n int) vim.Result {
	diags := slices.Clone(m.lsp.diags)
	if len(diags) == 0 {
		return vim.Result{Message: "nenhum diagnóstico"}
	}
	key := func(p lsp.Pos) editor.Position { return editor.Position{Line: p.Line, Column: p.Col} }
	slices.SortFunc(diags, func(a, b lsp.Diagnostic) int {
		if key(a.Start).Before(key(b.Start)) {
			return -1
		}
		if key(b.Start).Before(key(a.Start)) {
			return 1
		}
		return 0
	})
	cur := m.ed.Cursor()
	i := slices.IndexFunc(diags, func(d lsp.Diagnostic) bool { return cur.Before(key(d.Start)) })
	if i < 0 {
		i = len(diags) // past the last one
	}
	if n > 0 {
		i += n - 1
	} else {
		// The one at the cursor doesn't count when going back.
		if j := slices.IndexFunc(diags, func(d lsp.Diagnostic) bool { return !key(d.Start).Before(cur) }); j >= 0 {
			i = j
		}
		i += n
	}
	i = ((i % len(diags)) + len(diags)) % len(diags)
	m.vim.MoveCursor(key(diags[i].Start))
	return vim.Result{}
}

// lspReply handles the server's answers.
func (m Model) lspReply(msg tea.Msg) Model {
	st := m.lsp
	switch msg := msg.(type) {
	case hoverMsg:
		switch {
		case msg.err != nil:
			m.message, m.isError = msg.err.Error(), true
		case msg.text == "":
			m.message = "nada para mostrar aqui"
		default:
			st.hover = strings.Split(ansi.Wrap(msg.text, 70, ""), "\n")
			if len(st.hover) > 12 {
				st.hover = append(st.hover[:11], "…")
			}
		}
	case definitionMsg:
		switch {
		case msg.err != nil:
			m.message, m.isError = msg.err.Error(), true
		case len(msg.locs) == 0:
			m.message = "definição não encontrada"
		default:
			loc := msg.locs[0]
			if !sameFile(loc.Path, m.ed.Path()) {
				m.message = fmt.Sprintf("definição em %s:%d (outro arquivo: abrir ainda não dá)", filepath.Base(loc.Path), loc.Pos.Line+1)
				break
			}
			p := editor.Position{Line: loc.Pos.Line, Column: loc.Pos.Col}
			if m.vim != nil {
				m.vim.MoveCursor(p)
			} else {
				m.ed.MoveCursor(p)
			}
		}
	case completionMsg:
		if msg.seq != st.seq || !m.typing() {
			return m // the user moved on
		}
		if msg.err != nil || len(msg.items) == 0 {
			st.comp = nil
			return m
		}
		start := wordStart(m.ed)
		if it := msg.items[0]; it.HasStart && it.Start.Line == start.Line && it.Start.Col <= m.ed.Cursor().Column {
			start.Column = it.Start.Col
		}
		st.comp = &completion{items: msg.items, start: start}
		m.filterCompletion()
	}
	return m
}

// typing reports Insert mode (or the plain editor), where completion works.
func (m Model) typing() bool {
	return m.vim == nil || m.vim.Mode() == vim.Insert
}

// completionKey handles the keys of an open completion list; false lets
// the key through to the editor.
func (m Model) completionKey(k string) bool {
	comp := m.lsp.comp
	switch k {
	case "ctrl+n", "down":
		comp.selected = (comp.selected + 1) % len(comp.shown)
	case "ctrl+p", "up":
		comp.selected = (comp.selected + len(comp.shown) - 1) % len(comp.shown)
	case "tab", "enter":
		item := comp.shown[comp.selected]
		m.ed.Replace(editor.Range{Start: comp.start, End: m.ed.Cursor()}, item.Text)
		m.closeCompletion()
	case "esc":
		m.closeCompletion()
		return false // and leave Insert, like Vim
	default:
		return false
	}
	return true
}

func (m Model) closeCompletion() {
	m.lsp.comp = nil
	m.lsp.seq++ // drop answers still on the way
}

// afterKey keeps completion in step with what was typed: it asks the server
// on Ctrl-n / Ctrl-Space, on the first letter of a word and after ".", "->"
// and "::", and narrows an open list as the word grows.
func (m Model) afterKey(msg tea.KeyMsg) tea.Cmd {
	st := m.lsp
	if !m.typing() {
		st.comp = nil
		return nil
	}
	k := msg.String()
	trigger := k == "ctrl+n" || k == "ctrl+@" || k == "ctrl+ "
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		before := m.ed.Cursor()
		before.Column -= 2
		prev, _ := m.ed.Buffer().RuneAt(before)
		switch {
		case r == '.' || (r == '>' && prev == '-') || (r == ':' && prev == ':'):
			trigger = true
		case isWordRune(r) && st.comp == nil:
			trigger = true
		}
	}
	if st.comp != nil && !trigger {
		m.filterCompletion()
		return nil
	}
	if !trigger || st.client == nil {
		return nil
	}
	st.seq++
	seq, c := st.seq, st.client
	cur := m.ed.Cursor()
	p := lsp.Pos{Line: cur.Line, Col: cur.Column}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, err := c.Completion(ctx, p)
		return completionMsg{seq, items, err}
	}
}

// filterCompletion shows the items starting with what was typed since the
// list opened, closing it when nothing matches or the cursor left the word.
func (m Model) filterCompletion() {
	comp, cur := m.lsp.comp, m.ed.Cursor()
	if comp == nil {
		return
	}
	if cur.Line != comp.start.Line || cur.Column < comp.start.Column {
		m.lsp.comp = nil
		return
	}
	typed := strings.ToLower(m.ed.Buffer().Slice(editor.Range{Start: comp.start, End: cur}))
	comp.shown = comp.shown[:0]
	for _, it := range comp.items {
		if strings.HasPrefix(strings.ToLower(it.Text), typed) {
			comp.shown = append(comp.shown, it)
		}
	}
	// Nothing left, or only what is already typed: close.
	if len(comp.shown) == 0 || (len(comp.shown) == 1 && strings.EqualFold(comp.shown[0].Text, typed)) {
		m.lsp.comp = nil
		return
	}
	comp.selected = min(comp.selected, len(comp.shown)-1)
}

// sameFile compares paths on disk (macOS's /var is /private/var).
func sameFile(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fa, fb)
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// wordStart is where the word before the cursor starts.
func wordStart(ed *editor.Editor) editor.Position {
	p := ed.Cursor()
	for p.Column > 0 {
		r, _ := ed.Buffer().RuneAt(editor.Position{Line: p.Line, Column: p.Column - 1})
		if !isWordRune(r) {
			break
		}
		p.Column--
	}
	return p
}

// popups draws the hover box or the completion list over the text rows,
// below the cursor (above when there's no room).
func (m Model) popups(rows []string) {
	st := m.lsp
	if st == nil || (st.hover == nil && st.comp == nil) {
		return
	}
	v, cur := m.ed.Viewport(), m.ed.Cursor()
	row := cur.Line - v.Top
	col := m.gutterWidth() + m.ed.VisualColumn(cur.Line, cur.Column) - v.Left
	var box []string
	if st.comp != nil {
		box = m.completionBox()
		col = m.gutterWidth() + m.ed.VisualColumn(st.comp.start.Line, st.comp.start.Column) - v.Left
	} else {
		box = strings.Split(styleHover.Render(strings.Join(st.hover, "\n")), "\n")
	}
	top := row + 1
	if top+len(box) > len(rows) && row-len(box) >= 0 {
		top = row - len(box)
	}
	w := 0
	for _, line := range box {
		w = max(w, ansi.StringWidth(line))
	}
	col = max(min(col, m.width-w), 0)
	for i, line := range box {
		if r := top + i; r >= 0 && r < len(rows) {
			rows[r] = overlay(rows[r], col, line, m.width)
		}
	}
}

// completionBox renders the visible part of the completion list.
func (m Model) completionBox() []string {
	comp := m.lsp.comp
	from := max(0, min(comp.selected-popupRows/2, len(comp.shown)-popupRows))
	to := min(len(comp.shown), from+popupRows)
	labelW, detailW := 0, 0
	for _, it := range comp.shown[from:to] {
		labelW = max(labelW, ansi.StringWidth(it.Label))
		detailW = max(detailW, ansi.StringWidth(it.Detail))
	}
	labelW, detailW = min(labelW, 40), min(detailW, 24)
	var box []string
	for i := from; i < to; i++ {
		it := comp.shown[i]
		label := ansi.Truncate(it.Label, labelW, "…")
		label += strings.Repeat(" ", labelW-ansi.StringWidth(label))
		detail := ansi.Truncate(it.Detail, detailW, "…")
		detail += strings.Repeat(" ", detailW-ansi.StringWidth(detail))
		if i == comp.selected {
			box = append(box, stylePopupSel.Render(" "+label+"  "+detail+" "))
		} else {
			box = append(box, stylePopup.Render(" "+label+"  ")+stylePopupInfo.Render(detail+" "))
		}
	}
	return box
}

// overlay draws box over row from screen column col, keeping what's left
// of row on both sides.
func overlay(row string, col int, box string, width int) string {
	box = ansi.Truncate(box, width-col, "")
	left := ansi.Truncate(row, col, "")
	if pad := col - ansi.StringWidth(left); pad > 0 {
		left += strings.Repeat(" ", pad)
	}
	return left + box + ansi.TruncateLeft(row, col+ansi.StringWidth(box), "")
}
