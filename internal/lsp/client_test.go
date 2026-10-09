package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

// The test binary doubles as a fake language server (FAKE_LSP set): it
// reports a diagnostic on every "ERR" in the document.
func TestMain(m *testing.M) {
	switch os.Getenv("FAKE_LSP") {
	case "":
		os.Exit(m.Run())
	case "die":
		fmt.Fprintln(os.Stderr, "error: Unknown binary 'rust-analyzer' in official toolchain")
		os.Exit(1)
	default:
		fakeServer()
		os.Exit(0)
	}
}

func rng(l1, c1, l2, c2 int) map[string]any {
	return map[string]any{"start": map[string]int{"line": l1, "character": c1}, "end": map[string]int{"line": l2, "character": c2}}
}

func edit(l1, c1, l2, c2 int, text string) map[string]any {
	return map[string]any{"range": rng(l1, c1, l2, c2), "newText": text}
}

func fakeServer() {
	in := bufio.NewReader(os.Stdin)
	write := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Printf("Content-Length: %d\r\n\r\n%s", len(b), b)
	}
	uri := ""
	publish := func(text string) {
		var ds []any
		for line, s := range strings.Split(text, "\n") {
			if i := strings.Index(s, "ERR"); i >= 0 {
				col := len(utf16.Encode([]rune(s[:i])))
				if os.Getenv("FAKE_ENC") == "utf-32" {
					col = len([]rune(s[:i]))
				}
				ds = append(ds, map[string]any{
					"range":    map[string]any{"start": map[string]int{"line": line, "character": col}, "end": map[string]int{"line": line, "character": col + 3}},
					"severity": 2, "message": "erro aqui",
				})
			}
		}
		write(map[string]any{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics", "params": map[string]any{"uri": uri, "diagnostics": ds}})
	}
	for {
		body, err := readMessage(in)
		if err != nil {
			return
		}
		var m struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
		}
		_ = json.Unmarshal(body, &m)
		var p struct {
			Position struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"position"`
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		_ = json.Unmarshal(m.Params, &p)
		switch m.Method {
		case "initialize":
			enc := os.Getenv("FAKE_ENC")
			if enc == "" {
				enc = "utf-16"
			}
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": map[string]any{"capabilities": map[string]any{"positionEncoding": enc}}})
		case "initialized":
			// Servers ask the client things; it must answer.
			write(map[string]any{"jsonrpc": "2.0", "id": 99, "method": "workspace/configuration", "params": map[string]any{"items": []any{map[string]any{}}}})
		case "textDocument/didOpen":
			uri = p.TextDocument.URI
			if os.Getenv("FAKE_LSP") == "crash" {
				os.Exit(2)
			}
			publish(p.TextDocument.Text)
		case "textDocument/didChange":
			uri = p.TextDocument.URI
			publish(p.ContentChanges[0].Text)
		case "textDocument/hover":
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": map[string]any{
				"contents": map[string]string{"kind": "plaintext", "value": fmt.Sprintf("hover %d:%d", p.Position.Line, p.Position.Character)},
			}})
		case "textDocument/definition":
			pos := func(l, c int) map[string]any {
				return map[string]any{"start": map[string]int{"line": l, "character": c}, "end": map[string]int{"line": l, "character": c}}
			}
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": []any{
				map[string]any{"uri": uri, "range": pos(2, 2)}, // after 😀: rune column 1
				map[string]any{"targetUri": "file:///tmp/outro%20arquivo.c", "targetSelectionRange": pos(3, 4)},
			}})
		case "textDocument/completion":
			l, c := p.Position.Line, p.Position.Character
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": map[string]any{"isIncomplete": false, "items": []any{
				map[string]any{"label": " • strlen(const char *s)", "detail": "size_t", "sortText": "2",
					"textEdit": map[string]any{"newText": "strlen", "range": map[string]any{
						"start": map[string]int{"line": l, "character": c - 2}, "end": map[string]int{"line": l, "character": c}}}},
				map[string]any{"label": "printf", "insertText": "printf(${1:fmt})$0", "insertTextFormat": 2, "sortText": "1"},
			}}})
		case "textDocument/rename":
			var q struct {
				NewName string `json:"newName"`
			}
			_ = json.Unmarshal(m.Params, &q)
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": map[string]any{"changes": map[string]any{
				p.TextDocument.URI: []any{edit(0, 0, 0, 1, q.NewName)},
			}}})
		case "textDocument/references":
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": []any{
				map[string]any{"uri": p.TextDocument.URI, "range": rng(1, 2, 1, 3)},
			}})
		case "textDocument/codeAction":
			var q struct {
				Context struct {
					Diagnostics []json.RawMessage `json:"diagnostics"`
				} `json:"context"`
			}
			_ = json.Unmarshal(m.Params, &q)
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": []any{
				map[string]any{"title": fmt.Sprintf("corrigir (%d diagnósticos)", len(q.Context.Diagnostics)), "kind": "quickfix",
					"edit": map[string]any{"changes": map[string]any{p.TextDocument.URI: []any{edit(0, 0, 0, 0, ";")}}}},
				map[string]any{"title": "rodar comando", "command": "fake.apply", "arguments": []any{p.TextDocument.URI}},
				map[string]any{"title": "desligada", "disabled": map[string]string{"reason": "não"}},
			}})
		case "workspace/executeCommand":
			var q struct {
				Arguments []string `json:"arguments"`
			}
			_ = json.Unmarshal(m.Params, &q)
			write(map[string]any{"jsonrpc": "2.0", "id": 50, "method": "workspace/applyEdit", "params": map[string]any{
				"edit": map[string]any{"documentChanges": []any{map[string]any{
					"textDocument": map[string]any{"uri": q.Arguments[0], "version": 1},
					"edits":        []any{edit(0, 0, 0, 0, "/* cmd */")},
				}}},
			}})
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": nil})
		case "textDocument/formatting":
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": []any{edit(0, 0, 0, 0, "// fmt\n")}})
		case "shutdown":
			write(map[string]any{"jsonrpc": "2.0", "id": *m.ID, "result": nil})
		case "exit":
			return
		case "":
			if m.ID != nil && *m.ID == 99 && string(m.Result) != "[null]" {
				os.Exit(3) // the configuration answer was wrong
			}
		}
	}
}

// open starts s for the folder of path and opens the document.
func open(s Server, path, text string) (*Client, error) {
	c, err := Start(context.Background(), s, filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	c.DidOpen(path, text)
	return c, nil
}

func fake(t *testing.T, mode string) Server {
	t.Helper()
	t.Setenv("FAKE_LSP", mode)
	return Server{Name: "fake", LanguageID: "c", Commands: [][]string{{os.Args[0]}}, Hint: "instale o fake"}
}

func next(t *testing.T, c *Client) Event {
	t.Helper()
	select {
	case ev := <-c.Events():
		return ev
	case <-time.After(10 * time.Second):
		t.Fatal("o servidor não respondeu")
		return Event{}
	}
}

func TestDiagnosticsInRuneColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.c")
	c, err := open(fake(t, "ok"), path, "ok\nxy ERR\n😀ERR")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ev := next(t, c)
	want := []Diagnostic{
		{Start: Pos{1, 3}, End: Pos{1, 6}, Severity: Warning, Message: "erro aqui"},
		{Start: Pos{2, 1}, End: Pos{2, 4}, Severity: Warning, Message: "erro aqui"}, // 😀 is 2 UTF-16 units, 1 rune
	}
	for i := range ev.Diagnostics {
		ev.Diagnostics[i].raw = nil // compared on its own below
	}
	if fmt.Sprint(ev.Diagnostics) != fmt.Sprint(want) {
		t.Fatalf("diagnostics: %v, esperado %v", ev.Diagnostics, want)
	}

	c.DidChange(path, "ERR")
	if ev := next(t, c); len(ev.Diagnostics) != 1 || ev.Diagnostics[0].Start != (Pos{0, 0}) {
		t.Fatalf("depois da mudança: %+v", ev)
	}
	c.DidChange(path, "tudo certo")
	if ev := next(t, c); ev.Err != nil || len(ev.Diagnostics) != 0 {
		t.Fatalf("sem erros deveria limpar: %+v", ev)
	}
}

func TestHoverDefinitionCompletion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.c")
	c, err := open(fake(t, "ok"), path, "a\nb\n😀st")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx := context.Background()
	// Rune column 3 is after "😀st": 4 UTF-16 units.
	if h, err := c.Hover(ctx, path, Pos{2, 3}); err != nil || h != "hover 2:4" {
		t.Fatalf("hover: %q %v", h, err)
	}
	locs, err := c.Definition(ctx, path, Pos{2, 3})
	want := []Location{{Path: path, Pos: Pos{2, 1}}, {Path: "/tmp/outro arquivo.c", Pos: Pos{3, 4}}}
	if err != nil || fmt.Sprint(locs) != fmt.Sprint(want) {
		t.Fatalf("definition: %v %v", locs, err)
	}
	items, err := c.Completion(ctx, path, Pos{2, 3})
	if err != nil || len(items) != 2 {
		t.Fatalf("completion: %+v %v", items, err)
	}
	// Sorted by sortText; snippets become plain text; "•" and spaces go.
	if items[0].Label != "printf" || items[0].Text != "printf(fmt)" || items[0].HasStart {
		t.Fatalf("1º item: %+v", items[0])
	}
	if it := items[1]; it.Label != "strlen(const char *s)" || it.Text != "strlen" || it.Detail != "size_t" || !it.HasStart || it.Start != (Pos{2, 1}) {
		t.Fatalf("2º item: %+v", it)
	}
}

func TestEditsFromTheServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.c")
	c, err := open(fake(t, "ok"), path, "😀x ERR\nyyyy")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ev := next(t, c) // the diagnostic, for the code action
	ctx := context.Background()

	edits, err := c.Rename(ctx, path, Pos{0, 1}, "novo")
	if err != nil || len(edits) != 1 || edits[0].Path != path || fmt.Sprint(edits[0].Edits) != fmt.Sprint([]TextEdit{{Start: Pos{0, 0}, End: Pos{0, 1}, Text: "novo"}}) {
		t.Fatalf("rename: %+v %v", edits, err)
	}
	refs, err := c.References(ctx, path, Pos{0, 1})
	if err != nil || len(refs) != 1 || refs[0].Pos != (Pos{1, 2}) {
		t.Fatalf("references: %+v %v", refs, err)
	}
	actions, err := c.CodeActions(ctx, path, Pos{0, 3}, ev.Diagnostics)
	if err != nil || len(actions) != 2 || actions[0].Title != "corrigir (1 diagnósticos)" || len(actions[0].Edits) != 1 || actions[1].command == nil {
		t.Fatalf("code actions (a desligada some; o diagnóstico vai junto): %+v %v", actions, err)
	}
	// A command's edits come back from the server as an Event.
	if err := c.Run(ctx, actions[1]); err != nil {
		t.Fatal(err)
	}
	for {
		ev := next(t, c)
		if len(ev.Edits) == 1 && ev.Edits[0].Edits[0].Text == "/* cmd */" {
			break
		}
	}
	fmtEdits, err := c.Format(ctx, path, 4)
	if err != nil || len(fmtEdits) != 1 || fmtEdits[0].Text != "// fmt\n" {
		t.Fatalf("format: %+v %v", fmtEdits, err)
	}
}

func TestHoverText(t *testing.T) {
	for raw, want := range map[string]string{
		`{"kind":"markdown","value":"` + "```c\\nint x\\n```\\ndoc" + `"}`: "int x\ndoc",
		`"texto"`:                             "texto",
		`["a", {"language":"c","value":"b"}]`: "a\n\nb",
	} {
		if got := hoverText([]byte(raw)); got != want {
			t.Errorf("%s: %q, esperado %q", raw, got, want)
		}
	}
}

// One server, two documents: diagnostics say whose they are.
func TestSeveralDocuments(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.c"), filepath.Join(dir, "b.c")
	c, err := Start(context.Background(), fake(t, "ok"), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.DidOpen(a, "ERR")
	c.DidOpen(b, "ok\nok ERR")
	got := map[string]Pos{}
	for len(got) < 2 {
		ev := next(t, c)
		got[ev.Path] = ev.Diagnostics[0].Start
	}
	if got[a] != (Pos{0, 0}) || got[b] != (Pos{1, 3}) {
		t.Fatalf("diagnostics por arquivo: %v", got)
	}
	c.DidClose(a)
	c.DidChange(a, "ERR") // closed: ignored
	c.DidChange(b, "ERR")
	if ev := next(t, c); ev.Path != b {
		t.Fatalf("só o aberto muda: %+v", ev)
	}
	if Root(Server{RootMarkers: []string{"nada"}}, a) != dir {
		t.Fatal("Root sem marcador é a pasta do arquivo")
	}
}

func TestUTF32IsUsedWhenTheServerAgrees(t *testing.T) {
	t.Setenv("FAKE_ENC", "utf-32")
	c, err := open(fake(t, "ok"), filepath.Join(t.TempDir(), "a.c"), "😀ERR")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if ev := next(t, c); len(ev.Diagnostics) != 1 || ev.Diagnostics[0].Start != (Pos{0, 1}) {
		t.Fatalf("utf-32: %+v", ev)
	}
}

func TestServerProblemsExplainHowToInstall(t *testing.T) {
	dir := t.TempDir()
	_, err := open(fake(t, "die"), filepath.Join(dir, "a.c"), "")
	if err == nil || !strings.Contains(err.Error(), "Unknown binary 'rust-analyzer'") || !strings.Contains(err.Error(), "instale o fake") {
		t.Fatalf("servidor que morre ao iniciar: %v", err)
	}

	missing := Server{Name: "nada", Commands: [][]string{{"lightyear-servidor-que-nao-existe"}}, Hint: "instale o nada"}
	if _, err := open(missing, filepath.Join(dir, "a.c"), ""); err == nil || !strings.Contains(err.Error(), "não encontrado — instale o nada") {
		t.Fatalf("servidor ausente: %v", err)
	}

	c, err := open(fake(t, "crash"), filepath.Join(dir, "a.c"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if ev := next(t, c); ev.Err == nil || !strings.Contains(ev.Err.Error(), "o fake parou") {
		t.Fatalf("servidor que cai depois deveria avisar: %+v", ev)
	}
}

func TestServerFor(t *testing.T) {
	for path, want := range map[string]string{"main.c": "c", "ft.h": "c", "Fixed.hpp": "cpp", "x.go": "go", "a.py": "python", "lib.rs": "rust"} {
		if s, ok := ServerFor(path); !ok || s.LanguageID != want {
			t.Errorf("%s: %+v", path, s)
		}
	}
	if _, ok := ServerFor("Makefile"); ok {
		t.Error("Makefile não tem servidor")
	}
	c, _ := ServerFor("main.c")
	if got := fmt.Sprint(c.Options); !strings.Contains(got, "-Wall -Wextra -Werror") {
		t.Fatalf("o clangd deveria usar as flags da 42: %s", got)
	}
}

func TestFindRootAndURI(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "ex00", "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := findRoot(sub, []string{"go.mod"}); got != root {
		t.Fatalf("raiz: %s", got)
	}
	if got := findRoot(sub, []string{"nada"}); got != sub {
		t.Fatalf("sem marcador fica a pasta do arquivo: %s", got)
	}
	if got := fileURI("/tmp/meu projeto/a.c"); got != "file:///tmp/meu%20projeto/a.c" {
		t.Fatalf("uri: %s", got)
	}
}

// The real clangd: hover, definition and completion on a small program.
func TestClangdFeatures(t *testing.T) {
	strict := needClangd(t)
	path := filepath.Join(t.TempDir(), "main.c")
	text := "int\tadd(int a, int b)\n{\n\treturn (a + b);\n}\n\nint\tcounter;\n\nint\tmain(void)\n{\n\treturn (add(1, cou));\n}\n"
	s, _ := ServerFor(path)
	c, err := open(s, path, text)
	if err != nil {
		clangdFailed(t, strict, err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Line 9 is "\treturn (add(1, cou));": "add" starts at rune 9.
	if h, err := c.Hover(ctx, path, Pos{9, 10}); err != nil || !strings.Contains(h, "int add(int a, int b)") {
		t.Fatalf("hover: %q %v", h, err)
	}
	if locs, err := c.Definition(ctx, path, Pos{9, 10}); err != nil || len(locs) != 1 || locs[0].Pos.Line != 0 {
		t.Fatalf("definition: %+v %v", locs, err)
	}
	items, err := c.Completion(ctx, path, Pos{9, 19})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Text == "counter" {
			return
		}
	}
	t.Fatalf("completion sem counter: %+v", items)
}

// needClangd skips a test of the real clangd when it isn't installed,
// unless LIGHTYEAR_TEST_DOWNLOAD is set where lightyear downloads it: then
// the download is tested for real, and a clangd that doesn't start fails
// the test (strict).
func needClangd(t *testing.T) (strict bool) {
	t.Helper()
	return needServer(t, serverC)
}

// needServer is needClangd for any server.
func needServer(t *testing.T, s Server) (strict bool) {
	t.Helper()
	if s.installed() != nil {
		return false
	}
	if os.Getenv("LIGHTYEAR_TEST_DOWNLOAD") != "" && s.Download != nil {
		return true
	}
	t.Skipf("%s não instalado", s.Name)
	return false
}

// clangdFailed skips (clangd may not run here) or, when strict, fails.
func clangdFailed(t *testing.T, strict bool, err error) {
	t.Helper()
	if strict {
		t.Fatalf("clangd baixado não iniciou: %v", err)
	}
	t.Skipf("clangd não iniciou aqui: %v", err)
}

// The real clangd, when installed: the 42 flags turn an unused variable
// into an error, like the grader.
func TestClangd(t *testing.T) {
	strict := needClangd(t)
	path := filepath.Join(t.TempDir(), "main.c")
	s, _ := ServerFor(path)
	c, err := open(s, path, "int\tmain(void)\n{\n\tint\tx;\n\n\treturn (0);\n}\n")
	if err != nil {
		clangdFailed(t, strict, err)
	}
	defer c.Close()
	for {
		ev := next(t, c)
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
		for _, d := range ev.Diagnostics {
			if strings.Contains(strings.ToLower(d.Message), "unused variable 'x'") && d.Severity == Error && d.Start.Line == 2 {
				return
			}
		}
	}
}

// The real clangd: rename, references, the fix-it for a missing ";" as a
// code action, and formatting with the project's .clang-format.
func TestClangdEdits(t *testing.T) {
	strict := needClangd(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".clang-format"), []byte("BasedOnStyle: LLVM\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "main.c")
	text := "int\tadd(int a, int b)\n{\n\treturn (a + b);\n}\n\nint\tmain(void)\n{\n\tint\tx;\n\n\tx = add(1, 2)\n\treturn (x);\n}\n"
	s, _ := ServerFor(path)
	c, err := open(s, path, text)
	if err != nil {
		clangdFailed(t, strict, err)
	}
	defer c.Close()
	var diags []Diagnostic
	for diags == nil {
		ev := next(t, c)
		for _, d := range ev.Diagnostics {
			if strings.Contains(strings.ToLower(d.Message), "expected ';'") {
				diags = append(diags, d)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	edits, err := c.Rename(ctx, path, Pos{0, 5}, "soma")
	if err != nil || len(edits) != 1 || len(edits[0].Edits) != 2 {
		t.Fatalf("rename: %+v %v", edits, err)
	}
	refs, err := c.References(ctx, path, Pos{0, 5})
	if err != nil || len(refs) != 2 {
		t.Fatalf("references: %+v %v", refs, err)
	}
	actions, err := c.CodeActions(ctx, path, diags[0].Start, diags)
	if err != nil {
		t.Fatal(err)
	}
	fixed := false
	for _, a := range actions {
		for _, f := range a.Edits {
			for _, e := range f.Edits {
				fixed = fixed || e.Text == ";"
			}
		}
	}
	if !fixed {
		t.Fatalf("o fix-it do ';' como ação: %+v", actions)
	}
	if f, err := c.Format(ctx, path, 4); err != nil || len(f) == 0 {
		t.Fatalf("format com .clang-format: %+v %v", f, err)
	}
}

// The real Python server (ty when lightyear downloads it): a type error,
// hover and completion.
func TestPython(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.py")
	s, _ := ServerFor(path)
	strict := needServer(t, s)
	c, err := open(s, path, "def add(a: int, b: int) -> int:\n    return a + b\n\n\ncount: int = \"x\"\nprint(add(1, 2))\n")
	if err != nil {
		clangdFailed(t, strict, err)
	}
	defer c.Close()
	for found := false; !found; {
		ev := next(t, c)
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
		for _, d := range ev.Diagnostics {
			found = found || (d.Start.Line == 4 && d.Severity == Error)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Line 5 is "print(add(1, 2))": "add" starts at rune 6.
	if h, err := c.Hover(ctx, path, Pos{5, 7}); err != nil || !strings.Contains(h, "a: int") {
		t.Fatalf("hover: %q %v", h, err)
	}
	items, err := c.Completion(ctx, path, Pos{5, 9})
	if err != nil || !slices.ContainsFunc(items, func(it Item) bool { return it.Label == "add" }) {
		t.Fatalf("completion: %d itens %v", len(items), err)
	}
}
