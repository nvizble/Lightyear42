package editorview

import (
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// Hooks for a host that wraps the editor (lightyear test: the file tree,
// the Run button, the results).

// WithCommand adds a Normal-mode command by its keys (" e" is Space e: a
// host command starting with a space makes Space a leader key). Only the
// Vim-style editor has Normal mode; the plain one ignores it.
func (m Model) WithCommand(keys string, f func()) Model {
	if m.vim != nil {
		m.vim.Commands[keys] = func(int) vim.Result {
			f()
			return vim.Result{}
		}
	}
	return m
}

// SaveAll writes every modified buffer and reports how many it saved.
func (m Model) SaveAll() (int, error) {
	return m.ses.saveAll()
}

func (ses *session) saveAll() (int, error) {
	n := 0
	for _, b := range ses.bufs {
		if b.ed.Dirty() {
			if err := b.ed.Save(); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// OpenAt opens path (or switches to it) with the cursor on line (1-based),
// in the current window; Ctrl-o goes back, like after gd.
func (m Model) OpenAt(path string, line int) (Model, error) {
	err := m.jumpTo(path, editor.Position{Line: max(line-1, 0)})
	return m.synced(), err
}

// WithMessage shows msg on the status line until the next key.
func (m Model) WithMessage(msg string, isErr bool) Model {
	m.message, m.isError = msg, isErr
	return m
}
