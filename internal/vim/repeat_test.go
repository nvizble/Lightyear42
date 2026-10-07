package vim

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestDotRepeats(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		at     editor.Position
		seq    string
		want   string
		wantAt editor.Position
	}{
		{"dw.", "a b c d", pos(0, 0), "dw.", "c d", pos(0, 0)},
		{"x com contador e .", "abcdefg", pos(0, 0), "2x.", "efg", pos(0, 0)},
		{". com contador novo", "abcdefg", pos(0, 0), "2x3.", "fg", pos(0, 0)},
		{"dd.", "1\n2\n3", pos(0, 0), "dd.", "3", pos(0, 0)},
		{"ciw + texto, depois .", "foo bar", pos(0, 0), "ciwx<esc>w.", "x x", pos(0, 2)},
		{"A com texto", "a\nb", pos(0, 0), "A;<esc>j.", "a;\nb;", pos(1, 1)},
		{"o abre outra linha", "a", pos(0, 0), "oz<esc>.", "a\nz\nz", pos(2, 0)},
		{"p repete", "ab", pos(0, 0), "ylp.", "aaab", pos(0, 2)},
		{"vd repete o tamanho", "abcdef", pos(0, 0), "vld.", "ef", pos(0, 0)},
		{"u não vira o .", "abc", pos(0, 0), "xu.", "bc", pos(0, 0)},
		{"movimento não troca o .", "a b c", pos(0, 0), "xw.", "  c", pos(0, 1)},
		{". sem nada antes", "abc", pos(0, 0), ".", "abc", pos(0, 0)},
		{"dfx. (f dentro do .)", "a.b.c.d", pos(0, 0), "df..", "c.d", pos(0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := editor.New(tt.text)
			ed.MoveCursor(tt.at)
			c := New(ed)
			run(c, tt.seq)
			if got := ed.Buffer().Text(); got != tt.want || ed.Cursor() != tt.wantAt || c.Mode() != Normal {
				t.Fatalf("%q: texto %q cursor %v modo %v; esperado %q %v", tt.seq, got, ed.Cursor(), c.Mode(), tt.want, tt.wantAt)
			}
		})
	}
}

func TestDotRepeatsPaste(t *testing.T) {
	ed := editor.New("a\nb")
	c := New(ed)
	run(c, "A")
	c.HandleText("-x") // pasted in Insert
	run(c, "<esc>j.")
	if ed.Buffer().Text() != "a-x\nb-x" {
		t.Fatalf("o . repete o que foi colado: %q", ed.Buffer().Text())
	}
}

func TestMacros(t *testing.T) {
	ed := editor.New("1\n2\n3\n4")
	c := New(ed)
	run(c, "qa")
	if c.Recording() != "a" {
		t.Fatalf("gravando: %q", c.Recording())
	}
	run(c, "A!<esc>jq")
	if c.Recording() != "" || ed.Buffer().Text() != "1!\n2\n3\n4" {
		t.Fatalf("depois de gravar: %q %q", c.Recording(), ed.Buffer().Text())
	}
	run(c, "@a")
	run(c, "@@")
	if ed.Buffer().Text() != "1!\n2!\n3!\n4" {
		t.Fatalf("@a e @@: %q", ed.Buffer().Text())
	}
	run(c, "u")
	if ed.Buffer().Text() != "1!\n2!\n3\n4" {
		t.Fatalf("u desfaz a última execução: %q", ed.Buffer().Text())
	}
	run(c, "2@a")
	if ed.Buffer().Text() != "1!\n2!\n3!\n4!" {
		t.Fatalf("2@a: %q", ed.Buffer().Text())
	}
	if res := run(c, "@b"); !res.Err || !strings.Contains(res.Message, "@b") {
		t.Fatalf("macro vazia: %+v", res)
	}
	// A macro that calls itself stops instead of hanging.
	run(c, "qbx@bq")
	if res := run(c, "@b"); !res.Err {
		t.Fatalf("recursão infinita deveria parar: %+v", res)
	}
	// q waits for a register: "qz" then a key that isn't one is fine.
	ed2 := editor.New("ab")
	c2 := New(ed2)
	run(c2, "q1x")
	if c2.Recording() != "" || ed2.Buffer().Text() != "b" {
		t.Fatalf("q1 não grava: %q %q", c2.Recording(), ed2.Buffer().Text())
	}
}
