package vim

import "github.com/nvizble/Lightyear42/internal/editor"

// Operator acts on the text covered by a motion (d now; y and c come with
// registers and the next phases).
type Operator int

// Operators.
const (
	opNone Operator = iota
	opDelete
)

// operators maps keys to operators.
var operators = map[string]Operator{"d": opDelete}

// apply runs op from the cursor to target t.
func (c *Controller) apply(op Operator, t Target) {
	if op != opDelete {
		return
	}
	from := c.ed.Cursor()
	if t.Linewise {
		c.deleteLines(from.Line, t.Pos.Line)
		return
	}
	start, end := from, t.Pos
	if end.Before(start) {
		start, end = end, start
	}
	if t.Inclusive {
		end.Column++
	}
	c.ed.Delete(editor.Range{Start: start, End: end})
	c.ed.MoveCursor(start)
	c.clampNormal()
}

// deleteLines removes lines first..last (in any order) with their line
// breaks and lands on the first non-blank character of the line that takes
// their place.
func (c *Controller) deleteLines(first, last int) {
	buf := c.ed.Buffer()
	if last < first {
		first, last = last, first
	}
	r := editor.Range{Start: editor.Position{Line: first}, End: editor.Position{Line: last + 1}}
	switch {
	case last+1 < buf.LineCount():
		// Lines and their trailing newline.
	case first > 0:
		// Up to the last line: take the newline before them instead.
		r = editor.Range{Start: editor.Position{Line: first - 1, Column: buf.LineLen(first - 1)}, End: editor.Position{Line: last, Column: buf.LineLen(last)}}
	default:
		// The whole document: leave one empty line.
		r.End = editor.Position{Line: last, Column: buf.LineLen(last)}
	}
	c.ed.Delete(r)
	target := min(first, buf.LineCount()-1)
	c.ed.MoveCursor(editor.Position{Line: target, Column: indentWidth(buf, target)})
	c.clampNormal()
}
