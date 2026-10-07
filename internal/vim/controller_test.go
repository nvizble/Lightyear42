package vim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// keys splits a Vim-style sequence: "ihi<esc>" → "i", "h", "i", "esc".
func keys(seq string) []string {
	names := map[string]string{"esc": "esc", "cr": "enter", "bs": "backspace", "tab": "tab", "s-tab": "shift+tab", "c-r": "ctrl+r",
		"up": "up", "down": "down", "left": "left", "right": "right", "del": "delete"}
	var out []string
	for len(seq) > 0 {
		if seq[0] == '<' {
			if end := strings.IndexByte(seq, '>'); end > 0 {
				out = append(out, names[seq[1:end]])
				seq = seq[end+1:]
				continue
			}
		}
		r := []rune(seq)[0]
		out = append(out, string(r))
		seq = seq[len(string(r)):]
	}
	return out
}

func pos(line, col int) editor.Position { return editor.Position{Line: line, Column: col} }

func run(c *Controller, seq string) Result {
	var res Result
	for _, k := range keys(seq) {
		res = c.HandleKey(k)
	}
	return res
}

func TestNormalAndInsert(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		at       editor.Position
		seq      string
		want     string
		wantAt   editor.Position
		wantMode Mode
	}{
		{"insere e esc volta um", "", editor.Position{}, "ihello<esc>", "hello", pos(0, 4), Normal},
		{"modo insert", "", editor.Position{}, "iab", "ab", pos(0, 2), Insert},
		{"x apaga o caractere", "abc", editor.Position{}, "x", "bc", editor.Position{}, Normal},
		{"x no último puxa o cursor", "abc", pos(0, 2), "x", "ab", pos(0, 1), Normal},
		{"x em linha vazia não faz nada", "a\n\nb", pos(1, 0), "x", "a\n\nb", pos(1, 0), Normal},
		{"dd no meio", "a\n  b\nc", pos(1, 2), "dd", "a\nc", pos(1, 0), Normal},
		{"dd na última linha", "a\nb\nc", pos(2, 0), "dd", "a\nb", pos(1, 0), Normal},
		{"dd na única linha", "only", pos(0, 2), "dd", "", editor.Position{}, Normal},
		{"dd cai no primeiro não-branco", "x\n\ty", editor.Position{}, "dd", "\ty", pos(0, 1), Normal},
		{"esc cancela o operador", "abc", editor.Position{}, "d<esc>x", "bc", editor.Position{}, Normal},
		{"o mantém a indentação", "\tif (x)", pos(0, 1), "oy<esc>", "\tif (x)\n\ty", pos(1, 1), Normal},
		{"O abre acima com indentação", "\tif (x)", pos(0, 3), "Oz<esc>", "\tz\n\tif (x)", pos(0, 1), Normal},
		{"a insere depois do cursor", "abc", editor.Position{}, "ad<esc>", "adbc", pos(0, 1), Normal},
		{"A insere no fim", "abc", editor.Position{}, "AX<esc>", "abcX", pos(0, 3), Normal},
		{"I insere antes da indentação", "  abc", pos(0, 4), "IX<esc>", "  Xabc", pos(0, 2), Normal},
		{"enter no insert mantém a indentação", "\tx", pos(0, 1), "a<cr>y<esc>", "\tx\n\ty", pos(1, 1), Normal},
		{"backspace no insert junta linhas", "ab\ncd", pos(1, 0), "i<bs><esc>", "abcd", pos(0, 1), Normal},
		{"h no começo e l no fim não passam", "ab", editor.Position{}, "hhlll", "ab", pos(0, 1), Normal},
		{"j e k lembram a coluna", "abcdef\nab\nabcdef", pos(0, 5), "jj", "abcdef\nab\nabcdef", pos(2, 5), Normal},
		{"j em linha curta para no último", "abcdef\nab", pos(0, 5), "j", "abcdef\nab", pos(1, 1), Normal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := editor.New(tt.text)
			ed.MoveCursor(tt.at)
			c := New(ed)
			run(c, tt.seq)
			if got := ed.Buffer().Text(); got != tt.want || ed.Cursor() != tt.wantAt || c.Mode() != tt.wantMode {
				t.Fatalf("%q: texto %q cursor %v modo %v; esperado %q %v %v", tt.seq, got, ed.Cursor(), c.Mode(), tt.want, tt.wantAt, tt.wantMode)
			}
		})
	}
}

func TestUndoRedoInsertSession(t *testing.T) {
	ed := editor.New("")
	c := New(ed)
	run(c, "ihello<cr>world<esc>")
	if ed.Buffer().Text() != "hello\nworld" {
		t.Fatalf("texto: %q", ed.Buffer().Text())
	}
	run(c, "u")
	if ed.Buffer().Text() != "" {
		t.Fatalf("u deveria desfazer a sessão de insert inteira: %q", ed.Buffer().Text())
	}
	run(c, "<c-r>")
	if ed.Buffer().Text() != "hello\nworld" {
		t.Fatalf("ctrl+r: %q", ed.Buffer().Text())
	}
	if res := run(c, "<c-r>"); res.Message != "nada para refazer" {
		t.Fatalf("sem redo deveria avisar: %+v", res)
	}
	run(c, "ddu")
	if ed.Buffer().Text() != "hello\nworld" {
		t.Fatalf("u deveria desfazer o dd: %q", ed.Buffer().Text())
	}
}

func TestPendingIsVisible(t *testing.T) {
	c := New(editor.New("a"))
	run(c, "d")
	if c.Pending() != "d" {
		t.Fatalf("pendente: %q", c.Pending())
	}
	run(c, "<esc>")
	if c.Pending() != "" {
		t.Fatal("esc deveria cancelar o comando pendente")
	}
}

func TestCommandLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.c")
	ed, err := editor.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c := New(ed)
	run(c, "iint x;<esc>:w")
	if c.Mode() != Command || c.CommandLine() != "w" {
		t.Fatalf("linha de comando: modo %v %q", c.Mode(), c.CommandLine())
	}
	res := run(c, "<cr>")
	if data, _ := os.ReadFile(path); string(data) != "int x;" || res.Quit || !strings.Contains(res.Message, "salvo") {
		t.Fatalf(":w: arquivo %q, resultado %+v", data, res)
	}

	run(c, "x")
	if res := run(c, ":q<cr>"); res.Quit || !res.Err {
		t.Fatalf(":q com alterações deveria recusar: %+v", res)
	}
	if res := run(c, ":wq<cr>"); !res.Quit {
		t.Fatalf(":wq deveria sair: %+v", res)
	}
	// Esc left the cursor on ";" (like Vim), so x removed it.
	if data, _ := os.ReadFile(path); string(data) != "int x" {
		t.Fatalf(":wq deveria salvar antes de sair: %q", data)
	}
	run(c, "x")
	if res := run(c, ":q!<cr>"); !res.Quit {
		t.Fatalf(":q! deveria sair sem salvar: %+v", res)
	}
	if res := run(c, ":x<cr>"); !res.Quit {
		t.Fatalf(":x deveria salvar e sair: %+v", res)
	}
	if res := run(c, ":foo<cr>"); !res.Err || !strings.Contains(res.Message, "foo") {
		t.Fatalf("comando desconhecido: %+v", res)
	}
	run(c, ":<bs>")
	if c.Mode() != Normal {
		t.Fatal("backspace com a linha vazia deveria voltar ao Normal")
	}
	run(c, ":wq<esc>")
	if c.Mode() != Normal || c.CommandLine() != "" {
		t.Fatal("esc deveria cancelar a linha de comando")
	}
}

func TestCommandLineCompletion(t *testing.T) {
	c := New(editor.New(""))
	var ran string
	c.ExCommands = map[string]func(string) Result{"colo": func(arg string) Result { ran = arg; return Result{} }}
	c.ExCompletions = map[string]func() []string{"colo": func() []string { return []string{"dracula", "nord", "dark"} }}
	check := func(line string, items string, at int) {
		t.Helper()
		got, i := c.Completions()
		if c.CommandLine() != line || strings.Join(got, " ") != items || i != at {
			t.Fatalf("linha %q, opções %q (%d); queria %q, %q (%d)", c.CommandLine(), got, i, line, items, at)
		}
	}
	run(c, ":colo  d<tab>")
	check("colo  dracula", "dracula dark", 0)
	run(c, "<tab>")
	check("colo  dark", "dracula dark", 1)
	run(c, "<tab>")
	check("colo  dracula", "dracula dark", 0)
	run(c, "<s-tab><s-tab>")
	check("colo  dracula", "dracula dark", 0)
	run(c, "x")
	check("colo  draculax", "", 0) // typing ends it
	run(c, "<tab>")
	check("colo  draculax", "", 0) // nothing matches
	run(c, "<esc>:colo <s-tab>")
	check("colo dark", "dracula nord dark", 2) // backwards starts at the last
	run(c, "<cr>")
	if ran != "dark" || c.Mode() != Normal {
		t.Fatalf("Enter roda o comando completado: %q", ran)
	}
	for _, seq := range []string{":colo<tab>", ":e <tab>", "/colo <tab>"} {
		run(c, "<esc>"+seq)
		check(strings.TrimSuffix(seq[1:], "<tab>"), "", 0)
	}
}

func TestHandleText(t *testing.T) {
	ed := editor.New("")
	c := New(ed)
	run(c, "i")
	c.HandleText("a\r\nf5") // paste: text, never named keys
	run(c, "<esc>")
	if ed.Buffer().Text() != "a\nf5" {
		t.Fatalf("colar no insert: %q", ed.Buffer().Text())
	}
	c.HandleText("dd") // in Normal, text is read as keys
	if ed.Buffer().Text() != "a" {
		t.Fatalf("texto no normal deveria virar comandos: %q", ed.Buffer().Text())
	}
}

func TestSetEditorAndExCommands(t *testing.T) {
	a, b := editor.New("aaa"), editor.New("bbb")
	c := New(a)
	run(c, "ylvl")
	c.SetEditor(b)
	if c.Mode() != Normal || a.Selection().Mode != editor.SelectNone {
		t.Fatal("trocar de editor volta ao Normal e limpa a seleção")
	}
	run(c, "p")
	if b.Buffer().Text() != "babb" {
		t.Fatalf("o register vale entre buffers: %q", b.Buffer().Text())
	}
	run(c, "x")
	c.SetEditor(a)
	run(c, ".")
	if a.Buffer().Text() != "aa" {
		t.Fatalf("o . vale entre buffers: %q", a.Buffer().Text())
	}
	var got []string
	c.ExCommands = map[string]func(string) Result{
		"e": func(arg string) Result { got = append(got, "e "+arg); return Result{Message: "aberto"} },
		"q": func(string) Result { got = append(got, "q"); return Result{} },
	}
	if res := run(c, ":e  src/main.c <cr>"); res.Message != "aberto" {
		t.Fatalf(":e: %+v", res)
	}
	run(c, ":q<cr>")
	if strings.Join(got, "|") != "e src/main.c|q" {
		t.Fatalf("comandos do host: %q", got)
	}
}

func TestReadOnlyRefusesInsert(t *testing.T) {
	ed := editor.New("abc")
	ed.SetReadOnly(true)
	c := New(ed)
	for _, seq := range []string{"i", "A", "o", "ciw", "cc"} {
		if res := run(c, seq); !res.Err || !strings.Contains(res.Message, "E21") || c.Mode() != Normal {
			t.Fatalf("%q num arquivo só leitura: %+v modo %v", seq, res, c.Mode())
		}
	}
	run(c, "ddxp")
	if ed.Buffer().Text() != "abc" {
		t.Fatalf("nada muda: %q", ed.Buffer().Text())
	}
	run(c, "jlvly")
	if c.Mode() != Normal {
		t.Fatal("ler e copiar continuam funcionando")
	}
}
