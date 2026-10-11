package vim

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// More Normal-mode commands: J and gJ, r, ~, Ctrl-a and Ctrl-x, marks, and
// the commands that scroll the screen (Ctrl-d/u/f/b/e/y, zz/zt/zb).

// join is J (gJ when raw): count lines (at least two) become one. J drops
// the next line's indentation and puts one space between them (none before
// a ")" or after a line that already ends in a blank); gJ joins as is. The
// cursor lands where the last two lines met.
func (c *Controller) join(count int, raw bool) Result {
	buf := c.ed.Buffer()
	line := c.ed.Cursor().Line
	n := max(count, 2) - 1
	if line+1 >= buf.LineCount() {
		return Result{}
	}
	n = min(n, buf.LineCount()-1-line)
	c.ed.BeginGroup()
	var at editor.Position
	for i := 0; i < n; i++ {
		end := editor.Position{Line: line, Column: buf.LineLen(line)}
		nextText := buf.Line(line + 1)
		if raw {
			c.ed.Delete(editor.Range{Start: end, End: editor.Position{Line: line + 1}})
			at = end
			continue
		}
		trimmed := strings.TrimLeft(nextText, " \t")
		cut := editor.Position{Line: line + 1, Column: len([]rune(nextText)) - len([]rune(trimmed))}
		sep := " "
		cur := buf.Line(line)
		if trimmed == "" || cur == "" || strings.HasSuffix(cur, " ") || strings.HasSuffix(cur, "\t") || strings.HasPrefix(trimmed, ")") {
			sep = ""
		}
		c.ed.Replace(editor.Range{Start: end, End: cut}, sep)
		at = end
		if sep == "" && end.Column > 0 {
			at.Column--
		}
	}
	c.ed.EndGroup()
	c.ed.MoveCursor(at)
	c.clampNormal()
	return Result{}
}

// replaceChars is r{char}: the count characters from the cursor become
// char; the cursor stays on the last one. Nothing changes when the line is
// too short, like Vim.
func (c *Controller) replaceChars(char string, count int) Result {
	cur := c.ed.Cursor()
	if cur.Column+count > c.ed.Buffer().LineLen(cur.Line) {
		return Result{}
	}
	end := editor.Position{Line: cur.Line, Column: cur.Column + count}
	c.ed.Replace(editor.Range{Start: cur, End: end}, strings.Repeat(char, count))
	c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: end.Column - 1})
	return Result{}
}

// toggleCase is ~: the case of count characters flips and the cursor moves
// past them (staying on the line).
func (c *Controller) toggleCase(count int) Result {
	cur := c.ed.Cursor()
	n := c.ed.Buffer().LineLen(cur.Line)
	if n == 0 {
		return Result{}
	}
	end := editor.Position{Line: cur.Line, Column: min(cur.Column+count, n)}
	c.recase(opToggle, editor.Range{Start: cur, End: end})
	c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: min(end.Column, n-1)})
	return Result{}
}

// addNumber is Ctrl-a (Ctrl-x with a negative delta): the number under or
// after the cursor on the line changes by delta; the cursor ends on its
// last digit.
func (c *Controller) addNumber(delta int) Result {
	cur := c.ed.Cursor()
	line := []rune(c.ed.Buffer().Line(cur.Line))
	start := cur.Column
	// Back to the start of a number the cursor is in.
	for start > 0 && start < len(line) && unicode.IsDigit(line[start]) && unicode.IsDigit(line[start-1]) {
		start--
	}
	for start < len(line) && !unicode.IsDigit(line[start]) {
		start++
	}
	if start >= len(line) {
		return Result{}
	}
	end := start
	for end < len(line) && unicode.IsDigit(line[end]) {
		end++
	}
	if start > 0 && line[start-1] == '-' {
		start--
	}
	n, err := strconv.ParseInt(string(line[start:end]), 10, 64)
	if err != nil {
		return Result{Message: "número grande demais", Err: true}
	}
	text := strconv.FormatInt(n+int64(delta), 10)
	from := editor.Position{Line: cur.Line, Column: start}
	c.ed.Replace(editor.Range{Start: from, End: editor.Position{Line: cur.Line, Column: end}}, text)
	c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: start + len([]rune(text)) - 1})
	return Result{}
}

// scroll moves the screen by lines (down when positive) and the cursor
// with it: Ctrl-d/u (half a screen), Ctrl-f/b and PgDn/PgUp (a screen).
func (c *Controller) scroll(lines int) Result {
	buf := c.ed.Buffer()
	cur := c.ed.Cursor()
	line := min(max(cur.Line+lines, 0), buf.LineCount()-1)
	if line == cur.Line {
		return Result{}
	}
	c.ed.ScrollBy(lines)
	c.ed.MoveCursor(editor.Position{Line: line, Column: indentWidth(buf, line)})
	return Result{}
}

// scrollView is Ctrl-e/Ctrl-y: the screen moves lines (down when positive)
// and the cursor only as much as it takes to stay on it.
func (c *Controller) scrollView(lines int) Result {
	c.ed.ScrollBy(lines)
	v := c.ed.Viewport()
	if v.Height <= 0 {
		return Result{}
	}
	m := scrollMargin(v)
	cur := c.ed.Cursor()
	line := cur.Line
	if top := v.Top + m; v.Top > 0 && line < top {
		line = top
	}
	if bottom := v.Top + v.Height - 1 - m; line > bottom {
		line = bottom
	}
	line = min(max(line, 0), c.ed.Buffer().LineCount()-1)
	if line != cur.Line {
		c.ed.MoveCursor(editor.Position{Line: line, Column: cur.Column})
		c.clampNormal()
	}
	return Result{}
}

// place is zz, zt and zb: the cursor's line goes to the middle, top or
// bottom of the screen.
func (c *Controller) place(where byte) Result {
	v := c.ed.Viewport()
	if v.Height <= 0 {
		return Result{}
	}
	line := c.ed.Cursor().Line
	top := v.Top
	switch where {
	case 'z':
		top = line - v.Height/2
	case 't':
		top = line - scrollMargin(v)
	case 'b':
		top = line - v.Height + 1 + scrollMargin(v)
	}
	c.ed.ScrollBy(max(top, 0) - v.Top)
	return Result{}
}

// setMark is m{a-z}: remembers the cursor in the current file.
func (c *Controller) setMark(name rune) Result {
	if !unicode.IsLower(name) {
		return Result{Message: "E191: marca inválida (use a-z)", Err: true}
	}
	if c.marks == nil {
		c.marks = map[*editor.Editor]map[rune]editor.Position{}
	}
	if c.marks[c.ed] == nil {
		c.marks[c.ed] = map[rune]editor.Position{}
	}
	c.marks[c.ed][name] = c.ed.Cursor()
	return Result{}
}

// markMotion is '{a-z} (to the mark's line, linewise) and `{a-z} (to the
// exact place). Marks don't move with edits, so one may end up past the
// text; it is clamped.
func (c *Controller) markMotion(exact bool, name rune) Motion {
	return func(ed *editor.Editor, _ int, _ bool) Target {
		p, ok := c.marks[ed][name]
		if !ok {
			return Target{Failed: true}
		}
		p = ed.Buffer().Clamp(p)
		if exact {
			return Target{Pos: p}
		}
		return toLine(ed, p.Line)
	}
}

// deleteWordBefore is Ctrl-w in Insert mode: the word (and the blanks)
// before the cursor go; at the start of a line, the line break.
func (c *Controller) deleteWordBefore() {
	cur := c.ed.Cursor()
	if cur.Column == 0 {
		c.ed.DeleteBackward()
		return
	}
	buf := c.ed.Buffer()
	p := cur
	for p.Column > 0 {
		q := editor.Position{Line: p.Line, Column: p.Column - 1}
		if classAt(buf, q) != blankClass {
			break
		}
		p = q
	}
	if p.Column > 0 {
		cls := classAt(buf, editor.Position{Line: p.Line, Column: p.Column - 1})
		for p.Column > 0 && classAt(buf, editor.Position{Line: p.Line, Column: p.Column - 1}) == cls {
			p.Column--
		}
	}
	c.ed.Delete(editor.Range{Start: p, End: cur})
}

// deleteToLineStart is Ctrl-u in Insert mode: what was typed before the
// cursor on the line goes, down to the indentation (then to column 0).
func (c *Controller) deleteToLineStart() {
	cur := c.ed.Cursor()
	if cur.Column == 0 {
		c.ed.DeleteBackward()
		return
	}
	start := editor.Position{Line: cur.Line, Column: indentWidth(c.ed.Buffer(), cur.Line)}
	if start.Column >= cur.Column {
		start.Column = 0
	}
	c.ed.Delete(editor.Range{Start: start, End: cur})
}
