package editor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultTabSize is how many screen columns a tab spans.
const DefaultTabSize = 4

// Editor is one open document: buffer, cursor, viewport and history.
// Every change goes through Insert/Delete/Replace, so it can be undone.
type Editor struct {
	buf    *Buffer
	cursor Position
	// want is the screen column to aim for when moving up and down, so the
	// cursor keeps its column across short lines.
	want    int
	view    Viewport
	hist    history
	path    string
	dirty   bool
	tabSize int
	sel     Selection
}

// New opens an unnamed document with text.
func New(text string) *Editor {
	return &Editor{buf: NewBuffer(text), tabSize: DefaultTabSize}
}

// Open loads the file at path. A missing file opens an empty document that
// Save will create.
func Open(path string) (*Editor, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("abrir %s: %w", path, err)
	}
	e := New(string(data))
	e.path = path
	return e, nil
}

// Save writes the document to its file, creating missing directories.
func (e *Editor) Save() error {
	if e.path == "" {
		return errors.New("documento sem arquivo: use SaveAs")
	}
	return e.SaveAs(e.path)
}

// SaveAs writes the document to path and binds the editor to it.
func (e *Editor) SaveAs(path string) error {
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("salvar %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(e.buf.Text()), mode); err != nil {
		return fmt.Errorf("salvar %s: %w", path, err)
	}
	e.path, e.dirty = path, false
	return nil
}

// Buffer exposes the document for reading (renderers, syntax, LSP).
func (e *Editor) Buffer() *Buffer { return e.buf }

// Cursor is the current position.
func (e *Editor) Cursor() Position { return e.cursor }

// Path is the file being edited ("" for an unnamed document).
func (e *Editor) Path() string { return e.path }

// Dirty reports unsaved changes.
func (e *Editor) Dirty() bool { return e.dirty }

// TabSize is how many screen columns a tab spans.
func (e *Editor) TabSize() int { return e.tabSize }

// Viewport is the visible region, kept around the cursor.
func (e *Editor) Viewport() Viewport { return e.view }

// SetViewportSize resizes the visible region (in cells).
func (e *Editor) SetViewportSize(width, height int) {
	e.view.Width, e.view.Height = width, height
	e.follow()
}

// ScrollBy moves the viewport without moving the cursor (mouse wheel).
func (e *Editor) ScrollBy(lines int) {
	maxTop := max(e.buf.LineCount()-1, 0)
	e.view.Top = min(max(e.view.Top+lines, 0), maxTop)
}

// Insert types text at the cursor and moves the cursor after it.
func (e *Editor) Insert(text string) {
	if text == "" {
		return
	}
	start := e.cursor
	end := e.buf.Insert(start, text)
	e.hist.record(edit{kind: editInsert, start: start, text: text, before: start, after: end})
	e.setCursor(end)
	e.dirty = true
}

// Delete removes the text inside r, leaving the cursor at its start, and
// returns what was removed.
func (e *Editor) Delete(r Range) string {
	r = r.Normalized()
	r.Start, r.End = e.buf.Clamp(r.Start), e.buf.Clamp(r.End)
	before := e.cursor
	removed := e.buf.Delete(r)
	if removed == "" {
		return ""
	}
	e.hist.record(edit{kind: editDelete, start: r.Start, text: removed, before: before, after: r.Start})
	e.setCursor(r.Start)
	e.dirty = true
	return removed
}

// Replace swaps the text inside r for text, as one undo step, leaving the
// cursor after the new text.
func (e *Editor) Replace(r Range, text string) {
	r = r.Normalized()
	r.Start, r.End = e.buf.Clamp(r.Start), e.buf.Clamp(r.End)
	before := e.cursor
	removed := e.buf.Delete(r)
	end := e.buf.Insert(r.Start, text)
	if removed == "" && text == "" {
		return
	}
	e.hist.seal()
	if removed != "" {
		e.hist.record(edit{kind: editDelete, start: r.Start, text: removed, before: before, after: r.Start})
	}
	if text != "" {
		e.hist.record(edit{kind: editInsert, start: r.Start, text: text, before: r.Start, after: end, chained: removed != ""})
	}
	e.hist.seal()
	e.setCursor(end)
	e.dirty = true
}

// DeleteBackward erases the character before the cursor (Backspace),
// joining with the previous line at the start of a line.
func (e *Editor) DeleteBackward() {
	c := e.cursor
	switch {
	case c.Column > 0:
		e.Delete(Range{Start: Position{c.Line, c.Column - 1}, End: c})
	case c.Line > 0:
		e.Delete(Range{Start: Position{c.Line - 1, e.buf.LineLen(c.Line - 1)}, End: c})
	}
}

// DeleteForward erases the character under the cursor (Delete key),
// joining the next line at the end of a line.
func (e *Editor) DeleteForward() {
	c := e.cursor
	switch {
	case c.Column < e.buf.LineLen(c.Line):
		e.Delete(Range{Start: c, End: Position{c.Line, c.Column + 1}})
	case c.Line < e.buf.LineCount()-1:
		e.Delete(Range{Start: c, End: Position{c.Line + 1, 0}})
	}
}

// PlaceCursor moves the cursor to p (clamped) but keeps the column that
// vertical moves aim for, so moving up or down later resumes from it (e.g.
// a modal editor pulling the cursor back onto the last character).
func (e *Editor) PlaceCursor(p Position) {
	e.hist.seal()
	e.cursor = e.buf.Clamp(p)
	e.follow()
}

// BeginGroup starts a group of edits that undo and redo as one step (e.g. a
// Vim insert session). Groups don't nest; EndGroup closes it.
func (e *Editor) BeginGroup() {
	e.hist.seal()
	e.hist.grouping, e.hist.groupStarted = true, false
}

// EndGroup closes the group started by BeginGroup.
func (e *Editor) EndGroup() {
	e.hist.grouping = false
	e.hist.seal()
}

// MoveCursor jumps to p (clamped). A jump ends the current undo step.
func (e *Editor) MoveCursor(p Position) {
	e.hist.seal()
	e.setCursor(e.buf.Clamp(p))
}

// Movement is a basic cursor move, shared by any front end.
type Movement int

// Basic movements.
const (
	MoveLeft Movement = iota
	MoveRight
	MoveUp
	MoveDown
	MoveLineStart
	MoveLineEnd
	MoveDocStart
	MoveDocEnd
	MovePageUp
	MovePageDown
)

// Move applies a basic movement. Vertical moves keep the screen column the
// cursor had before (across shorter lines and tabs).
func (e *Editor) Move(m Movement) {
	c := e.cursor
	switch m {
	case MoveLeft:
		if c.Column > 0 {
			c.Column--
		} else if c.Line > 0 {
			c = Position{c.Line - 1, e.buf.LineLen(c.Line - 1)}
		}
	case MoveRight:
		if c.Column < e.buf.LineLen(c.Line) {
			c.Column++
		} else if c.Line < e.buf.LineCount()-1 {
			c = Position{c.Line + 1, 0}
		}
	case MoveUp, MoveDown, MovePageUp, MovePageDown:
		step := map[Movement]int{MoveUp: -1, MoveDown: 1, MovePageUp: -max(e.view.Height-1, 1), MovePageDown: max(e.view.Height-1, 1)}[m]
		line := min(max(c.Line+step, 0), e.buf.LineCount()-1)
		e.hist.seal()
		e.cursor = Position{line, e.columnAt(line, e.want)}
		e.follow()
		return
	case MoveLineStart:
		c.Column = 0
	case MoveLineEnd:
		c.Column = e.buf.LineLen(c.Line)
	case MoveDocStart:
		c = Position{}
	case MoveDocEnd:
		c = e.buf.End()
	}
	e.MoveCursor(c)
}

// Undo takes back the last change (with the edits chained to it); false
// when there is nothing to undo.
func (e *Editor) Undo() bool {
	if len(e.hist.undo) == 0 {
		return false
	}
	for {
		n := len(e.hist.undo)
		ed := e.hist.undo[n-1]
		e.hist.undo = e.hist.undo[:n-1]
		switch ed.kind {
		case editInsert:
			e.buf.Delete(Range{Start: ed.start, End: advance(ed.start, ed.text)})
		case editDelete:
			e.buf.Insert(ed.start, ed.text)
		}
		e.hist.redo = append(e.hist.redo, ed)
		e.setCursor(ed.before)
		if !ed.chained || len(e.hist.undo) == 0 {
			break
		}
	}
	e.hist.seal()
	e.dirty = true
	return true
}

// Redo reapplies the last undone change (with the edits chained after it);
// false when there is none.
func (e *Editor) Redo() bool {
	if len(e.hist.redo) == 0 {
		return false
	}
	for {
		n := len(e.hist.redo)
		ed := e.hist.redo[n-1]
		e.hist.redo = e.hist.redo[:n-1]
		switch ed.kind {
		case editInsert:
			e.buf.Insert(ed.start, ed.text)
		case editDelete:
			e.buf.Delete(Range{Start: ed.start, End: advance(ed.start, ed.text)})
		}
		e.hist.undo = append(e.hist.undo, ed)
		e.setCursor(ed.after)
		if len(e.hist.redo) == 0 || !e.hist.redo[len(e.hist.redo)-1].chained {
			break
		}
	}
	e.hist.seal()
	e.dirty = true
	return true
}

// VisualColumn is the screen column of (line, col), expanding tabs.
func (e *Editor) VisualColumn(line, col int) int {
	visual := 0
	for i, r := range []rune(e.buf.Line(line)) {
		if i >= col {
			break
		}
		if r == '\t' {
			visual += e.tabSize - visual%e.tabSize
		} else {
			visual++
		}
	}
	return visual
}

// columnAt is the rune column of line closest to the screen column visual.
func (e *Editor) columnAt(line, visual int) int {
	runes := []rune(e.buf.Line(line))
	col, x := 0, 0
	for col < len(runes) {
		w := 1
		if runes[col] == '\t' {
			w = e.tabSize - x%e.tabSize
		}
		if x+w > visual {
			break
		}
		x += w
		col++
	}
	return col
}

// ColumnAt maps a screen column of line to a rune column (mouse clicks).
func (e *Editor) ColumnAt(line, visual int) int {
	return e.columnAt(e.buf.Clamp(Position{Line: line}).Line, visual)
}

func (e *Editor) setCursor(p Position) {
	e.cursor = p
	e.want = e.VisualColumn(p.Line, p.Column)
	e.follow()
}

func (e *Editor) follow() {
	e.view.follow(e.cursor.Line, e.VisualColumn(e.cursor.Line, e.cursor.Column))
}
