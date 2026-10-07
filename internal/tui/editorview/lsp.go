package editorview

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
)

// The language server side of the editor (internal/lsp): the server starts
// with the editor, gets the text after every change, and its diagnostics
// are drawn on the code, in the gutter and in the status line.

var (
	colorWarn = lipgloss.AdaptiveColor{Light: "130", Dark: "214"}

	// By lsp.Severity: the gutter and status line, and the code underlined.
	severityStyles = [...]lipgloss.Style{
		lsp.Error:       lipgloss.NewStyle().Foreground(colorFail),
		lsp.Warning:     lipgloss.NewStyle().Foreground(colorWarn),
		lsp.Information: lipgloss.NewStyle().Foreground(colorAccent),
		lsp.Hint:        lipgloss.NewStyle().Foreground(colorAccent),
	}
	markStyles = [...]lipgloss.Style{
		lsp.Error:       severityStyles[lsp.Error].Underline(true),
		lsp.Warning:     severityStyles[lsp.Warning].Underline(true),
		lsp.Information: severityStyles[lsp.Information].Underline(true),
		lsp.Hint:        severityStyles[lsp.Hint].Underline(true),
	}
	severityNames = [...]string{lsp.Error: "erro", lsp.Warning: "aviso", lsp.Information: "info", lsp.Hint: "dica"}
)

// lspState is shared by the model's copies.
type lspState struct {
	server   lsp.Server
	client   *lsp.Client   // nil until started, and after the server stops
	starting bool          // started in the background, not ready yet
	stop     chan struct{} // closed by Close
	dirty    bool          // the text changed since the server last got it
	diags    []lsp.Diagnostic
}

type lspStartedMsg struct {
	client *lsp.Client
	err    error
}

type lspEventMsg lsp.Event

// WithLSP makes the editor run the language server for its file (clangd,
// gopls, pyright, rust-analyzer), when there is one. A missing server is
// reported in the status line, with how to install it.
func (m Model) WithLSP() Model {
	s, ok := lsp.ServerFor(m.ed.Path())
	if !ok || m.ed.Path() == "" {
		return m
	}
	st := &lspState{server: s, starting: true, stop: make(chan struct{})}
	m.ed.OnChange(func(editor.Change) { st.dirty = true })
	m.lsp = st
	return m
}

// startLSP starts the server in the background.
func (m Model) startLSP() tea.Cmd {
	if m.lsp == nil {
		return nil
	}
	server, path, text := m.lsp.server, m.ed.Path(), m.ed.Buffer().Text()
	return func() tea.Msg {
		c, err := lsp.Start(context.Background(), server, path, text)
		return lspStartedMsg{c, err}
	}
}

// waitLSP waits for the server's next event.
func waitLSP(st *lspState) tea.Cmd {
	events := st.client.Events()
	return func() tea.Msg {
		select {
		case ev := <-events:
			return lspEventMsg(ev)
		case <-st.stop:
			return nil
		}
	}
}

// lspMsg handles the server's news.
func (m Model) lspMsg(msg tea.Msg) (Model, tea.Cmd) {
	st := m.lsp
	switch msg := msg.(type) {
	case lspStartedMsg:
		st.starting = false
		if msg.err != nil {
			m.message, m.isError = msg.err.Error(), true
			return m, nil
		}
		select {
		case <-st.stop: // closed while starting
			msg.client.Close()
			return m, nil
		default:
		}
		st.client = msg.client
		m.syncLSP() // edits made while it started
		return m, waitLSP(st)
	case lspEventMsg:
		if msg.Err != nil {
			st.client, st.diags = nil, nil
			m.message, m.isError = msg.Err.Error(), true
			return m, nil
		}
		st.diags = msg.Diagnostics
		return m, waitLSP(st)
	}
	return m, nil
}

// syncLSP sends the text to the server when it changed.
func (m Model) syncLSP() {
	if st := m.lsp; st != nil && st.client != nil && st.dirty {
		st.client.DidChange(m.ed.Buffer().Text())
		st.dirty = false
	}
}

// closeLSP stops the server.
func (m Model) closeLSP() {
	if st := m.lsp; st != nil {
		close(st.stop)
		if st.client != nil {
			st.client.Close()
		}
	}
}

// marks are the diagnostics' severities on line (length runes long), by
// rune; 0 where there is none, and the most severe wins. Empty ranges
// still mark one cell.
func (m Model) marks(line, length int) []lsp.Severity {
	if m.lsp == nil {
		return nil
	}
	var out []lsp.Severity
	for _, d := range m.lsp.diags {
		if line < d.Start.Line || line > d.End.Line {
			continue
		}
		from, to := 0, length
		if line == d.Start.Line {
			from = d.Start.Col
		}
		if line == d.End.Line {
			to = d.End.Col
		}
		to = max(to, from+1)
		for len(out) < to {
			out = append(out, 0)
		}
		for i := from; i < to; i++ {
			if out[i] == 0 || d.Severity < out[i] {
				out[i] = d.Severity
			}
		}
	}
	return out
}

// count is how many diagnostics have severity sev.
func (m Model) count(sev lsp.Severity) int {
	n := 0
	for _, d := range m.lsp.diags {
		if d.Severity == sev {
			n++
		}
	}
	return n
}

// worst is the most severe diagnostic on line, or nil.
func (m Model) worst(line int) *lsp.Diagnostic {
	if m.lsp == nil {
		return nil
	}
	var w *lsp.Diagnostic
	for i, d := range m.lsp.diags {
		if d.Start.Line <= line && line <= d.End.Line && (w == nil || d.Severity < w.Severity) {
			w = &m.lsp.diags[i]
		}
	}
	return w
}
