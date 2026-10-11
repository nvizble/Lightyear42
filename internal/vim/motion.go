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
	"W": bigWordForward, "B": bigWordBackward, "E": bigWordEnd,
	"ge": wordEndBackward, "gE": bigWordEndBackward,
	"0": lineStart, "^": firstNonBlank, "$": lineEnd, "g_": lastNonBlank,
	"home": lineStart, "end": lineEnd, "|": column,
	"gg": firstLine, "G": lastLine, "ctrl+home": firstLine, "ctrl+end": lastLine,
	"+": nextLineStart, "enter": nextLineStart, "-": prevLineStart, "_": lineStartDown,
	"}": paragraphForward, "{": paragraphBackward, "%": matchOrPercent,
	"H": screenTop, "M": screenMiddle, "L": screenBottom,
	"backspace": left, " ": right,
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

// classifier tells word characters, other characters and blanks apart; w,
// b and e use classAt, W, B and E (WORDs: anything but blanks) bigClassAt.
type classifier func(buf *editor.Buffer, p editor.Position) int

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

func bigClassAt(buf *editor.Buffer, p editor.Position) int {
	if classAt(buf, p) == blankClass {
		return blankClass
	}
	return wordClass
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
// W is the same over WORDs.
func wordForward(ed *editor.Editor, count int, op bool) Target {
	return wordForwardBy(ed, count, op, classAt)
}

func bigWordForward(ed *editor.Editor, count int, op bool) Target {
	return wordForwardBy(ed, count, op, bigClassAt)
}

func wordForwardBy(ed *editor.Editor, count int, op bool, classAt classifier) Target {
	buf := ed.Buffer()
	p, stepStart := ed.Cursor(), ed.Cursor()
	for i := 0; i < times(count); i++ {
		stepStart = p
		p = wordStep(buf, p, classAt)
	}
	if op && p.Line > stepStart.Line {
		p = editor.Position{Line: stepStart.Line, Column: buf.LineLen(stepStart.Line)}
	}
	return Target{Pos: p}
}

func wordStep(buf *editor.Buffer, p editor.Position, classAt classifier) editor.Position {
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

// wordBackward (b) goes to the start of the current or previous word; B
// over WORDs.
func wordBackward(ed *editor.Editor, count int, op bool) Target {
	return wordBackwardBy(ed, count, op, classAt)
}

func bigWordBackward(ed *editor.Editor, count int, op bool) Target {
	return wordBackwardBy(ed, count, op, bigClassAt)
}

func wordBackwardBy(ed *editor.Editor, count int, _ bool, classAt classifier) Target {
	buf, p := ed.Buffer(), ed.Cursor()
	for i := 0; i < times(count); i++ {
		p = backStep(buf, p, classAt)
	}
	return Target{Pos: p}
}

func backStep(buf *editor.Buffer, p editor.Position, classAt classifier) editor.Position {
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

// changeWord is "cw": like Vim, on a word it changes only up to the word's
// end (not the blanks after it), as "ce" but counting the current word.
func changeWord(ed *editor.Editor, count int, op bool) Target {
	buf, p := ed.Buffer(), ed.Cursor()
	cls := classAt(buf, p)
	if cls == blankClass {
		return wordForward(ed, count, op)
	}
	for {
		n, ok := next(buf, p)
		if !ok || n.Line != p.Line || classAt(buf, n) != cls {
			break
		}
		p = n
	}
	for i := 1; i < times(count); i++ {
		p = endStep(buf, p, classAt)
	}
	return Target{Pos: p, Inclusive: true}
}

// wordEnd (e) goes to the end of the current or next word, inclusive; E
// over WORDs.
func wordEnd(ed *editor.Editor, count int, op bool) Target {
	return wordEndBy(ed, count, op, classAt)
}

func bigWordEnd(ed *editor.Editor, count int, op bool) Target {
	return wordEndBy(ed, count, op, bigClassAt)
}

func wordEndBy(ed *editor.Editor, count int, _ bool, classAt classifier) Target {
	buf, p := ed.Buffer(), ed.Cursor()
	for i := 0; i < times(count); i++ {
		p = endStep(buf, p, classAt)
	}
	return Target{Pos: p, Inclusive: true}
}

func endStep(buf *editor.Buffer, p editor.Position, classAt classifier) editor.Position {
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

// findMotion is f, F, t and T: to the count-th char on the line, after the
// cursor (f, t: inclusive) or before it (F, T); t and T stop one short.
// Repeated with ; a t doesn't get stuck on the char right next to it.
func findMotion(kind byte, char rune, repeat bool) Motion {
	return func(ed *editor.Editor, count int, _ bool) Target {
		cur := ed.Cursor()
		line := []rune(ed.Buffer().Line(cur.Line))
		step := 1
		if kind == 'F' || kind == 'T' {
			step = -1
		}
		col := cur.Column
		if repeat && (kind == 't' || kind == 'T') && col+step >= 0 && col+step < len(line) && line[col+step] == char {
			col += step
		}
		for n := 0; n < times(count); n++ {
			for col += step; col >= 0 && col < len(line) && line[col] != char; col += step {
			}
			if col < 0 || col >= len(line) {
				return Target{Failed: true}
			}
		}
		switch kind {
		case 't':
			col--
		case 'T':
			col++
		}
		return Target{Pos: editor.Position{Line: cur.Line, Column: col}, Inclusive: step > 0}
	}
}

// wordEndBackward (ge) goes to the end of the previous word, inclusive; gE
// over WORDs.
func wordEndBackward(ed *editor.Editor, count int, _ bool) Target {
	return backEndBy(ed, count, classAt)
}

func bigWordEndBackward(ed *editor.Editor, count int, _ bool) Target {
	return backEndBy(ed, count, bigClassAt)
}

func backEndBy(ed *editor.Editor, count int, classAt classifier) Target {
	buf, p := ed.Buffer(), ed.Cursor()
	for i := 0; i < times(count); i++ {
		p = backEndStep(buf, p, classAt)
	}
	if p == ed.Cursor() {
		return Target{Failed: true}
	}
	return Target{Pos: p, Inclusive: true}
}

func backEndStep(buf *editor.Buffer, p editor.Position, classAt classifier) editor.Position {
	q := p
	if cls := classAt(buf, q); cls != blankClass {
		// Out of the current word first.
		for {
			n, ok := prev(buf, q)
			if !ok || n.Line != q.Line || classAt(buf, n) != cls {
				break
			}
			q = n
		}
	}
	n, ok := prev(buf, q)
	if !ok {
		return p
	}
	q = n
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
	return q
}

// lastNonBlank (g_) goes to the last non-blank character, count-1 lines down.
func lastNonBlank(ed *editor.Editor, count int, _ bool) Target {
	buf := ed.Buffer()
	line := min(ed.Cursor().Line+times(count)-1, buf.LineCount()-1)
	runes := []rune(buf.Line(line))
	col := len(runes) - 1
	for col > 0 && (runes[col] == ' ' || runes[col] == '\t') {
		col--
	}
	return Target{Pos: editor.Position{Line: line, Column: max(col, 0)}, Inclusive: true}
}

// column (|) goes to screen column count (tabs expanded) of the line.
func column(ed *editor.Editor, count int, _ bool) Target {
	line := ed.Cursor().Line
	col := min(ed.ColumnAt(line, times(count)-1), max(ed.Buffer().LineLen(line)-1, 0))
	return Target{Pos: editor.Position{Line: line, Column: col}}
}

// nextLineStart (+, Enter) and prevLineStart (-) go count lines down or up,
// to the first non-blank character; lineStartDown (_) count-1 lines down.
func nextLineStart(ed *editor.Editor, count int, _ bool) Target {
	line := ed.Cursor().Line + times(count)
	if line >= ed.Buffer().LineCount() {
		return Target{Failed: true}
	}
	return toLine(ed, line)
}

func prevLineStart(ed *editor.Editor, count int, _ bool) Target {
	line := ed.Cursor().Line - times(count)
	if line < 0 {
		return Target{Failed: true}
	}
	return toLine(ed, line)
}

func lineStartDown(ed *editor.Editor, count int, _ bool) Target {
	return toLine(ed, ed.Cursor().Line+times(count)-1)
}

// paragraphForward (}) goes to the next empty line (or the end of the
// document); paragraphBackward ({) to the previous one (or the start).
func paragraphForward(ed *editor.Editor, count int, _ bool) Target {
	buf := ed.Buffer()
	line := ed.Cursor().Line
	for i := 0; i < times(count); i++ {
		for line < buf.LineCount()-1 && buf.LineLen(line) == 0 {
			line++
		}
		for line < buf.LineCount()-1 && buf.LineLen(line) > 0 {
			line++
		}
	}
	if line == buf.LineCount()-1 && buf.LineLen(line) > 0 {
		return Target{Pos: editor.Position{Line: line, Column: buf.LineLen(line)}}
	}
	return Target{Pos: editor.Position{Line: line}}
}

func paragraphBackward(ed *editor.Editor, count int, _ bool) Target {
	buf := ed.Buffer()
	line := ed.Cursor().Line
	for i := 0; i < times(count); i++ {
		for line > 0 && buf.LineLen(line) == 0 {
			line--
		}
		for line > 0 && buf.LineLen(line) > 0 {
			line--
		}
	}
	return Target{Pos: editor.Position{Line: line}}
}

// matchOrPercent is %: to the bracket matching the one at or after the
// cursor on the line, inclusive; with a count, to that percent of the file.
func matchOrPercent(ed *editor.Editor, count int, _ bool) Target {
	buf := ed.Buffer()
	if count > 0 {
		if count > 100 {
			return Target{Failed: true}
		}
		return toLine(ed, (count*buf.LineCount()+99)/100-1)
	}
	cur := ed.Cursor()
	line := []rune(buf.Line(cur.Line))
	pairs := map[rune]rune{'(': ')', '[': ']', '{': '}', ')': '(', ']': '[', '}': '{'}
	col := cur.Column
	for col < len(line) && pairs[line[col]] == 0 {
		col++
	}
	if col >= len(line) {
		return Target{Failed: true}
	}
	open, want := line[col], pairs[line[col]]
	forward := open == '(' || open == '[' || open == '{'
	step := next
	if !forward {
		step = prev
	}
	depth := 0
	for p, ok := (editor.Position{Line: cur.Line, Column: col}), true; ok; p, ok = step(buf, p) {
		switch r, _ := buf.RuneAt(p); r {
		case open:
			depth++
		case want:
			if depth--; depth == 0 {
				return Target{Pos: p, Inclusive: true}
			}
		}
	}
	return Target{Failed: true}
}

// screenTop (H), screenMiddle (M) and screenBottom (L) go to the lines on
// screen, to the first non-blank character; H and L stay off the scroll
// margin so the screen doesn't move, unless at the document's edge.
func screenTop(ed *editor.Editor, count int, _ bool) Target {
	v, last := screen(ed)
	line := v.Top + times(count) - 1
	if v.Top > 0 {
		line = max(line, v.Top+scrollMargin(v))
	}
	return toLine(ed, min(line, last))
}

func screenMiddle(ed *editor.Editor, _ int, _ bool) Target {
	v, last := screen(ed)
	return toLine(ed, v.Top+(last-v.Top)/2)
}

func screenBottom(ed *editor.Editor, count int, _ bool) Target {
	v, last := screen(ed)
	line := last - times(count) + 1
	if last < ed.Buffer().LineCount()-1 {
		line = min(line, last-scrollMargin(v))
	}
	return toLine(ed, max(line, v.Top))
}

// screen is the viewport and its last visible line (the whole document
// when it has no size yet).
func screen(ed *editor.Editor) (editor.Viewport, int) {
	v, lines := ed.Viewport(), ed.Buffer().LineCount()
	if v.Height <= 0 {
		return editor.Viewport{Height: lines}, lines - 1
	}
	return v, min(v.Top+v.Height-1, lines-1)
}

// scrollMargin is the editor's: lines kept above and below the cursor.
func scrollMargin(v editor.Viewport) int {
	return min(3, max((v.Height-1)/2, 0))
}
