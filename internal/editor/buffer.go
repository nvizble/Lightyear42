package editor

import "strings"

// Buffer holds the document text as lines of runes. Everything outside
// goes through its methods, so the storage can later become a piece table
// or a rope without touching the rest of the editor.
//
// Line endings are normalized to "\n" (a "\r\n" file is saved with "\n").
type Buffer struct {
	lines [][]rune
}

// NewBuffer builds a buffer from text. A trailing newline yields a last,
// empty line, so saving reproduces it.
func NewBuffer(text string) *Buffer {
	parts := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	lines := make([][]rune, len(parts))
	for i, p := range parts {
		lines[i] = []rune(p)
	}
	return &Buffer{lines: lines}
}

// Text returns the whole document.
func (b *Buffer) Text() string {
	var sb strings.Builder
	for i, l := range b.lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(string(l))
	}
	return sb.String()
}

// LineCount is the number of lines (at least 1).
func (b *Buffer) LineCount() int {
	return len(b.lines)
}

// Line returns line i without its newline ("" when out of range).
func (b *Buffer) Line(i int) string {
	if i < 0 || i >= len(b.lines) {
		return ""
	}
	return string(b.lines[i])
}

// LineLen is the number of runes in line i.
func (b *Buffer) LineLen(i int) int {
	if i < 0 || i >= len(b.lines) {
		return 0
	}
	return len(b.lines[i])
}

// Clamp moves p to the nearest valid position of the document.
func (b *Buffer) Clamp(p Position) Position {
	p.Line = min(max(p.Line, 0), len(b.lines)-1)
	p.Column = min(max(p.Column, 0), len(b.lines[p.Line]))
	return p
}

// End is the position after the last character of the document.
func (b *Buffer) End() Position {
	last := len(b.lines) - 1
	return Position{Line: last, Column: len(b.lines[last])}
}

// Insert puts text at p and returns the position right after it.
func (b *Buffer) Insert(p Position, text string) Position {
	p = b.Clamp(p)
	if text == "" {
		return p
	}
	parts := strings.Split(text, "\n")
	line := b.lines[p.Line]
	head := append([]rune(nil), line[:p.Column]...)
	tail := append([]rune(nil), line[p.Column:]...)

	if len(parts) == 1 {
		ins := []rune(parts[0])
		b.lines[p.Line] = append(append(head, ins...), tail...)
		return Position{Line: p.Line, Column: p.Column + len(ins)}
	}

	newLines := make([][]rune, 0, len(parts))
	newLines = append(newLines, append(head, []rune(parts[0])...))
	for _, mid := range parts[1 : len(parts)-1] {
		newLines = append(newLines, []rune(mid))
	}
	last := []rune(parts[len(parts)-1])
	end := Position{Line: p.Line + len(parts) - 1, Column: len(last)}
	newLines = append(newLines, append(last, tail...))

	b.lines = append(b.lines[:p.Line], append(newLines, b.lines[p.Line+1:]...)...)
	return end
}

// Slice returns the text inside r.
func (b *Buffer) Slice(r Range) string {
	r = r.Normalized()
	s, e := b.Clamp(r.Start), b.Clamp(r.End)
	if s.Line == e.Line {
		return string(b.lines[s.Line][s.Column:e.Column])
	}
	var sb strings.Builder
	sb.WriteString(string(b.lines[s.Line][s.Column:]))
	for i := s.Line + 1; i < e.Line; i++ {
		sb.WriteByte('\n')
		sb.WriteString(string(b.lines[i]))
	}
	sb.WriteByte('\n')
	sb.WriteString(string(b.lines[e.Line][:e.Column]))
	return sb.String()
}

// Delete removes the text inside r and returns it.
func (b *Buffer) Delete(r Range) string {
	r = r.Normalized()
	s, e := b.Clamp(r.Start), b.Clamp(r.End)
	removed := b.Slice(Range{Start: s, End: e})
	if removed == "" {
		return ""
	}
	head := append([]rune(nil), b.lines[s.Line][:s.Column]...)
	joined := append(head, b.lines[e.Line][e.Column:]...)
	b.lines = append(b.lines[:s.Line], append([][]rune{joined}, b.lines[e.Line+1:]...)...)
	return removed
}
