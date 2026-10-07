// Package editor is the core of lightyear's embedded code editor: the
// document buffer, the cursor, the viewport and reversible edits.
//
// It knows nothing about Vim, the TUI, LSP or syntax highlighting; those
// layers drive it through the methods of Editor (see DESIGN.md).
package editor

// Position is a place in the document: 0-based line and column. Columns
// count runes (not bytes, not screen cells); expanding tabs and wide
// characters is the renderer's concern.
type Position struct {
	Line, Column int
}

// Before reports whether p comes before q in the document.
func (p Position) Before(q Position) bool {
	return p.Line < q.Line || (p.Line == q.Line && p.Column < q.Column)
}

// Range is a span of text from Start (inclusive) to End (exclusive).
type Range struct {
	Start, End Position
}

// Normalized returns r with Start not after End.
func (r Range) Normalized() Range {
	if r.End.Before(r.Start) {
		return Range{Start: r.End, End: r.Start}
	}
	return r
}

// Empty reports whether the range covers no text.
func (r Range) Empty() bool {
	return r.Start == r.End
}

// advance returns the position right after text when it is inserted at p.
func advance(p Position, text string) Position {
	for _, r := range text {
		if r == '\n' {
			p.Line++
			p.Column = 0
		} else {
			p.Column++
		}
	}
	return p
}
