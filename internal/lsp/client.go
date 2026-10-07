// Package lsp is a small Language Server Protocol client for the embedded
// editor: it starts the server for a file, keeps that document in sync and
// reports the server's diagnostics. It knows nothing about the TUI.
//
// Positions are lines and rune columns, like the editor's; the client
// converts them to the server's unit (UTF-32 when the server accepts it,
// UTF-16 otherwise).
package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

// Pos is a position in the document: line and rune column, from 0.
type Pos struct{ Line, Col int }

// Severity of a diagnostic, numbered as in LSP.
type Severity int

// Severities.
const (
	Error Severity = iota + 1
	Warning
	Information
	Hint
)

// Diagnostic is a problem the server found in the document.
type Diagnostic struct {
	Start, End Pos
	Severity   Severity
	Message    string
}

// Event is news from the server: the document's diagnostics, or Err when
// the server stopped.
type Event struct {
	Diagnostics []Diagnostic
	Err         error
}

// Client talks to one language server about one open document.
type Client struct {
	server Server
	cmd    *exec.Cmd
	conn   *conn
	stderr *tail
	exited chan struct{} // closed when the server process ends
	closed bool

	uri     string
	utf16   bool // the server counts columns in UTF-16 code units
	mu      sync.Mutex
	version int
	lines   []string // the document as last sent, to convert positions

	events chan Event
}

// Start runs the server for the file at path, holding text, and opens the
// document. A server that is missing or stops while starting is an error
// that tells how to install it.
func Start(ctx context.Context, s Server, path, text string) (*Client, error) {
	argv, err := s.command()
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root := findRoot(filepath.Dir(abs), s.RootMarkers)

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = root
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	c := &Client{server: s, cmd: cmd, stderr: &tail{}, exited: make(chan struct{}), uri: fileURI(abs), events: make(chan Event, 1)}
	cmd.Stderr = c.stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%s não iniciou: %w — %s", s.Name, err, s.Hint)
	}
	go func() {
		_ = cmd.Wait()
		close(c.exited)
	}()

	c.conn = newConn(stdin)
	c.conn.onNotify = c.notification
	c.conn.onRequest = serverRequest
	go c.conn.read(stdout)

	// Generous: on macOS the first launch of clangd by a new lightyear
	// binary can take ~15s (the system checks it); later ones are instant.
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var res struct {
		Capabilities struct {
			PositionEncoding string `json:"positionEncoding"`
		} `json:"capabilities"`
	}
	params := map[string]any{
		"processId":  os.Getpid(),
		"clientInfo": map[string]string{"name": "lightyear"},
		"rootUri":    fileURI(root),
		"workspaceFolders": []map[string]string{
			{"uri": fileURI(root), "name": filepath.Base(root)},
		},
		"capabilities": map[string]any{
			"general": map[string]any{"positionEncodings": []string{"utf-32", "utf-16"}},
			"textDocument": map[string]any{
				"publishDiagnostics": map[string]any{},
			},
		},
	}
	if s.Options != nil {
		params["initializationOptions"] = s.Options
	}
	if err := c.conn.call(ctx, "initialize", params, &res); err != nil {
		c.kill()
		return nil, c.startError(err)
	}
	c.utf16 = res.Capabilities.PositionEncoding != "utf-32"
	_ = c.conn.notify("initialized", map[string]any{})

	c.mu.Lock()
	c.version = 1
	c.lines = strings.Split(text, "\n")
	c.mu.Unlock()
	_ = c.conn.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": c.uri, "languageId": s.LanguageID, "version": 1, "text": text},
	})
	go c.watch()
	return c, nil
}

// Events delivers diagnostics as they change, and the server stopping.
func (c *Client) Events() <-chan Event { return c.events }

// Server is the server this client runs.
func (c *Client) Server() Server { return c.server }

// DidChange sends the document's new text (the whole text: full sync).
func (c *Client) DidChange(text string) {
	c.mu.Lock()
	c.version++
	version := c.version
	c.lines = strings.Split(text, "\n")
	c.mu.Unlock()
	_ = c.conn.notify("textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": c.uri, "version": version},
		"contentChanges": []map[string]string{{"text": text}},
	})
}

// Close closes the document and stops the server, politely first.
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	_ = c.conn.notify("textDocument/didClose", map[string]any{"textDocument": map[string]string{"uri": c.uri}})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if c.conn.call(ctx, "shutdown", nil, nil) == nil {
		_ = c.conn.notify("exit", nil)
	}
	select {
	case <-c.exited:
	case <-time.After(2 * time.Second):
		c.kill()
	}
}

func (c *Client) kill() {
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

// watch reports the server stopping on its own.
func (c *Client) watch() {
	<-c.exited
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if !closed {
		c.emit(Event{Err: fmt.Errorf("o %s parou%s", c.server.Name, c.stderr.reason())})
	}
}

// startError explains a server that didn't start, with the install hint.
func (c *Client) startError(err error) error {
	select {
	case <-c.exited:
	case <-time.After(500 * time.Millisecond):
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("o %s não respondeu ao iniciar — %s", c.server.Name, c.server.Hint)
	}
	return fmt.Errorf("o %s não iniciou%s — %s", c.server.Name, c.stderr.reason(), c.server.Hint)
}

// emit delivers an event, replacing one the editor hasn't taken yet
// (diagnostics supersede each other).
func (c *Client) emit(ev Event) {
	for {
		select {
		case c.events <- ev:
			return
		default:
		}
		select {
		case <-c.events:
		default:
		}
	}
}

func (c *Client) notification(method string, params json.RawMessage) {
	if method != "textDocument/publishDiagnostics" {
		return
	}
	var p struct {
		URI         string `json:"uri"`
		Diagnostics []struct {
			Range    lspRange `json:"range"`
			Severity int      `json:"severity"`
			Message  string   `json:"message"`
		} `json:"diagnostics"`
	}
	if json.Unmarshal(params, &p) != nil || p.URI != c.uri {
		return
	}
	ds := make([]Diagnostic, 0, len(p.Diagnostics))
	for _, d := range p.Diagnostics {
		sev := Severity(d.Severity)
		if sev < Error || sev > Hint {
			sev = Error
		}
		ds = append(ds, Diagnostic{Start: c.fromLSP(d.Range.Start), End: c.fromLSP(d.Range.End), Severity: sev, Message: d.Message})
	}
	c.emit(Event{Diagnostics: ds})
}

// serverRequest answers what servers commonly ask the client.
func serverRequest(method string, params json.RawMessage) (any, error) {
	switch method {
	case "workspace/configuration":
		var p struct {
			Items []json.RawMessage `json:"items"`
		}
		_ = json.Unmarshal(params, &p)
		return make([]any, len(p.Items)), nil
	case "client/registerCapability", "client/unregisterCapability", "window/workDoneProgress/create", "window/showMessageRequest":
		return nil, nil
	}
	return nil, errors.New("método não suportado: " + method)
}

type lspPos struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPos `json:"start"`
	End   lspPos `json:"end"`
}

// fromLSP converts a server position to a rune column.
func (c *Client) fromLSP(p lspPos) Pos {
	if !c.utf16 {
		return Pos{p.Line, p.Character}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if p.Line >= len(c.lines) {
		return Pos{p.Line, p.Character}
	}
	col, units := 0, 0
	for _, r := range c.lines[p.Line] {
		if units >= p.Character {
			break
		}
		units += len(utf16.Encode([]rune{r}))
		col++
	}
	return Pos{p.Line, col}
}

// fileURI is the file:// URI of an absolute path.
func fileURI(path string) string {
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path // Windows: C:/x → /C:/x
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

// findRoot is the nearest directory from dir up holding one of markers,
// or dir itself.
func findRoot(dir string, markers []string) string {
	for d := dir; ; {
		for _, m := range markers {
			if _, err := os.Stat(filepath.Join(d, m)); err == nil {
				return d
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir
		}
		d = parent
	}
}

// tail keeps the end of the server's stderr, to explain why it stopped.
type tail struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > 4096 {
		t.buf = t.buf[len(t.buf)-4096:]
	}
	return len(p), nil
}

// reason is the last line the server wrote to stderr, as ": line".
func (t *tail) reason() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	lines := bytes.Split(bytes.TrimSpace(t.buf), []byte("\n"))
	if last := strings.TrimSpace(string(lines[len(lines)-1])); last != "" {
		return ": " + last
	}
	return ""
}
