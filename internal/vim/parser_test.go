package vim

import (
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestMotions(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		at     editor.Position
		seq    string
		wantAt editor.Position
	}{
		{"w para no início da palavra", "int main(void)", pos(0, 0), "w", pos(0, 4)},
		{"w separa palavra e pontuação", "int main(void)", pos(0, 4), "w", pos(0, 8)},
		{"4w", "int main(void)", pos(0, 0), "4w", pos(0, 13)},
		{"w para em linha vazia", "a b\n\nc", pos(0, 2), "w", pos(1, 0)},
		{"w sai da linha vazia", "a b\n\nc", pos(1, 0), "w", pos(2, 0)},
		{"b volta ao início da palavra", "foo bar", pos(0, 6), "b", pos(0, 4)},
		{"2b", "foo bar", pos(0, 6), "2b", pos(0, 0)},
		{"b cruza a linha", "abc\n  x", pos(1, 2), "b", pos(0, 0)},
		{"e vai ao fim da palavra", "foo bar", pos(0, 0), "e", pos(0, 2)},
		{"e no fim vai à próxima", "foo bar", pos(0, 2), "e", pos(0, 6)},
		{"0 vai ao começo", "\t  x = 1;", pos(0, 5), "0", pos(0, 0)},
		{"^ vai ao primeiro não-branco", "\t  x = 1;", pos(0, 5), "^", pos(0, 3)},
		{"$ vai ao último caractere", "\t  x = 1;", pos(0, 0), "$", pos(0, 8)},
		{"G vai à última linha", "a\n  b\nc", pos(0, 0), "G", pos(2, 0)},
		{"gg vai à primeira", "a\n  b\nc", pos(2, 0), "gg", pos(0, 0)},
		{"2G vai à linha 2, no não-branco", "a\n  b\nc", pos(0, 0), "2G", pos(1, 2)},
		{"3gg", "a\n  b\nc", pos(0, 0), "3gg", pos(2, 0)},
		{"3j", "a\nb\nc\nd", pos(0, 0), "3j", pos(3, 0)},
		{"2k", "a\nb\nc\nd", pos(3, 0), "2k", pos(1, 0)},
		{"9j para na última", "a\nb\nc\nd", pos(0, 0), "9j", pos(3, 0)},
		{"3l", "abcdef", pos(0, 0), "3l", pos(0, 3)},
		{"10l para no último (0 continua o contador)", "abcdefghijk", pos(0, 5), "10l", pos(0, 10)},
		{"g seguido de outra tecla cancela", "a\nb", pos(1, 0), "gxk", pos(0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := editor.New(tt.text)
			ed.MoveCursor(tt.at)
			c := New(ed)
			run(c, tt.seq)
			if ed.Cursor() != tt.wantAt || ed.Buffer().Text() != tt.text {
				t.Fatalf("%q: cursor %v (texto %q), esperado %v", tt.seq, ed.Cursor(), ed.Buffer().Text(), tt.wantAt)
			}
		})
	}
}

func TestOperators(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		at     editor.Position
		seq    string
		want   string
		wantAt editor.Position
	}{
		{"dw", "foo bar baz", pos(0, 0), "dw", "bar baz", pos(0, 0)},
		{"3dw", "a b c d", pos(0, 0), "3dw", "d", pos(0, 0)},
		{"2d2w multiplica", "a b c d e", pos(0, 0), "2d2w", "e", pos(0, 0)},
		{"dw na última palavra não junta linhas", "foo bar\nbaz", pos(0, 4), "dw", "foo \nbaz", pos(0, 3)},
		{"d$", "hello world", pos(0, 6), "d$", "hello ", pos(0, 5)},
		{"de inclui o fim da palavra", "foo.bar", pos(0, 0), "de", ".bar", pos(0, 0)},
		{"db", "foo bar", pos(0, 4), "db", "bar", pos(0, 0)},
		{"d0", "abcdef", pos(0, 3), "d0", "def", pos(0, 0)},
		{"d^", "  abc", pos(0, 4), "d^", "  c", pos(0, 2)},
		{"dj apaga duas linhas", "a\nb\nc", pos(0, 0), "dj", "c", pos(0, 0)},
		{"dk na primeira linha não faz nada", "a\nb", pos(0, 0), "dk", "a\nb", pos(0, 0)},
		{"3dd", "1\n2\n3\n4\n5", pos(1, 0), "3dd", "1\n5", pos(1, 0)},
		{"5dd além do fim", "1\n2\n3", pos(1, 0), "5dd", "1", pos(0, 0)},
		{"d3d é 3dd", "1\n2\n3\n4", pos(0, 0), "d3d", "4", pos(0, 0)},
		{"2d2d apaga 4", "1\n2\n3\n4\n5", pos(0, 0), "2d2d", "5", pos(0, 0)},
		{"dG", "a\nb\nc", pos(1, 0), "dG", "a", pos(0, 0)},
		{"dgg", "a\nb\nc", pos(1, 0), "dgg", "c", pos(0, 0)},
		{"dl no último caractere", "ab", pos(0, 1), "dl", "a", pos(0, 0)},
		{"3x", "abcdef", pos(0, 1), "3x", "aef", pos(0, 1)},
		{"9x para no fim da linha", "abcdef", pos(0, 4), "9x", "abcd", pos(0, 3)},
		{"operador cancelado por tecla inválida", "abc", pos(0, 0), "dzx", "bc", pos(0, 0)},
		{"d seguido de : cancela", "abc", pos(0, 0), "d:<esc>x", "bc", pos(0, 0)},
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

func TestCountsOnUndoRedoAndPending(t *testing.T) {
	ed := editor.New("")
	c := New(ed)
	run(c, "ia<esc>Ab<esc>")
	if ed.Buffer().Text() != "ab" {
		t.Fatalf("texto: %q", ed.Buffer().Text())
	}
	run(c, "2u")
	if ed.Buffer().Text() != "" {
		t.Fatalf("2u deveria desfazer as duas sessões: %q", ed.Buffer().Text())
	}
	run(c, "2<c-r>")
	if ed.Buffer().Text() != "ab" {
		t.Fatalf("2 ctrl+r: %q", ed.Buffer().Text())
	}

	run(c, "3d2")
	if c.Pending() != "3d2" {
		t.Fatalf("pendente deveria mostrar o comando digitado: %q", c.Pending())
	}
	run(c, "<esc>")
	if c.Pending() != "" || ed.Buffer().Text() != "ab" {
		t.Fatalf("esc cancela tudo: pendente %q texto %q", c.Pending(), ed.Buffer().Text())
	}
}
