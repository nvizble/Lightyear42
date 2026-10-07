package lsp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// Edits from the server: rename, references, code actions and formatting.

// TextEdit replaces the text from Start to End with Text.
type TextEdit struct {
	Start, End Pos
	Text       string
}

// FileEdit is the edits to one file.
type FileEdit struct {
	Path  string
	Edits []TextEdit
}

// Action is a code action: what to show, and what it does — edits, a
// command for the server to run (which then sends its edits back as an
// Event), or both.
type Action struct {
	Title   string
	Edits   []FileEdit
	command json.RawMessage // a Command ({command, arguments}), or nil
}

type lspEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}

type workspaceEdit struct {
	Changes         map[string][]lspEdit `json:"changes"`
	DocumentChanges []struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Edits []lspEdit `json:"edits"`
	} `json:"documentChanges"`
}

// Rename asks the server for the edits that rename the symbol at p to name,
// across the project.
func (c *Client) Rename(ctx context.Context, path string, p Pos, name string) ([]FileEdit, error) {
	params := c.at(path, p)
	params["newName"] = name
	var we *workspaceEdit
	if err := c.conn.call(ctx, "textDocument/rename", params, &we); err != nil || we == nil {
		return nil, err
	}
	return c.fileEdits(*we), nil
}

// References are the places that use the symbol at p, its declaration
// included.
func (c *Client) References(ctx context.Context, path string, p Pos) ([]Location, error) {
	params := c.at(path, p)
	params["context"] = map[string]bool{"includeDeclaration": true}
	var res []struct {
		URI   string   `json:"uri"`
		Range lspRange `json:"range"`
	}
	if err := c.conn.call(ctx, "textDocument/references", params, &res); err != nil {
		return nil, err
	}
	out := make([]Location, 0, len(res))
	seen := map[Location]bool{}
	for _, r := range res {
		l := Location{Path: uriPath(r.URI), Pos: c.converter(r.URI)(r.Range.Start)}
		// The same file can come under two paths (macOS's /var is
		// /private/var): list it once.
		key := l
		if dir, err := filepath.EvalSymlinks(filepath.Dir(l.Path)); err == nil {
			key.Path = filepath.Join(dir, filepath.Base(l.Path)) // the file may not be saved yet
		}
		if !seen[key] {
			seen[key] = true
			out = append(out, l)
		}
	}
	return out, nil
}

// CodeActions are the fixes and refactors the server offers at p (on the
// line's diagnostics too).
func (c *Client) CodeActions(ctx context.Context, path string, p Pos, diags []Diagnostic) ([]Action, error) {
	at := c.toLSP(pathURI(path), p)
	raw := []json.RawMessage{}
	for _, d := range diags {
		if d.raw != nil {
			raw = append(raw, d.raw)
		}
	}
	params := map[string]any{
		"textDocument": map[string]string{"uri": pathURI(path)},
		"range":        lspRange{Start: at, End: at},
		"context":      map[string]any{"diagnostics": raw},
	}
	var res []struct {
		Title    string          `json:"title"`
		Command  json.RawMessage `json:"command"` // a string in a Command, an object in a CodeAction
		Edit     *workspaceEdit  `json:"edit"`
		Disabled *struct{}       `json:"disabled"`
	}
	var whole []json.RawMessage
	if err := c.conn.call(ctx, "textDocument/codeAction", params, &whole); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(whole)
	_ = json.Unmarshal(b, &res)
	var out []Action
	for i, r := range res {
		if r.Disabled != nil {
			continue
		}
		a := Action{Title: r.Title}
		if r.Edit != nil {
			a.Edits = c.fileEdits(*r.Edit)
		}
		var name string
		switch {
		case json.Unmarshal(r.Command, &name) == nil:
			a.command = whole[i] // the item itself is a Command
		case len(r.Command) > 0 && string(r.Command) != "null":
			a.command = r.Command
		}
		out = append(out, a)
	}
	return out, nil
}

// Run has the server run an action's command (its edits come back as an
// Event).
func (c *Client) Run(ctx context.Context, a Action) error {
	if a.command == nil {
		return nil
	}
	var cmd struct {
		Command   string            `json:"command"`
		Arguments []json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(a.command, &cmd); err != nil {
		return err
	}
	return c.conn.call(ctx, "workspace/executeCommand", cmd, nil)
}

// Format asks for the edits that format the document at path.
func (c *Client) Format(ctx context.Context, path string, tabSize int) ([]TextEdit, error) {
	uri := pathURI(path)
	params := map[string]any{
		"textDocument": map[string]string{"uri": uri},
		"options":      map[string]any{"tabSize": tabSize, "insertSpaces": false},
	}
	var res []lspEdit
	if err := c.conn.call(ctx, "textDocument/formatting", params, &res); err != nil {
		return nil, err
	}
	return c.textEdits(uri, res), nil
}

// fileEdits converts a WorkspaceEdit (changes or documentChanges).
func (c *Client) fileEdits(we workspaceEdit) []FileEdit {
	var out []FileEdit
	for uri, edits := range we.Changes {
		out = append(out, FileEdit{Path: uriPath(uri), Edits: c.textEdits(uri, edits)})
	}
	for _, dc := range we.DocumentChanges {
		if dc.TextDocument.URI != "" {
			out = append(out, FileEdit{Path: uriPath(dc.TextDocument.URI), Edits: c.textEdits(dc.TextDocument.URI, dc.Edits)})
		}
	}
	return out
}

func (c *Client) textEdits(uri string, edits []lspEdit) []TextEdit {
	conv := c.converter(uri)
	out := make([]TextEdit, 0, len(edits))
	for _, e := range edits {
		out = append(out, TextEdit{Start: conv(e.Range.Start), End: conv(e.Range.End), Text: e.NewText})
	}
	return out
}

// converter turns positions in the document at uri into rune columns,
// reading the file from disk when it isn't open.
func (c *Client) converter(uri string) func(lspPos) Pos {
	c.mu.Lock()
	_, open := c.docs[uri]
	c.mu.Unlock()
	if open || !c.utf16 {
		return func(p lspPos) Pos { return c.fromLSP(uri, p) }
	}
	data, _ := os.ReadFile(uriPath(uri))
	lines := strings.Split(string(data), "\n")
	return func(p lspPos) Pos {
		if p.Line >= len(lines) {
			return Pos{p.Line, p.Character}
		}
		col, units := 0, 0
		for _, r := range lines[p.Line] {
			if units >= p.Character {
				break
			}
			units += len(utf16.Encode([]rune{r}))
			col++
		}
		return Pos{p.Line, col}
	}
}
