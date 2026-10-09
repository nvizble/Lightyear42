package editorview

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
)

// The language server side of the editor (internal/lsp): a server per
// project starts with the editor (or when a buffer of a new project opens),
// gets each buffer's text after every change, and its diagnostics are drawn
// on the code, in the gutter and in the status line. Hover, definition and
// completion are in lspfeatures.go.

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

// lspServer is one running language server, shared by the buffers of its
// project.
type lspServer struct {
	server   lsp.Server
	program  string // what runs for the server (lsp.Server.Program)
	root     string
	client   *lsp.Client   // nil until started, and after the server stops
	starting bool          // started in the background, not ready yet
	fetching bool          // and being downloaded first (lsp.Server.Download)
	stop     chan struct{} // closed by Close
}

// lspState is a buffer's side of its language server.
type lspState struct {
	srv   *lspServer
	path  string
	open  bool // the server got the document
	dirty bool // the text changed since the server last got it
	diags []lsp.Diagnostic

	pending tea.Cmd     // a request a key started, for Update to return
	hover   []string    // the hover popup's lines (nil when closed)
	comp    *completion // the completion list (nil when closed)
	seq     int         // completion requests; answers to older ones are dropped
}

type lspStartedMsg struct {
	srv    *lspServer
	client *lsp.Client
	err    error
}

type lspEventMsg struct {
	srv *lspServer
	ev  lsp.Event
}

// WithLSP makes the editor run language servers for its files (clangd,
// gopls, pyright, rust-analyzer): one per project, shared by its buffers. A
// missing server is reported in the status line, with how to install it.
func (m Model) WithLSP() Model {
	m.ses.lsp = true
	for _, b := range m.ses.bufs {
		m.ses.attach(b)
	}
	if m.vim != nil {
		for key, f := range m.lspCommands() {
			m.vim.Commands[key] = f
		}
		for key, f := range m.editCommands() {
			m.vim.Commands[key] = f
		}
		for name, f := range m.editExCommands() {
			m.vim.ExCommands[name] = f
		}
	}
	return m.synced()
}

// attach links b to the server of its project, starting it when needed
// (Update returns the start, see session.starts).
func (ses *session) attach(b *buffer) {
	path := b.ed.Path()
	s, ok := lsp.ServerFor(path)
	if !ses.lsp || !ok || path == "" {
		return
	}
	root := lsp.Root(s, path)
	key := s.Name + "\x00" + root
	srv, running := ses.servers[key]
	if !running {
		srv = &lspServer{server: s, program: s.Program(), root: root, starting: true, fetching: s.WillDownload(), stop: make(chan struct{})}
		ses.servers[key] = srv
		ses.starts = append(ses.starts, func() tea.Msg {
			c, err := lsp.Start(context.Background(), s, root)
			return lspStartedMsg{srv, c, err}
		})
	}
	st := &lspState{srv: srv, path: path}
	b.lsp = st
	b.ed.OnChange(func(editor.Change) { st.dirty = true })
	if srv.client != nil {
		srv.client.DidOpen(path, b.ed.Buffer().Text())
		st.open = true
	}
}

// waitLSP waits for the server's next event.
func waitLSP(srv *lspServer) tea.Cmd {
	events := srv.client.Events()
	return func() tea.Msg {
		select {
		case ev := <-events:
			return lspEventMsg{srv, ev}
		case <-srv.stop:
			return nil
		}
	}
}

// lspMsg handles the servers' news.
func (m Model) lspMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case lspStartedMsg:
		srv := msg.srv
		srv.starting = false
		if msg.err != nil {
			m.message, m.isError = msg.err.Error(), true
			return m, nil
		}
		select {
		case <-srv.stop: // closed while starting
			msg.client.Close()
			return m, nil
		default:
		}
		srv.client = msg.client
		// The buffers opened while it started, as they are now.
		for _, b := range m.ses.bufs {
			if st := b.lsp; st != nil && st.srv == srv && !st.open {
				srv.client.DidOpen(st.path, b.ed.Buffer().Text())
				st.open, st.dirty = true, false
			}
		}
		return m, waitLSP(srv)
	case lspEventMsg:
		srv := msg.srv
		if msg.ev.Err != nil {
			srv.client = nil
			for _, b := range m.ses.bufs {
				if b.lsp != nil && b.lsp.srv == srv {
					b.lsp.diags, b.lsp.open = nil, false
				}
			}
			m.message, m.isError = msg.ev.Err.Error(), true
			return m, nil
		}
		if len(msg.ev.Edits) > 0 { // after a code action's command
			m = m.applyEdits(msg.ev.Edits, "ação")
			return m, waitLSP(srv)
		}
		for _, b := range m.ses.bufs {
			if b.lsp != nil && b.lsp.srv == srv && sameFile(b.lsp.path, msg.ev.Path) {
				b.lsp.diags = msg.ev.Diagnostics
			}
		}
		return m, waitLSP(srv)
	}
	return m, nil
}

// syncLSP sends the current buffer's text to its server when it changed.
func (m Model) syncLSP() {
	if st := m.lsp; st != nil && st.srv.client != nil && st.open && st.dirty {
		st.srv.client.DidChange(st.path, m.ed.Buffer().Text())
		st.dirty = false
	}
}

// closeLSP stops the servers.
func (m Model) closeLSP() {
	for _, srv := range m.ses.servers {
		close(srv.stop)
		if srv.client != nil {
			srv.client.Close()
		}
	}
}

// marks are the diagnostics' severities on line (length runes long), by
// rune; 0 where there is none, and the most severe wins. Empty ranges
// still mark one cell.
func (st *lspState) marks(line, length int) []lsp.Severity {
	if st == nil {
		return nil
	}
	var out []lsp.Severity
	for _, d := range st.diags {
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
func (st *lspState) worst(line int) *lsp.Diagnostic {
	if st == nil {
		return nil
	}
	var w *lsp.Diagnostic
	for i, d := range st.diags {
		if d.Start.Line <= line && line <= d.End.Line && (w == nil || d.Severity < w.Severity) {
			w = &st.diags[i]
		}
	}
	return w
}

// worst is the current buffer's (see lspState.worst).
func (m Model) worst(line int) *lsp.Diagnostic { return m.lsp.worst(line) }
