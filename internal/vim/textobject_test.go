package vim

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestTextObjectsAndFind(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		at       editor.Position
		seq      string
		want     string
		wantAt   editor.Position
		wantMode Mode
	}{
		// Words
		{"diw", "foo bar baz", pos(0, 5), "diw", "foo  baz", pos(0, 4), Normal},
		{"daw leva o branco depois", "foo bar baz", pos(0, 5), "daw", "foo baz", pos(0, 4), Normal},
		{"daw na última palavra leva o branco antes", "foo bar", pos(0, 5), "daw", "foo", pos(0, 2), Normal},
		{"ciw", "foo bar baz", pos(0, 4), "ciwx<esc>", "foo x baz", pos(0, 4), Normal},
		{"diw em pontuação", "a->b", pos(0, 1), "diw", "ab", pos(0, 1), Normal},
		{"viwd", "foo bar baz", pos(0, 5), "viwd", "foo  baz", pos(0, 4), Normal},
		// Quotes
		{`di"`, `say "hello world" ok`, pos(0, 8), `di"`, `say "" ok`, pos(0, 5), Normal},
		{`da" leva o branco depois`, `say "hello" ok`, pos(0, 6), `da"`, `say ok`, pos(0, 4), Normal},
		{`ci" antes das aspas usa o primeiro par`, `x = "ab";`, pos(0, 0), `ci"z<esc>`, `x = "z";`, pos(0, 5), Normal},
		{`di" ignora aspas escapadas`, `"a\"b" c`, pos(0, 1), `di"`, `"" c`, pos(0, 1), Normal},
		// Brackets
		{"di(", "f(a, (b), c)", pos(0, 2), "di(", "f()", pos(0, 2), Normal},
		{"di( no par de dentro", "f(a, (b), c)", pos(0, 6), "di(", "f(a, (), c)", pos(0, 6), Normal},
		{"da(", "f(a, (b), c)", pos(0, 2), "da(", "f", pos(0, 0), Normal},
		{"di( com o cursor no (", "f(a)", pos(0, 1), "di(", "f()", pos(0, 2), Normal},
		{"di) com o cursor no )", "f(a, (b))", pos(0, 8), "di)", "f()", pos(0, 2), Normal},
		{"dib", "(x)", pos(0, 1), "dib", "()", pos(0, 1), Normal},
		{"di{ em várias linhas mantém as chaves", "int\tf(void)\n{\n\treturn (0);\n}", pos(2, 3), "di{", "int\tf(void)\n{\n}", pos(2, 0), Normal},
		{"ci{ em várias linhas mantém a indentação", "{\n\tx;\n\ty;\n}", pos(1, 1), "ci{z<esc>", "{\n\tz\n}", pos(1, 1), Normal},
		{"yi{ copia as linhas", "{\n\tx;\n}", pos(1, 1), "yi{P", "{\n\tx;\n\tx;\n}", pos(1, 1), Normal},
		{"di{ vazio não faz nada", "{\n}", pos(0, 0), "di{", "{\n}", pos(0, 0), Normal},
		{"ci( vazio digita dentro", "f()", pos(0, 1), "ci(x<esc>", "f(x)", pos(0, 2), Normal},
		{"di( sem parênteses não faz nada", "abc", pos(0, 1), "di(", "abc", pos(0, 1), Normal},
		{"vi(d", "f(abc)", pos(0, 3), "vi(d", "f()", pos(0, 2), Normal},
		{"i sem operador ainda insere", "ab", pos(0, 0), "iw<esc>", "wab", pos(0, 0), Normal},
		// f F t T ; ,
		{"fb", "abcabc", pos(0, 0), "fbx", "acabc", pos(0, 1), Normal},
		{"2fb", "abcabc", pos(0, 0), "2fbx", "abcac", pos(0, 4), Normal},
		{"tb ao lado não anda; ; pula", "abcabc", pos(0, 0), "tb;x", "abcbc", pos(0, 3), Normal},
		{"Fa", "abcabc", pos(0, 5), "Fax", "abcbc", pos(0, 3), Normal},
		{"Ta", "abcabc", pos(0, 5), "Tax", "abcac", pos(0, 4), Normal},
		{"; e ,", "abcabcabc", pos(0, 0), "fb;;,x", "abcacabc", pos(0, 4), Normal},
		{"dfc inclui o c", "abcabc", pos(0, 0), "dfc", "abc", pos(0, 0), Normal},
		{"dtc", "abcabc", pos(0, 0), "dtc", "cabc", pos(0, 0), Normal},
		{"dFa exclui o cursor", "abcabc", pos(0, 5), "dFa", "abcc", pos(0, 3), Normal},
		{"cf", "a.b.c", pos(0, 0), "cf.x<esc>", "xb.c", pos(0, 0), Normal},
		{"fz não acha e não anda", "abc", pos(0, 1), "fzdfz", "abc", pos(0, 1), Normal},
		{"f2 procura o 2 (não é contador)", "a12", pos(0, 0), "f2x", "a1", pos(0, 1), Normal},
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

func TestSearch(t *testing.T) {
	ed := editor.New("foo bar\nbar baz\nfoobar foo")
	c := New(ed)
	run(c, "/ba")
	if c.Mode() != Command || c.Prompt() != "/" || c.CommandLine() != "ba" {
		t.Fatalf("prompt da busca: %v %q %q", c.Mode(), c.Prompt(), c.CommandLine())
	}
	steps := []struct {
		seq  string
		want editor.Position
	}{
		{"r<cr>", pos(0, 4)},
		{"n", pos(1, 0)},
		{"n", pos(2, 3)},
		{"n", pos(0, 4)}, // wraps around
		{"N", pos(2, 3)}, // and back
		{"2n", pos(1, 0)},
		{"?foo<cr>", pos(0, 0)},
		{"n", pos(2, 7)}, // ? goes backwards: wraps to the end
		{"N", pos(0, 0)},
		{"*", pos(2, 7)}, // whole word: skips "foobar"
		{"#", pos(0, 0)},
		{"/<cr>", pos(2, 7)}, // empty: the last search (* made it whole-word "foo", forward)
	}
	for _, s := range steps {
		if res := run(c, s.seq); res.Err || ed.Cursor() != s.want {
			t.Fatalf("%q: cursor %v (%+v), esperado %v", s.seq, ed.Cursor(), res, s.want)
		}
	}
	if starts, n := c.Hits("foo foobar foo"); n != 3 || len(starts) != 2 || starts[1] != 11 {
		t.Fatalf("destaques: %v %d", starts, n)
	}
	run(c, ":noh<cr>")
	if starts, _ := c.Hits("foo"); starts != nil {
		t.Fatal(":noh apaga os destaques")
	}
	if res := run(c, "/nada<cr>"); !res.Err || !strings.Contains(res.Message, "E486") {
		t.Fatalf("não encontrado: %+v", res)
	}
	if res := run(New(editor.New("x")), "n"); !res.Err || !strings.Contains(res.Message, "E35") {
		t.Fatalf("n sem busca: %+v", res)
	}
}
