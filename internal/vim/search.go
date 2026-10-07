package vim

import (
	"unicode"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// Search: "/" and "?" type a pattern (plain text, not a regex), n and N
// repeat it, * and # look for the word under the cursor. Matches stay
// highlighted until :noh.

type search struct {
	pattern   []rune
	backward  bool
	word      bool // whole words only (* and #)
	highlight bool
}

// findPattern moves to the count-th match of pattern (the last one when
// empty).
func (c *Controller) findPattern(pattern string, backward, word bool, count int) Result {
	if pattern != "" {
		c.search = search{pattern: []rune(pattern), backward: backward, word: word}
	} else if c.search.pattern == nil {
		return Result{Message: "E35: nenhuma busca anterior", Err: true}
	}
	c.search.highlight = true
	return c.repeatSearch(c.search.backward, count)
}

// repeatSearch is n (backward=false, same direction) and N.
func (c *Controller) repeatSearch(backward bool, count int) Result {
	s := c.search
	if s.pattern == nil {
		return Result{Message: "E35: nenhuma busca anterior", Err: true}
	}
	c.search.highlight = true
	p := c.ed.Cursor()
	for i := 0; i < count; i++ {
		q, ok := c.nextMatch(p, backward)
		if !ok {
			return Result{Message: "E486: não encontrado: " + string(s.pattern), Err: true}
		}
		p = q
	}
	c.ed.MoveCursor(p)
	c.clampNormal()
	return Result{}
}

// nextMatch is the first match after (or before) p, wrapping around.
func (c *Controller) nextMatch(p editor.Position, backward bool) (editor.Position, bool) {
	buf := c.ed.Buffer()
	n := buf.LineCount()
	for i := 0; i <= n; i++ {
		line := (p.Line + i) % n
		if backward {
			line = ((p.Line-i)%n + n) % n
		}
		starts := c.hits([]rune(buf.Line(line)))
		if backward {
			for j := len(starts) - 1; j >= 0; j-- {
				if i > 0 || starts[j] < p.Column {
					return editor.Position{Line: line, Column: starts[j]}, true
				}
			}
			continue
		}
		for _, col := range starts {
			if i > 0 || col > p.Column {
				return editor.Position{Line: line, Column: col}, true
			}
		}
	}
	return p, false
}

// hits are the columns where the pattern starts in line.
func (c *Controller) hits(line []rune) []int {
	pat := c.search.pattern
	var out []int
	for i := 0; i+len(pat) <= len(line) && len(pat) > 0; i++ {
		if !equalRunes(line[i:i+len(pat)], pat) {
			continue
		}
		end := i + len(pat)
		if c.search.word && ((i > 0 && isWord(line[i-1])) || (end < len(line) && isWord(line[end]))) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// Hits are the matches of the search on line, to highlight: their starting
// rune columns and the pattern's length. None after :noh.
func (c *Controller) Hits(line string) (starts []int, length int) {
	if !c.search.highlight {
		return nil, 0
	}
	return c.hits([]rune(line)), len(c.search.pattern)
}

// searchWord is * and #: the word under the cursor, whole.
func (c *Controller) searchWord(backward bool, count int) Result {
	r, _, ok := wordObject(c.ed, true)
	word := []rune(c.ed.Buffer().Slice(r))
	if !ok || len(word) == 0 || !isWord(word[0]) {
		return Result{Message: "E348: nenhuma palavra sob o cursor", Err: true}
	}
	// Start from the word's beginning, so the first match is the next one.
	c.ed.MoveCursor(r.Start)
	return c.findPattern(string(word), backward, true, count)
}

func equalRunes(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
