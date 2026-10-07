package editorview

import (
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/lsp"
)

// Diagnostics on demand: resting the mouse on underlined code (or on the
// ● in the gutter) shows the full messages in a box; K shows the ones on
// the cursor's line above the server's hover text.

// tip is the box shown where the mouse rests.
type tip struct {
	x, y  int // the screen cell under the mouse
	lines []string
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
	return &tip{x: x, y: y, lines: strings.Split(styleHover.Render(strings.Join(diagnosticLines(found), "\n")), "\n")}
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
// wrapped, with the server's extra lines (notes) below each, and how to
// apply the fix when the server has one ("(fix available)", from clangd).
func diagnosticLines(diags []lsp.Diagnostic) []string {
	diags = slices.Clone(diags)
	slices.SortStableFunc(diags, func(a, b lsp.Diagnostic) int { return int(a.Severity) - int(b.Severity) })
	var lines []string
	fix := false
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
	if fix {
		lines = append(lines, styleStatus.Render("com o cursor nesta linha, gra aplica a correção"))
	}
	return lines
}

// place draws box over rows below screen row (above when there's no
// room), from column col.
func (m Model) place(rows, box []string, row, col int) {
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
