package vim

import "github.com/nvizble/Lightyear42/internal/editor"

// Operator acts on the text covered by a motion or a selection.
type Operator int

// Operators.
const (
	opNone Operator = iota
	opDelete
	opYank
	opChange
)

// operators maps keys to operators.
var operators = map[string]Operator{"d": opDelete, "y": opYank, "c": opChange}

// apply runs op from the cursor to target t.
func (c *Controller) apply(op Operator, t Target) {
	from := c.ed.Cursor()
	if t.Linewise {
		c.applyLines(op, from.Line, t.Pos.Line)
		return
	}
	start, end := from, t.Pos
	if end.Before(start) {
		start, end = end, start
	}
	if t.Inclusive {
		end.Column++
	}
	c.applyRange(op, editor.Range{Start: start, End: end})
}

// applyRange runs op on the characters in r; the text goes to the register.
func (c *Controller) applyRange(op Operator, r editor.Range) {
	if text := c.ed.Buffer().Slice(r); text != "" {
		c.regs[unnamed] = Register{Text: text}
	}
	switch op {
	case opYank:
		c.ed.MoveCursor(r.Start)
		c.clampNormal()
	case opDelete:
		c.deleteRange(r)
	case opChange:
		c.ed.BeginGroup() // the deletion and what gets typed undo together
		c.ed.MoveCursor(r.Start)
		c.ed.Delete(r)
		c.mode = Insert
	}
}

// applyLines runs op on lines first..last (in any order); they go to the
// register as whole lines.
func (c *Controller) applyLines(op Operator, first, last int) {
	if last < first {
		first, last = last, first
	}
	buf := c.ed.Buffer()
	c.regs[unnamed] = Register{Text: buf.Slice(editor.Range{Start: editor.Position{Line: first}, End: editor.Position{Line: last, Column: buf.LineLen(last)}}), Linewise: true}
	switch op {
	case opYank:
		if cur := c.ed.Cursor(); cur.Line != first {
			c.ed.MoveCursor(editor.Position{Line: first, Column: cur.Column})
			c.clampNormal()
		}
	case opDelete:
		c.deleteLines(first, last)
	case opChange:
		// The lines become one, empty but for the first one's indentation.
		indent := editor.Position{Line: first, Column: indentWidth(buf, first)}
		c.ed.BeginGroup()
		if cur := c.ed.Cursor(); cur.Line != first {
			c.ed.MoveCursor(editor.Position{Line: first, Column: cur.Column})
		}
		c.ed.Delete(editor.Range{Start: indent, End: editor.Position{Line: last, Column: buf.LineLen(last)}})
		c.ed.MoveCursor(indent)
		c.mode = Insert
	}
}

// deleteRange removes r and leaves the cursor at its start. The cursor goes
// there first, so undo brings it back to the start of the change, like Vim.
func (c *Controller) deleteRange(r editor.Range) {
	c.ed.MoveCursor(r.Start)
	c.ed.Delete(r)
	c.clampNormal()
}

// deleteLines removes lines first..last with their line breaks and lands
// on the first non-blank character of the line that takes their place.
func (c *Controller) deleteLines(first, last int) {
	buf := c.ed.Buffer()
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
	if cur := c.ed.Cursor(); cur.Line != first {
		// Undo brings the cursor back to the first line, like Vim.
		c.ed.MoveCursor(editor.Position{Line: first, Column: cur.Column})
	}
	c.ed.Delete(r)
	target := min(first, buf.LineCount()-1)
	c.ed.MoveCursor(editor.Position{Line: target, Column: indentWidth(buf, target)})
	c.clampNormal()
}
