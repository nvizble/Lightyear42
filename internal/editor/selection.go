package editor

// SelectionMode is how a selection covers text.
type SelectionMode int

// Selection modes.
const (
	SelectNone  SelectionMode = iota
	SelectChars               // from the anchor to the cursor, both included
	SelectLines               // every line from the anchor's to the cursor's
)

// Selection is the selected text: it spans from Anchor to the cursor.
type Selection struct {
	Anchor Position
	Mode   SelectionMode
}

// Select starts a selection anchored at the cursor, or changes the mode of
// the current one (keeping its anchor).
func (e *Editor) Select(mode SelectionMode) {
	if e.sel.Mode == SelectNone {
		e.sel.Anchor = e.cursor
	}
	e.sel.Mode = mode
}

// ClearSelection ends the selection.
func (e *Editor) ClearSelection() {
	e.sel = Selection{}
}

// Selection is the current selection (Mode is SelectNone when none).
func (e *Editor) Selection() Selection {
	return e.sel
}

// SwapSelectionEnds puts the cursor on the anchor and the anchor where the
// cursor was (Vim's "o" in Visual mode).
func (e *Editor) SwapSelectionEnds() {
	if e.sel.Mode == SelectNone {
		return
	}
	anchor := e.sel.Anchor
	e.sel.Anchor = e.cursor
	e.MoveCursor(anchor)
}

// SelectedRange is the text the selection covers, with an exclusive end.
// Linewise selections go from the start of the first line to the end of
// the last (line breaks are the caller's call) and report linewise=true.
// ok is false when nothing is selected.
func (e *Editor) SelectedRange() (r Range, linewise, ok bool) {
	if e.sel.Mode == SelectNone {
		return Range{}, false, false
	}
	start, end := e.buf.Clamp(e.sel.Anchor), e.cursor
	if end.Before(start) {
		start, end = end, start
	}
	if e.sel.Mode == SelectLines {
		return Range{Start: Position{Line: start.Line}, End: Position{Line: end.Line, Column: e.buf.LineLen(end.Line)}}, true, true
	}
	if end.Column >= e.buf.LineLen(end.Line) && end.Line+1 < e.buf.LineCount() {
		// Ending on a line break (an empty line) takes it too, like Vim.
		end = Position{Line: end.Line + 1}
	} else {
		end.Column = min(end.Column+1, e.buf.LineLen(end.Line))
	}
	return Range{Start: start, End: end}, false, true
}

// SelectedColumns is the part of line covered by the selection, as rune
// columns [from, to); ok is false when the line isn't selected. An empty
// selected line reports [0, 1) so it can still be drawn.
func (e *Editor) SelectedColumns(line int) (from, to int, ok bool) {
	r, linewise, ok := e.SelectedRange()
	if !ok {
		return 0, 0, false
	}
	if !linewise && r.End.Column == 0 && r.End.Line > r.Start.Line {
		// The range ends on the line break before r.End.Line.
		r.End = Position{Line: r.End.Line - 1, Column: e.buf.LineLen(r.End.Line - 1)}
	}
	if line < r.Start.Line || line > r.End.Line {
		return 0, 0, false
	}
	from, to = 0, e.buf.LineLen(line)
	if !linewise {
		if line == r.Start.Line {
			from = r.Start.Column
		}
		if line == r.End.Line {
			to = r.End.Column
		}
	}
	if to <= from {
		to = from + 1
	}
	return from, to, true
}
