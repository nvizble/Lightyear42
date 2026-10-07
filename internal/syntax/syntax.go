// Package syntax highlights source code with Tree-sitter. A Highlighter
// keeps the syntax tree of one document in step with the editor's changes
// (reparsing incrementally, reusing what didn't change) and tells the
// renderer the class of each character: keyword, string, comment...
//
// The grammars are the official Tree-sitter ones, compiled with cgo; the
// highlight queries in queries/ come from the same repositories.
package syntax

import (
	"cmp"
	"embed"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
	"unsafe"

	ts "github.com/tree-sitter/go-tree-sitter"
	tsc "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tscpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tsgo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tspython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tsrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// Class is how a piece of code is highlighted.
type Class uint8

// Classes, from the queries' capture names (see classOf).
const (
	Plain Class = iota
	Keyword
	String
	Comment
	Number // numbers and constants
	Function
	Type
)

//go:embed queries/*.scm
var queries embed.FS

type language struct {
	grammar func() unsafe.Pointer
	queries []string // files in queries/, in order
}

var (
	langC   = language{tsc.Language, []string{"c.scm"}}
	langCpp = language{tscpp.Language, []string{"c.scm", "cpp.scm"}}

	languages = map[string]language{
		".c": langC, ".h": langC,
		".cpp": langCpp, ".cc": langCpp, ".cxx": langCpp, ".hpp": langCpp, ".hh": langCpp, ".hxx": langCpp,
		".go": {tsgo.Language, []string{"go.scm"}},
		".py": {tspython.Language, []string{"python.scm"}},
		".rs": {tsrust.Language, []string{"rust.scm"}},
	}
)

// Highlighter keeps the syntax tree of one document in step with its edits.
// It holds C memory: call Close when done.
type Highlighter struct {
	parser  *ts.Parser
	tree    *ts.Tree
	query   *ts.Query
	classes []Class // by capture index
	src     []byte
	lines   []int // byte offset where each line starts
	stale   bool  // edited since the last parse
}

// For returns a highlighter for the file at path holding text, or nil when
// its language isn't supported.
func For(path, text string) *Highlighter {
	lang, ok := languages[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return nil
	}
	l := ts.NewLanguage(lang.grammar())
	var src strings.Builder
	for _, name := range lang.queries {
		b, _ := queries.ReadFile("queries/" + name)
		src.Write(b)
	}
	query, qerr := ts.NewQuery(l, src.String())
	if qerr != nil {
		return nil // the embedded queries are tested; this can't happen
	}
	parser := ts.NewParser()
	if err := parser.SetLanguage(l); err != nil {
		query.Close()
		parser.Close()
		return nil
	}
	h := &Highlighter{parser: parser, query: query, src: []byte(text)}
	for _, name := range query.CaptureNames() {
		h.classes = append(h.classes, classOf(name))
	}
	h.index()
	h.tree = parser.Parse(h.src, nil)
	return h
}

// Close frees the parser, the tree and the query.
func (h *Highlighter) Close() {
	h.tree.Close()
	h.query.Close()
	h.parser.Close()
}

// Edit applies one change from the editor (see editor.Editor.OnChange).
// The tree is reparsed, reusing what didn't change, when next asked for
// highlights.
func (h *Highlighter) Edit(c editor.Change) {
	start, oldEnd := h.offset(c.Start), h.offset(c.OldEnd)
	newEnd := start + len(c.Text)
	endColumn := start - h.lines[c.Start.Line] + len(c.Text)
	if i := strings.LastIndexByte(c.Text, '\n'); i >= 0 {
		endColumn = len(c.Text) - i - 1
	}
	h.tree.Edit(&ts.InputEdit{
		StartByte:      uint(start),
		OldEndByte:     uint(oldEnd),
		NewEndByte:     uint(newEnd),
		StartPosition:  ts.Point{Row: uint(c.Start.Line), Column: uint(start - h.lines[c.Start.Line])},
		OldEndPosition: ts.Point{Row: uint(c.OldEnd.Line), Column: uint(oldEnd - h.lines[c.OldEnd.Line])},
		NewEndPosition: ts.Point{Row: uint(c.NewEnd.Line), Column: uint(endColumn)},
	})
	h.src = slices.Concat(h.src[:start], []byte(c.Text), h.src[oldEnd:])
	h.index()
	h.stale = true
}

// Lines returns the class of each character (rune) of lines first..last.
func (h *Highlighter) Lines(first, last int) [][]Class {
	last = min(last, len(h.lines)-1)
	if first < 0 || first > last {
		return nil
	}
	if h.stale {
		if tree := h.parser.Parse(h.src, h.tree); tree != nil {
			h.tree.Close()
			h.tree = tree
		}
		h.stale = false
	}
	from, to := h.lines[first], h.lineEnd(last)

	type capture struct {
		start, end, pattern int
		class               Class
	}
	var caps []capture
	qc := ts.NewQueryCursor()
	defer qc.Close()
	qc.SetByteRange(uint(from), uint(to))
	it := qc.Captures(h.query, h.tree.RootNode(), h.src)
	for m, i := it.Next(); m != nil; m, i = it.Next() {
		c := m.Captures[i]
		caps = append(caps, capture{int(c.Node.StartByte()), int(c.Node.EndByte()), int(m.PatternIndex), h.classes[c.Index]})
	}
	// Paint outer nodes before the ones inside them, and patterns in query
	// order: the innermost node and, on the same node, the later pattern win.
	slices.SortFunc(caps, func(a, b capture) int {
		return cmp.Or(cmp.Compare(a.start, b.start), cmp.Compare(b.end, a.end), cmp.Compare(a.pattern, b.pattern))
	})
	paint := make([]Class, to-from) // by byte
	for _, c := range caps {
		for b := max(c.start, from); b < min(c.end, to); b++ {
			paint[b-from] = c.class
		}
	}

	out := make([][]Class, 0, last-first+1)
	for line := first; line <= last; line++ {
		var classes []Class
		for b, end := h.lines[line], h.lineEnd(line); b < end; {
			classes = append(classes, paint[b-from])
			_, size := utf8.DecodeRune(h.src[b:end])
			b += size
		}
		out = append(out, classes)
	}
	return out
}

// index finds where each line starts.
func (h *Highlighter) index() {
	h.lines = append(h.lines[:0], 0)
	for i, b := range h.src {
		if b == '\n' {
			h.lines = append(h.lines, i+1)
		}
	}
}

// lineEnd is the byte offset where line ends (before its line break).
func (h *Highlighter) lineEnd(line int) int {
	if line+1 < len(h.lines) {
		return h.lines[line+1] - 1
	}
	return len(h.src)
}

// offset is the byte offset of p (a rune column) in the current text.
func (h *Highlighter) offset(p editor.Position) int {
	b, end := h.lines[p.Line], h.lineEnd(p.Line)
	for n := 0; n < p.Column && b < end; n++ {
		_, size := utf8.DecodeRune(h.src[b:end])
		b += size
	}
	return b
}

// classOf maps a capture name ("keyword", "function.method",
// "string.special"...) to a class; the part before the first dot decides.
func classOf(capture string) Class {
	name, _, _ := strings.Cut(capture, ".")
	switch name {
	case "keyword", "include", "conditional", "repeat", "exception", "storageclass":
		return Keyword
	case "string", "escape", "char":
		return String
	case "comment":
		return Comment
	case "number", "float", "constant", "boolean":
		return Number
	case "function", "method":
		return Function
	case "type", "constructor", "module", "namespace":
		return Type
	}
	return Plain // variables, properties, operators, punctuation
}
