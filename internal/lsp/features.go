package lsp

import (
	"cmp"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf16"
)

// Location is a place in a file.
type Location struct {
	Path string
	Pos  Pos
}

// Item is a completion suggestion.
type Item struct {
	Label  string // shown in the list
	Detail string // e.g. the type or signature
	Text   string // what gets inserted
	// Start is where the text being completed begins, when the server says
	// (HasStart); otherwise the caller uses the word before the cursor.
	Start    Pos
	HasStart bool
}

// Hover is the server's description of what is at p, as plain text ("" when
// there is nothing).
func (c *Client) Hover(ctx context.Context, p Pos) (string, error) {
	var res *struct {
		Contents json.RawMessage `json:"contents"`
	}
	if err := c.conn.call(ctx, "textDocument/hover", c.at(p), &res); err != nil || res == nil {
		return "", err
	}
	return hoverText(res.Contents), nil
}

// hoverText flattens MarkupContent, MarkedString or []MarkedString, and
// drops Markdown code fences.
func hoverText(raw json.RawMessage) string {
	var parts []string
	var add func(json.RawMessage)
	add = func(raw json.RawMessage) {
		var s string
		var obj struct {
			Value string `json:"value"`
		}
		var list []json.RawMessage
		switch {
		case json.Unmarshal(raw, &s) == nil:
			parts = append(parts, s)
		case json.Unmarshal(raw, &list) == nil:
			for _, item := range list {
				add(item)
			}
		case json.Unmarshal(raw, &obj) == nil:
			parts = append(parts, obj.Value)
		}
	}
	add(raw)
	var lines []string
	for _, line := range strings.Split(strings.Join(parts, "\n\n"), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "```") {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// Definition is where the symbol at p is defined.
func (c *Client) Definition(ctx context.Context, p Pos) ([]Location, error) {
	var raw json.RawMessage
	if err := c.conn.call(ctx, "textDocument/definition", c.at(p), &raw); err != nil {
		return nil, err
	}
	// Location, []Location or []LocationLink.
	type place struct {
		URI                  string   `json:"uri"`
		Range                lspRange `json:"range"`
		TargetURI            string   `json:"targetUri"`
		TargetSelectionRange lspRange `json:"targetSelectionRange"`
	}
	var places []place
	if json.Unmarshal(raw, &places) != nil {
		var one place
		if json.Unmarshal(raw, &one) != nil || one.URI == "" {
			return nil, nil
		}
		places = []place{one}
	}
	var out []Location
	for _, pl := range places {
		uri, r := pl.URI, pl.Range
		if pl.TargetURI != "" {
			uri, r = pl.TargetURI, pl.TargetSelectionRange
		}
		if uri == c.uri {
			out = append(out, Location{Path: uriPath(uri), Pos: c.fromLSP(r.Start)})
			continue
		}
		path := uriPath(uri)
		out = append(out, Location{Path: path, Pos: Pos{r.Start.Line, c.fileColumn(path, r.Start)}})
	}
	return out, nil
}

// Completion lists what could be typed at p, best first.
func (c *Client) Completion(ctx context.Context, p Pos) ([]Item, error) {
	var raw json.RawMessage
	if err := c.conn.call(ctx, "textDocument/completion", c.at(p), &raw); err != nil {
		return nil, err
	}
	type textEdit struct {
		NewText string    `json:"newText"`
		Range   *lspRange `json:"range"`
		Insert  *lspRange `json:"insert"` // InsertReplaceEdit
	}
	type item struct {
		Label            string    `json:"label"`
		Detail           string    `json:"detail"`
		InsertText       string    `json:"insertText"`
		InsertTextFormat int       `json:"insertTextFormat"`
		SortText         string    `json:"sortText"`
		TextEdit         *textEdit `json:"textEdit"`
	}
	var list struct {
		Items []item `json:"items"`
	}
	if json.Unmarshal(raw, &list.Items) != nil {
		_ = json.Unmarshal(raw, &list) // CompletionList
	}
	slices.SortStableFunc(list.Items, func(a, b item) int {
		return cmp.Compare(cmp.Or(a.SortText, a.Label), cmp.Or(b.SortText, b.Label))
	})
	out := make([]Item, 0, len(list.Items))
	for _, it := range list.Items {
		text := cmp.Or(it.InsertText, it.Label)
		var start *lspRange
		if it.TextEdit != nil {
			text = it.TextEdit.NewText
			start = cmp.Or(it.TextEdit.Range, it.TextEdit.Insert)
		}
		if it.InsertTextFormat == 2 {
			text = plainSnippet(text)
		}
		// clangd marks items that would add an #include with "•".
		label := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(it.Label), "•"))
		i := Item{Label: label, Detail: it.Detail, Text: text}
		if start != nil {
			i.Start, i.HasStart = c.fromLSP(start.Start), true
		}
		out = append(out, i)
	}
	return out, nil
}

var snippetVar = regexp.MustCompile(`\$\{\d+(?::([^}]*))?\}|\$\d+`)

// plainSnippet turns a snippet ("f(${1:x})$0") into plain text ("f(x)").
func plainSnippet(s string) string {
	return snippetVar.ReplaceAllString(s, "$1")
}

// at is the params of a request about position p of the document.
func (c *Client) at(p Pos) map[string]any {
	return map[string]any{"textDocument": map[string]string{"uri": c.uri}, "position": c.toLSP(p)}
}

// toLSP converts a rune column to the server's unit.
func (c *Client) toLSP(p Pos) lspPos {
	if !c.utf16 {
		return lspPos{p.Line, p.Col}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if p.Line >= len(c.lines) {
		return lspPos{p.Line, p.Col}
	}
	units := 0
	for i, r := range []rune(c.lines[p.Line]) {
		if i >= p.Col {
			break
		}
		units += len(utf16.Encode([]rune{r}))
	}
	return lspPos{p.Line, units}
}

// fileColumn converts a position in another file, read from disk.
func (c *Client) fileColumn(path string, p lspPos) int {
	if !c.utf16 {
		return p.Character
	}
	data, err := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")
	if err != nil || p.Line >= len(lines) {
		return p.Character
	}
	col, units := 0, 0
	for _, r := range lines[p.Line] {
		if units >= p.Character {
			break
		}
		units += len(utf16.Encode([]rune{r}))
		col++
	}
	return col
}

// uriPath is the file path of a file:// URI.
func uriPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	path := u.Path
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:] // Windows: /C:/x → C:/x
	}
	return filepath.FromSlash(path)
}
