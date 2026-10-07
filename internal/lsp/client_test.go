package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
			publish(p.ContentChanges[0].Text)
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
	c, err := Start(context.Background(), fake(t, "ok"), path, "ok\nxy ERR\n😀ERR")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ev := next(t, c)
	want := []Diagnostic{
		{Start: Pos{1, 3}, End: Pos{1, 6}, Severity: Warning, Message: "erro aqui"},
		{Start: Pos{2, 1}, End: Pos{2, 4}, Severity: Warning, Message: "erro aqui"}, // 😀 is 2 UTF-16 units, 1 rune
	}
	if fmt.Sprint(ev.Diagnostics) != fmt.Sprint(want) {
		t.Fatalf("diagnostics: %v, esperado %v", ev.Diagnostics, want)
	}

	c.DidChange("ERR")
	if ev := next(t, c); len(ev.Diagnostics) != 1 || ev.Diagnostics[0].Start != (Pos{0, 0}) {
		t.Fatalf("depois da mudança: %+v", ev)
	}
	c.DidChange("tudo certo")
	if ev := next(t, c); ev.Err != nil || len(ev.Diagnostics) != 0 {
		t.Fatalf("sem erros deveria limpar: %+v", ev)
	}
}

func TestUTF32IsUsedWhenTheServerAgrees(t *testing.T) {
	t.Setenv("FAKE_ENC", "utf-32")
	c, err := Start(context.Background(), fake(t, "ok"), filepath.Join(t.TempDir(), "a.c"), "😀ERR")
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
	_, err := Start(context.Background(), fake(t, "die"), filepath.Join(dir, "a.c"), "")
	if err == nil || !strings.Contains(err.Error(), "Unknown binary 'rust-analyzer'") || !strings.Contains(err.Error(), "instale o fake") {
		t.Fatalf("servidor que morre ao iniciar: %v", err)
	}

	missing := Server{Name: "nada", Commands: [][]string{{"lightyear-servidor-que-nao-existe"}}, Hint: "instale o nada"}
	if _, err := Start(context.Background(), missing, filepath.Join(dir, "a.c"), ""); err == nil || !strings.Contains(err.Error(), "não encontrado — instale o nada") {
		t.Fatalf("servidor ausente: %v", err)
	}

	c, err := Start(context.Background(), fake(t, "crash"), filepath.Join(dir, "a.c"), "")
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

// The real clangd, when installed: the 42 flags turn an unused variable
// into an error, like the grader.
func TestClangd(t *testing.T) {
	if _, err := exec.LookPath("clangd"); err != nil {
		t.Skip("clangd não instalado")
	}
	path := filepath.Join(t.TempDir(), "main.c")
	s, _ := ServerFor(path)
	c, err := Start(context.Background(), s, path, "int\tmain(void)\n{\n\tint\tx;\n\n\treturn (0);\n}\n")
	if err != nil {
		t.Skipf("clangd não iniciou aqui: %v", err)
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
