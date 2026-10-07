package vim

import (
	"unicode"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// Target is where a motion lands and how an operator treats the text up to
// it, following Vim: exclusive or inclusive of the last character, or
// linewise (whole lines).
type Target struct {
	Pos       editor.Position
	Linewise  bool
	Inclusive bool
	// Vertical, when set, moves the cursor that many lines (up when
	// negative) keeping its column, instead of jumping to Pos.
	Vertical int
	// Failed is set when the motion can't move (h at column 0, j on the last
	// line); an operator then does nothing, like Vim.
	Failed bool
}

// Motion resolves a target from the cursor. count is 0 when none was typed;
// op is set when an operator will act on the target.
type Motion func(ed *editor.Editor, count int, op bool) Target

// motions are the keys that move the cursor (and feed operators).
var motions = map[string]Motion{
	"h": left, "left": left,
	"l": right, "right": right,
	"j": down, "down": down,
	"k": up, "up": up,
	"w": wordForward, "b": wordBackward, "e": wordEnd,
	"0": lineStart, "^": firstNonBlank, "$": lineEnd,
	"gg": firstLine, "G": lastLine,
}

func times(count int) int { return max(count, 1) }

func left(ed *editor.Editor, count int, _ bool) Target {
	cur := ed.Cursor()
	if cur.Column == 0 {
		return Target{Failed: true}
	}
	return Target{Pos: editor.Position{Line: cur.Line, Column: max(cur.Column-times(count), 0)}}
}

func right(ed *editor.Editor, count int, op bool) Target {
	cur, n := ed.Cursor(), ed.Buffer().LineLen(ed.Cursor().Line)
	// Moving stops on the last character; an operator may reach the end
	// (dl deletes the last character).
	last := max(n-1, 0)
	if op {
		last = n
	}
	if cur.Column >= last {
		return Target{Failed: true}
	}
	return Target{Pos: editor.Position{Line: cur.Line, Column: min(cur.Column+times(count), last)}}
}

func down(ed *editor.Editor, count int, _ bool) Target {
	cur := ed.Cursor()
	line := min(cur.Line+times(count), ed.Buffer().LineCount()-1)
	if line == cur.Line {
		return Target{Failed: true}
	}
	return Target{Pos: editor.Position{Line: line}, Linewise: true, Vertical: line - cur.Line}
}

func up(ed *editor.Editor, count int, _ bool) Target {
	cur := ed.Cursor()
	line := max(cur.Line-times(count), 0)
	if line == cur.Line {
		return Target{Failed: true}
	}
	return Target{Pos: editor.Position{Line: line}, Linewise: true, Vertical: line - cur.Line}
}

func lineStart(ed *editor.Editor, _ int, _ bool) Target {
	return Target{Pos: editor.Position{Line: ed.Cursor().Line}}
}

func firstNonBlank(ed *editor.Editor, _ int, _ bool) Target {
	line := ed.Cursor().Line
	return Target{Pos: editor.Position{Line: line, Column: indentWidth(ed.Buffer(), line)}}
}

// lineEnd ($) goes to the last character; a count goes count-1 lines down.
func lineEnd(ed *editor.Editor, count int, _ bool) Target {
	buf := ed.Buffer()
	line := min(ed.Cursor().Line+times(count)-1, buf.LineCount()-1)
	return Target{Pos: editor.Position{Line: line, Column: max(buf.LineLen(line)-1, 0)}, Inclusive: true}
}

// firstLine (gg) and lastLine (G) go to line count when one was typed.
func firstLine(ed *editor.Editor, count int, _ bool) Target {
	return toLine(ed, times(count)-1)
}

func lastLine(ed *editor.Editor, count int, _ bool) Target {
	if count == 0 {
		return toLine(ed, ed.Buffer().LineCount()-1)
	}
	return toLine(ed, count-1)
}

func toLine(ed *editor.Editor, line int) Target {
	buf := ed.Buffer()
	line = min(max(line, 0), buf.LineCount()-1)
	return Target{Pos: editor.Position{Line: line, Column: indentWidth(buf, line)}, Linewise: true}
}

func indentWidth(buf *editor.Buffer, line int) int {
	return len([]rune(buf.Indent(line)))
}

// Word motions. Like Vim, a word is a run of letters, digits and "_", or a
// run of other non-blank characters; an empty line also counts as a word.

const (
	blankClass = iota
	punctClass
	wordClass
)

func classAt(buf *editor.Buffer, p editor.Position) int {
	r, ok := buf.RuneAt(p)
	switch {
	case !ok || r == ' ' || r == '\t':
		return blankClass // the end of a line (newline) is blank too
	case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
		return wordClass
	}
	return punctClass
}

// next and prev step one position, through the newline at each line end.
func next(buf *editor.Buffer, p editor.Position) (editor.Position, bool) {
	if p.Column < buf.LineLen(p.Line) {
		return editor.Position{Line: p.Line, Column: p.Column + 1}, true
	}
	if p.Line+1 < buf.LineCount() {
		return editor.Position{Line: p.Line + 1}, true
	}
	return p, false
}

func prev(buf *editor.Buffer, p editor.Position) (editor.Position, bool) {
	if p.Column > 0 {
		return editor.Position{Line: p.Line, Column: p.Column - 1}, true
	}
	if p.Line > 0 {
		return editor.Position{Line: p.Line - 1, Column: buf.LineLen(p.Line - 1)}, true
	}
	return p, false
}

func emptyLine(buf *editor.Buffer, p editor.Position) bool {
	return p.Column == 0 && buf.LineLen(p.Line) == 0
}

// wordForward (w) goes to the start of the next word. With an operator, the
// last word of a line ends at the end of that line ("dw" never joins lines).
func wordForward(ed *editor.Editor, count int, op bool) Target {
	buf := ed.Buffer()
	p, stepStart := ed.Cursor(), ed.Cursor()
	for i := 0; i < times(count); i++ {
		stepStart = p
		p = wordStep(buf, p)
	}
	if op && p.Line > stepStart.Line {
		p = editor.Position{Line: stepStart.Line, Column: buf.LineLen(stepStart.Line)}
	}
	return Target{Pos: p}
}

func wordStep(buf *editor.Buffer, p editor.Position) editor.Position {
	q, ok := p, true
	if cls := classAt(buf, p); cls != blankClass {
		for ok && q.Line == p.Line && classAt(buf, q) == cls {
			q, ok = next(buf, q)
		}
	}
	for ok && classAt(buf, q) == blankClass {
		if emptyLine(buf, q) && q != p {
			break
		}
		var n editor.Position
		if n, ok = next(buf, q); ok {
			q = n
		}
	}
	return q
}

// wordBackward (b) goes to the start of the current or previous word.
func wordBackward(ed *editor.Editor, count int, _ bool) Target {
	buf, p := ed.Buffer(), ed.Cursor()
	for i := 0; i < times(count); i++ {
		p = backStep(buf, p)
	}
	return Target{Pos: p}
}

func backStep(buf *editor.Buffer, p editor.Position) editor.Position {
	q, ok := prev(buf, p)
	if !ok {
		return p
	}
	for classAt(buf, q) == blankClass {
		if emptyLine(buf, q) {
			return q
		}
		n, ok := prev(buf, q)
		if !ok {
			return q
		}
		q = n
	}
	cls := classAt(buf, q)
	for {
		n, ok := prev(buf, q)
		if !ok || n.Line != q.Line || classAt(buf, n) != cls {
			return q
		}
		q = n
	}
}

// wordEnd (e) goes to the end of the current or next word, inclusive.
func wordEnd(ed *editor.Editor, count int, _ bool) Target {
	buf, p := ed.Buffer(), ed.Cursor()
	for i := 0; i < times(count); i++ {
		p = endStep(buf, p)
	}
	return Target{Pos: p, Inclusive: true}
}

func endStep(buf *editor.Buffer, p editor.Position) editor.Position {
	q, ok := next(buf, p)
	if !ok {
		return p
	}
	for classAt(buf, q) == blankClass {
		n, ok := next(buf, q)
		if !ok {
			return q
		}
		q = n
	}
	cls := classAt(buf, q)
	for {
		n, ok := next(buf, q)
		if !ok || n.Line != q.Line || classAt(buf, n) != cls {
			return q
		}
		q = n
	}
}
