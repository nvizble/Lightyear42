package editor

import "strings"

type editKind int

const (
	editInsert editKind = iota
	editDelete
)

// edit is one reversible change: text inserted at start, or deleted from
// start. before/after are the cursor around the change, restored by undo/redo.
type edit struct {
	kind          editKind
	start         Position
	text          string
	before, after Position
	// chained edits are undone and redone together with the previous one
	// (e.g. the insert half of a Replace).
	chained bool
}

// history keeps the undo and redo stacks. Consecutive typing (or erasing)
// merges into one step, like most editors; any cursor jump seals it.
type history struct {
	undo, redo []edit
	sealed     bool
	// grouping is set between BeginGroup and EndGroup: every edit after the
	// group's first is chained to it, so the whole group undoes as one.
	grouping, groupStarted bool
}

// record stores a new edit, merging it into the previous one when the user
// is just continuing to type or erase.
func (h *history) record(e edit) {
	h.redo = nil
	if h.grouping {
		e.chained = e.chained || h.groupStarted
		h.groupStarted = true
		h.undo = append(h.undo, e)
		return
	}
	if n := len(h.undo); n > 0 && !h.sealed && !e.chained && !strings.Contains(e.text, "\n") {
		last := &h.undo[n-1]
		switch {
		case e.kind == editInsert && last.kind == editInsert && advance(last.start, last.text) == e.start:
			last.text += e.text
			last.after = e.after
			return
		case e.kind == editDelete && last.kind == editDelete && advance(e.start, e.text) == last.start:
			// Backspace run: the new deletion sits right before the previous one.
			last.start, last.text = e.start, e.text+last.text
			last.after = e.after
			return
		case e.kind == editDelete && last.kind == editDelete && e.start == last.start:
			// Delete-key run: deleting forward from the same spot.
			last.text += e.text
			last.after = e.after
			return
		}
	}
	h.undo = append(h.undo, e)
	// A new line ends the step: undo takes back one line of typing at a time.
	h.sealed = strings.Contains(e.text, "\n")
}

// seal stops the next edit from merging into the current step.
func (h *history) seal() {
	h.sealed = true
}
