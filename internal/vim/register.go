package vim

import (
	"slices"
	"strings"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// Register holds yanked or deleted text.
type Register struct {
	Text string
	// Linewise text is whole lines (yy, dd, V): p puts it below the
	// cursor's line instead of after the cursor.
	Linewise bool
}

// unnamed is the register d, c, x, y and p use. Named registers ("a…"z)
// only need the parser to pick another key of Controller.regs.
const unnamed = '"'

// put pastes the register count times after the cursor (p) or before it
// (P), as one undo step.
func (c *Controller) put(before bool, count int) Result {
	reg, ok := c.regs[unnamed]
	if !ok {
		return Result{Message: "E353: register vazio (y copia, d apaga)", Err: true}
	}
	cur := c.ed.Cursor()
	if reg.Linewise {
		line := cur.Line + 1
		if before {
			line = cur.Line
		}
		c.putLines(line, strings.Join(slices.Repeat([]string{reg.Text}, count), "\n"))
		return Result{}
	}
	if !before && c.ed.Buffer().LineLen(cur.Line) > 0 {
		cur.Column++
	}
	c.putChars(cur, strings.Repeat(reg.Text, count))
	return Result{}
}

// putLines inserts text as whole lines above line (after the last line when
// line is past it) and lands on the first non-blank of the first one.
func (c *Controller) putLines(line int, text string) {
	buf := c.ed.Buffer()
	at := editor.Position{Line: line}
	if line < buf.LineCount() {
		text += "\n"
	} else {
		line = buf.LineCount()
		at = editor.Position{Line: line - 1, Column: buf.LineLen(line - 1)}
		text = "\n" + text
	}
	c.ed.Replace(editor.Range{Start: at, End: at}, text)
	c.ed.MoveCursor(editor.Position{Line: line, Column: indentWidth(buf, line)})
	c.clampNormal()
}

// putChars inserts text at p and lands like Vim: on the last character put,
// or on the first one when the text spans lines.
func (c *Controller) putChars(p editor.Position, text string) {
	c.ed.Replace(editor.Range{Start: p, End: p}, text)
	if !strings.Contains(text, "\n") {
		end := c.ed.Cursor()
		p = editor.Position{Line: end.Line, Column: end.Column - 1}
	}
	c.ed.MoveCursor(p)
	c.clampNormal()
}

// replaceSelection puts the register over the selection (Visual p), as one
// undo step. Like Vim, the replaced text goes to the register.
func (c *Controller) replaceSelection(count int) Result {
	reg, ok := c.regs[unnamed]
	if !ok {
		c.exitVisual()
		return Result{Message: "E353: register vazio (y copia, d apaga)", Err: true}
	}
	r, linewise, _ := c.ed.SelectedRange()
	whole := linewise && r.Start.Line == 0 && r.End.Line == c.ed.Buffer().LineCount()-1
	text := strings.Repeat(reg.Text, count)
	if reg.Linewise {
		text = strings.Join(slices.Repeat([]string{reg.Text}, count), "\n")
	}

	c.ed.BeginGroup()
	defer c.ed.EndGroup()
	c.operate(opDelete)
	switch {
	case whole:
		// Nothing left but an empty line: the text takes its place.
		c.ed.Replace(editor.Range{}, text)
		c.ed.MoveCursor(editor.Position{Column: indentWidth(c.ed.Buffer(), 0)})
		c.clampNormal()
	case linewise:
		c.putLines(r.Start.Line, text)
	case reg.Linewise:
		// Lines into a run of characters split its line, like Vim.
		c.putChars(r.Start, "\n"+text+"\n")
		c.ed.MoveCursor(editor.Position{Line: r.Start.Line + 1, Column: indentWidth(c.ed.Buffer(), r.Start.Line+1)})
		c.clampNormal()
	default:
		c.putChars(r.Start, text)
	}
	return Result{}
}
