package editorview

import (
	"context"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
)

// Diagnostics on demand: resting the mouse on underlined code (or on the
// ● in the gutter) shows the full messages in a box — and when the server
// has a fix, a click on the box applies it (listing them when there are
// several); K shows the ones on the cursor's line above the server's hover
// text.

var styleFixButton = lipgloss.NewStyle().Foreground(colorGood).Bold(true)

// tip is the box shown where the mouse rests.
type tip struct {
	x, y  int // the screen cell under the mouse
	lines []string
	win   int              // the window it is about
	diags []lsp.Diagnostic // what it shows
	fix   bool             // the server has a fix: a click applies it
}

// tipAt is the box for the diagnostics under screen cell (x, y) of window
// i (drawn in r), or nil when there are none.
func (m Model) tipAt(i int, r rect, x, y int) *tip {
	b := m.ses.wins[i].buf
	if b.lsp == nil || len(b.lsp.diags) == 0 {
		return nil
	}
	ed := b.ed
	v := ed.Viewport()
	line := v.Top + y - r.y
	if line >= ed.Buffer().LineCount() {
		return nil
	}
	var found []lsp.Diagnostic
	gutter := gutterWidth(ed.Buffer().LineCount())
	if x-r.x < gutter {
		found = diagnosticsOn(b.lsp.diags, line) // the line number, the ●
	} else {
		visual := x - r.x - gutter + v.Left
		end := ed.VisualColumn(line, ed.Buffer().LineLen(line))
		if visual > end {
			return nil // past the cell right after the text
		}
		col := ed.ColumnAt(line, visual)
		for _, d := range b.lsp.diags {
			if covers(d, line, col) {
				found = append(found, d)
			}
		}
	}
	if len(found) == 0 {
		return nil
	}
	lines, fix := diagnosticLines(found)
	if fix {
		lines = append(lines, styleFixButton.Render("▸ clique aqui para corrigir"))
	}
	box := strings.Split(styleHover.Render(strings.Join(lines, "\n")), "\n")
	return &tip{x: x, y: y, lines: box, win: i, diags: found, fix: fix}
}

// onTip reports screen cell (x, y) inside the tip's box.
func (m Model) onTip(t *tip, x, y int) bool {
	top, left := m.placement(max(m.height-1, 1), t.lines, t.y, t.x)
	w := 0
	for _, line := range t.lines {
		w = max(w, ansi.StringWidth(line))
	}
	return x >= left && x < left+w && y >= top && y < top+len(t.lines)
}

// applyFix asks the server for the code actions of the tip's diagnostics
// (from their line, which gets the cursor) and applies the fix when it is
// the only one; several open the list (see editReply).
func (m *Model) applyFix(t *tip) {
	m.focus(t.win)
	*m = m.synced()
	d := t.diags[0]
	m.moveCursor(editor.Position{Line: d.Start.Line, Column: d.Start.Col})
	diags := t.diags
	res := m.askCursor(func(ctx context.Context, c *lsp.Client, path string, p lsp.Pos) tea.Msg {
		actions, err := c.CodeActions(ctx, path, p, diags)
		return actionsMsg{actions: actions, client: c, err: err, apply: true}
	})
	if res.Message != "" {
		m.message, m.isError = res.Message, res.Err
	}
}

// covers reports d spanning (line, col); an empty range covers its start
// (where the underline shows one cell).
func covers(d lsp.Diagnostic, line, col int) bool {
	before := func(l1, c1, l2, c2 int) bool { return l1 < l2 || (l1 == l2 && c1 < c2) }
	if !before(d.Start.Line, d.Start.Col, d.End.Line, d.End.Col) {
		return line == d.Start.Line && col == d.Start.Col
	}
	return !before(line, col, d.Start.Line, d.Start.Col) && before(line, col, d.End.Line, d.End.Col)
}

// diagnosticsOn are the diagnostics touching line.
func diagnosticsOn(diags []lsp.Diagnostic, line int) []lsp.Diagnostic {
	var out []lsp.Diagnostic
	for _, d := range diags {
		if d.Start.Line <= line && line <= d.End.Line {
			out = append(out, d)
		}
	}
	return out
}

// diagnosticLines are the messages, worst first: "erro: expected ';'…",
// wrapped, with the server's extra lines (notes) below each; fix reports
// that the server has a fix for one of them ("(fix available)", from
// clangd).
func diagnosticLines(diags []lsp.Diagnostic) (lines []string, fix bool) {
	diags = slices.Clone(diags)
	slices.SortStableFunc(diags, func(a, b lsp.Diagnostic) int { return int(a.Severity) - int(b.Severity) })
	for _, d := range diags {
		fix = fix || strings.Contains(d.Message, "fix available")
		label := severityStyles[d.Severity].Render(severityNames[d.Severity] + ":")
		text := strings.Split(ansi.Wrap(d.Message, 70, ""), "\n")
		lines = append(lines, label+" "+text[0])
		for _, l := range text[1:] {
			lines = append(lines, "  "+l)
		}
	}
	if len(lines) > 14 {
		lines = append(lines[:13], "…")
	}
	return lines, fix
}

// place draws box over rows below screen row (above when there's no
// room), from column col.
func (m Model) place(rows, box []string, row, col int) {
	top, left := m.placement(len(rows), box, row, col)
	for i, line := range box {
		if r := top + i; r >= 0 && r < len(rows) {
			rows[r] = overlay(rows[r], left, line, m.width)
		}
	}
}

// placement is where place puts box over n rows: its top row and left
// column.
func (m Model) placement(n int, box []string, row, col int) (top, left int) {
	top = row + 1
	if top+len(box) > n && row-len(box) >= 0 {
		top = row - len(box)
	}
	w := 0
	for _, line := range box {
		w = max(w, ansi.StringWidth(line))
	}
	return top, max(min(col, m.width-w), 0)
}
