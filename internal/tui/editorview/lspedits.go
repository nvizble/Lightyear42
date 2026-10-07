package editorview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// Language server edits: grn (or :Rename name) renames across the project,
// grr lists the references, gra the code actions (fixes), and :Format
// formats the file. Edits to other files open them in buffers, modified,
// to review and save with :wa.

// picker is a list to choose from (references, code actions).
type picker struct {
	title    string
	items    []string
	selected int
	choose   func(m Model, i int) (Model, tea.Cmd)
}

type renameMsg struct {
	edits []lsp.FileEdit
	err   error
}

type referencesMsg struct {
	locs []lsp.Location
	err  error
}

type actionsMsg struct {
	actions []lsp.Action
	client  *lsp.Client
	err     error
	apply   bool // a single action is applied right away (a click on a fix)
}

type formatMsg struct {
	path  string
	edits []lsp.TextEdit
	err   error
}

type ranMsg struct{ err error }

// editCommands are grn, grr and gra.
func (m Model) editCommands() map[string]func(int) vim.Result {
	return map[string]func(int) vim.Result{
		"grn": func(int) vim.Result {
			m.vim.OpenCommandLine("Rename ")
			return vim.Result{}
		},
		"grr": func(int) vim.Result {
			return m.askCursor(func(ctx context.Context, c *lsp.Client, path string, p lsp.Pos) tea.Msg {
				locs, err := c.References(ctx, path, p)
				return referencesMsg{locs, err}
			})
		},
		"gra": func(int) vim.Result {
			b := m.ses.current()
			var diags []lsp.Diagnostic
			if b.lsp != nil {
				line := b.ed.Cursor().Line
				for _, d := range b.lsp.diags {
					if d.Start.Line <= line && line <= d.End.Line {
						diags = append(diags, d)
					}
				}
			}
			return m.askCursor(func(ctx context.Context, c *lsp.Client, path string, p lsp.Pos) tea.Msg {
				actions, err := c.CodeActions(ctx, path, p, diags)
				return actionsMsg{actions: actions, client: c, err: err}
			})
		},
	}
}

// editExCommands are :Rename and :Format.
func (m Model) editExCommands() map[string]func(string) vim.Result {
	rename := func(name string) vim.Result {
		if name == "" {
			return vim.Result{Message: "falta o nome novo (:Rename nome)", Err: true}
		}
		return m.askCursor(func(ctx context.Context, c *lsp.Client, path string, p lsp.Pos) tea.Msg {
			edits, err := c.Rename(ctx, path, p, name)
			return renameMsg{edits, err}
		})
	}
	format := func(string) vim.Result {
		b := m.ses.current()
		path := b.ed.Path()
		if lang := language(path); (lang == "C" || lang == "C++") && !hasClangFormat(path) {
			// clangd's default style breaks the 42 norm: only format when
			// the project says how.
			return vim.Result{Message: "sem .clang-format no projeto: a formatação de C fica desligada (o estilo padrão quebra a norminette)", Err: true}
		}
		tab := b.ed.TabSize()
		return m.askCursor(func(ctx context.Context, c *lsp.Client, path string, _ lsp.Pos) tea.Msg {
			edits, err := c.Format(ctx, path, tab)
			return formatMsg{path, edits, err}
		})
	}
	return map[string]func(string) vim.Result{
		"Rename": rename, "rename": rename,
		"Format": format, "format": format,
	}
}

// askCursor sends a request about the cursor of the current buffer.
func (m Model) askCursor(f func(ctx context.Context, c *lsp.Client, path string, p lsp.Pos) tea.Msg) vim.Result {
	b := m.ses.current()
	if b.lsp == nil {
		return vim.Result{Message: "sem language server para este arquivo", Err: true}
	}
	cur, path := b.ed.Cursor(), b.lsp.path
	p := lsp.Pos{Line: cur.Line, Col: cur.Column}
	return b.lsp.request(func(ctx context.Context, c *lsp.Client) tea.Msg { return f(ctx, c, path, p) })
}

// editReply handles the answers.
func (m Model) editReply(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case renameMsg:
		switch {
		case msg.err != nil:
			m.message, m.isError = msg.err.Error(), true
		case len(msg.edits) == 0:
			m.message = "nada para renomear aqui"
		default:
			m = m.applyEdits(msg.edits, "rename")
		}
	case referencesMsg:
		switch {
		case msg.err != nil:
			m.message, m.isError = msg.err.Error(), true
		case len(msg.locs) == 0:
			m.message = "nenhuma referência"
		case len(msg.locs) == 1:
			m = m.goTo(msg.locs[0])
		default:
			items := make([]string, len(msg.locs))
			for i, l := range msg.locs {
				items[i] = fmt.Sprintf("%s:%d  %s", filepath.Base(l.Path), l.Pos.Line+1, m.lineOf(l))
			}
			locs := msg.locs
			m.ses.pick = &picker{title: fmt.Sprintf("%d referências", len(locs)), items: items,
				choose: func(m Model, i int) (Model, tea.Cmd) { return m.goTo(locs[i]), nil }}
		}
	case actionsMsg:
		switch {
		case msg.err != nil:
			m.message, m.isError = msg.err.Error(), true
		case len(msg.actions) == 0:
			m.message = "nenhuma ação aqui"
		default:
			items := make([]string, len(msg.actions))
			for i, a := range msg.actions {
				items[i] = a.Title
			}
			actions, c := msg.actions, msg.client
			run := func(m Model, i int) (Model, tea.Cmd) {
				a := actions[i]
				if len(a.Edits) > 0 {
					m = m.applyEdits(a.Edits, a.Title)
				}
				// A command runs on the server, which sends its edits back.
				return m, func() tea.Msg {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					return ranMsg{c.Run(ctx, a)}
				}
			}
			if msg.apply && len(actions) == 1 {
				return run(m, 0)
			}
			m.ses.pick = &picker{title: "ações", items: items, choose: run}
		}
	case formatMsg:
		switch {
		case msg.err != nil:
			m.message, m.isError = msg.err.Error(), true
		case len(msg.edits) == 0:
			m.message = "já está formatado"
		default:
			m = m.applyEdits([]lsp.FileEdit{{Path: msg.path, Edits: msg.edits}}, "format")
		}
	case ranMsg:
		if msg.err != nil {
			m.message, m.isError = msg.err.Error(), true
		}
	}
	return m, nil
}

// goTo jumps to a location (opening its file), for Ctrl-o to come back.
func (m Model) goTo(l lsp.Location) Model {
	if err := m.jumpTo(l.Path, editor.Position{Line: l.Pos.Line, Column: l.Pos.Col}); err != nil {
		m.message, m.isError = err.Error(), true
	}
	return m.synced()
}

// lineOf is the trimmed text of a location's line, from its buffer or file.
func (m Model) lineOf(l lsp.Location) string {
	for _, b := range m.ses.bufs {
		if sameFile(b.ed.Path(), l.Path) {
			return strings.TrimSpace(b.ed.Buffer().Line(l.Pos.Line))
		}
	}
	data, _ := os.ReadFile(l.Path)
	if lines := strings.Split(string(data), "\n"); l.Pos.Line < len(lines) {
		return strings.TrimSpace(lines[l.Pos.Line])
	}
	return ""
}

// applyEdits makes the edits, each file in its buffer (opened when needed)
// as one undo step; the current buffer and cursor stay.
func (m Model) applyEdits(files []lsp.FileEdit, what string) Model {
	home, cursor := m.ses.cur, m.ses.current().ed.Cursor()
	changes := 0
	for _, f := range files {
		if err := m.open(f.Path); err != nil {
			m.message, m.isError = err.Error(), true
			continue
		}
		ed := m.ses.current().ed
		edits := slices.Clone(f.Edits)
		// Last ones first, so earlier positions stay right.
		slices.SortFunc(edits, func(a, b lsp.TextEdit) int {
			pa, pb := editor.Position{Line: a.Start.Line, Column: a.Start.Col}, editor.Position{Line: b.Start.Line, Column: b.Start.Col}
			switch {
			case pb.Before(pa):
				return -1
			case pa.Before(pb):
				return 1
			}
			return 0
		})
		ed.BeginGroup()
		for _, e := range edits {
			ed.Replace(editor.Range{Start: editor.Position{Line: e.Start.Line, Column: e.Start.Col}, End: editor.Position{Line: e.End.Line, Column: e.End.Col}}, e.Text)
			changes++
		}
		ed.EndGroup()
	}
	m.show(home)
	m.moveCursor(cursor)
	m = m.synced()
	if m.message == "" {
		m.message = fmt.Sprintf("%s: %d mudança(s) em %d arquivo(s)", what, changes, len(files))
		if len(files) > 1 {
			m.message += " (:wa salva)"
		}
	}
	return m
}

// pickKey handles the keys of an open picker.
func (m Model) pickKey(k string) (Model, tea.Cmd) {
	p := m.ses.pick
	switch k {
	case "j", "down", "ctrl+n", "tab":
		p.selected = (p.selected + 1) % len(p.items)
	case "k", "up", "ctrl+p", "shift+tab":
		p.selected = (p.selected + len(p.items) - 1) % len(p.items)
	case "enter":
		m.ses.pick = nil
		return p.choose(m, p.selected)
	case "esc", "q":
		m.ses.pick = nil
	}
	return m, nil
}

// pickerBox renders a picker: its title (when it has one) and the items
// around the selected one.
func (m Model) pickerBox(p *picker) []string {
	from := max(0, min(p.selected-popupRows/2, len(p.items)-popupRows))
	to := min(len(p.items), from+popupRows)
	w := ansi.StringWidth(p.title)
	for _, it := range p.items[from:to] {
		w = max(w, ansi.StringWidth(it))
	}
	w = min(w, max(m.width-6, 10))
	var lines []string
	if p.title != "" {
		lines = append(lines, stylePopupInfo.Render(" "+ansi.Truncate(p.title, w, "…")+strings.Repeat(" ", max(w-ansi.StringWidth(p.title), 0))+" "))
	}
	for i := from; i < to; i++ {
		text := ansi.Truncate(p.items[i], w, "…")
		text = " " + text + strings.Repeat(" ", w-ansi.StringWidth(text)) + " "
		if i == p.selected {
			lines = append(lines, stylePopupSel.Render(text))
		} else {
			lines = append(lines, stylePopup.Render(text))
		}
	}
	return lines
}

// hasClangFormat reports a .clang-format (or _clang-format) in the file's
// folder or above.
func hasClangFormat(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for dir := filepath.Dir(abs); ; dir = filepath.Dir(dir) {
		for _, name := range []string{".clang-format", "_clang-format"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return true
			}
		}
		if filepath.Dir(dir) == dir {
			return false
		}
	}
}
