package vim

import (
	"strings"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// Text objects select a piece of text around the cursor, after an operator
// (diw, ci", da() or in Visual mode (viw). "i" is the inner part, "a"
// includes the surroundings (quotes, brackets, the blanks after a word).

// textObject finds the range of the object named by key ("iw", "a(") at
// the cursor; linewise when it covers whole lines (an inner block), ok is
// false when there is none.
type textObject func(ed *editor.Editor, inner bool) (r editor.Range, linewise, ok bool)

var textObjects = map[string]textObject{
	"w": wordObject,
	`"`: quoteObject('"'), "'": quoteObject('\''), "`": quoteObject('`'),
	"(": bracketObject('(', ')'), ")": bracketObject('(', ')'), "b": bracketObject('(', ')'),
	"{": bracketObject('{', '}'), "}": bracketObject('{', '}'), "B": bracketObject('{', '}'),
	"[": bracketObject('[', ']'), "]": bracketObject('[', ']'),
	"<": bracketObject('<', '>'), ">": bracketObject('<', '>'),
}

// lookupObject resolves a key like "iw" or "a{".
func lookupObject(key string) (textObject, bool, bool) {
	if len(key) < 2 || (key[0] != 'i' && key[0] != 'a') {
		return nil, false, false
	}
	obj, ok := textObjects[key[1:]]
	return obj, key[0] == 'i', ok
}

// wordObject is iw (the run of word characters, punctuation or blanks under
// the cursor) and aw (the word plus the blanks after it, or before it when
// there are none after).
func wordObject(ed *editor.Editor, inner bool) (editor.Range, bool, bool) {
	buf, cur := ed.Buffer(), ed.Cursor()
	line := []rune(buf.Line(cur.Line))
	if len(line) == 0 {
		return editor.Range{}, false, false
	}
	class := func(i int) int { return classAt(buf, editor.Position{Line: cur.Line, Column: i}) }
	from, to := cur.Column, cur.Column+1
	cls := class(cur.Column)
	for from > 0 && class(from-1) == cls {
		from--
	}
	for to < len(line) && class(to) == cls {
		to++
	}
	if !inner {
		switch {
		case cls == blankClass:
			// Blanks, then the word after them.
			if to < len(line) {
				next := class(to)
				for to < len(line) && class(to) == next {
					to++
				}
			}
		case to < len(line) && class(to) == blankClass:
			for to < len(line) && class(to) == blankClass {
				to++
			}
		default:
			for from > 0 && class(from-1) == blankClass {
				from--
			}
		}
	}
	return editor.Range{Start: editor.Position{Line: cur.Line, Column: from}, End: editor.Position{Line: cur.Line, Column: to}}, false, true
}

// quoteObject is i" (inside the quotes on the cursor's line) and a" (with
// the quotes and the blanks after them). Quotes pair up from the start of
// the line, skipping escaped ones, like Vim.
func quoteObject(q rune) textObject {
	return func(ed *editor.Editor, inner bool) (editor.Range, bool, bool) {
		cur := ed.Cursor()
		line := []rune(ed.Buffer().Line(cur.Line))
		var quotes []int
		for i, r := range line {
			if r == q && (i == 0 || line[i-1] != '\\') {
				quotes = append(quotes, i)
			}
		}
		for i := 0; i+1 < len(quotes); i += 2 {
			open, end := quotes[i], quotes[i+1]
			// The pair around the cursor, or the first one after it.
			if cur.Column > end {
				continue
			}
			from, to := open, end+1
			if inner {
				from, to = open+1, end
			} else {
				for to < len(line) && (line[to] == ' ' || line[to] == '\t') {
					to++
				}
			}
			return editor.Range{Start: editor.Position{Line: cur.Line, Column: from}, End: editor.Position{Line: cur.Line, Column: to}}, false, true
		}
		return editor.Range{}, false, false
	}
}

// bracketObject is i( (inside the brackets around the cursor, across lines)
// and a( (with the brackets). Like Vim, an inner block between a bracket
// ending its line and one alone on its line is whole lines (linewise), so
// di{ keeps "{" and "}" on their lines.
func bracketObject(open, close rune) textObject {
	return func(ed *editor.Editor, inner bool) (editor.Range, bool, bool) {
		buf, cur := ed.Buffer(), ed.Cursor()
		at := func(p editor.Position) rune { r, _ := buf.RuneAt(p); return r }

		// The open bracket: under the cursor, or the first unmatched one
		// before it (a close bracket under the cursor looks for its pair).
		start, ok := cur, true
		if at(cur) == close {
			start, ok = prev(buf, cur)
		}
		for depth := 0; ok; start, ok = prev(buf, start) {
			if r := at(start); r == close {
				depth++
			} else if r == open {
				if depth == 0 {
					break
				}
				depth--
			}
		}
		if !ok || at(start) != open {
			return editor.Range{}, false, false
		}
		end, depth := start, 0
		for {
			p, ok := next(buf, end)
			if !ok {
				return editor.Range{}, false, false
			}
			end = p
			if r := at(end); r == open {
				depth++
			} else if r == close {
				if depth == 0 {
					break
				}
				depth--
			}
		}
		if !inner {
			return editor.Range{Start: start, End: editor.Position{Line: end.Line, Column: end.Column + 1}}, false, true
		}
		from, _ := next(buf, start)
		to := end
		startsLine := from.Column == buf.LineLen(from.Line) && from.Line < end.Line
		if startsLine {
			from = editor.Position{Line: from.Line + 1}
		}
		endsLine := to.Line > from.Line && strings.TrimLeft(string([]rune(buf.Line(to.Line))[:to.Column]), " \t") == ""
		if endsLine {
			to = editor.Position{Line: to.Line - 1, Column: buf.LineLen(to.Line - 1)}
		}
		if !from.Before(to) {
			if startsLine {
				return editor.Range{}, false, false // "{\n}": nothing inside
			}
			return editor.Range{Start: from, End: from}, false, true // "()": ci( types inside
		}
		return editor.Range{Start: from, End: to}, startsLine && endsLine, true
	}
}
