package vim

import (
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestVisualMode(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		at     editor.Position
		seq    string
		want   string
		wantAt editor.Position
	}{
		{"vjd apaga do cursor até a linha de baixo, inclusive", "abc\ndef\nghi", pos(0, 1), "vjd", "af\nghi", pos(0, 1)},
		{"vlld", "abcdef", pos(0, 0), "vlld", "def", pos(0, 0)},
		{"seleção para trás (vbd)", "foo bar", pos(0, 4), "vbd", "ar", pos(0, 0)},
		{"vex", "foo bar", pos(0, 0), "vex", " bar", pos(0, 0)},
		{"v$d inclui o último caractere", "hello world", pos(0, 6), "v$d", "hello ", pos(0, 5)},
		{"vggd até o começo", "a\nbc\nde", pos(2, 1), "vggd", "", pos(0, 0)},
		{"v3ld com contador", "abcdef", pos(0, 0), "v3ld", "ef", pos(0, 0)},
		{"Vjd apaga linhas inteiras", "a\nb\nc", pos(0, 0), "Vjd", "c", pos(0, 0)},
		{"V2jd", "a\nb\nc", pos(0, 0), "V2jd", "", pos(0, 0)},
		{"V no fim do documento", "a\nb\nc", pos(2, 0), "Vkd", "a", pos(0, 0)},
		{"o troca a ponta e continua", "abcdef", pos(0, 2), "vlohd", "aef", pos(0, 1)},
		{"vV troca para linhas", "ab\ncd", pos(0, 1), "vVd", "cd", pos(0, 0)},
		{"termina numa linha vazia e leva a quebra", "abc\n\ndef", pos(0, 1), "vjd", "adef", pos(0, 1)},
		{"esc sai sem apagar", "abc", pos(0, 0), "vl<esc>x", "ac", pos(0, 1)},
		{"v de novo sai sem apagar", "abc", pos(0, 0), "vlvx", "ac", pos(0, 1)},
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
			if ed.Selection().Mode != editor.SelectNone {
				t.Fatal("a seleção deveria acabar ao sair do Visual")
			}
		})
	}
}

func TestVisualModesAndUndo(t *testing.T) {
	ed := editor.New("abc\ndef")
	c := New(ed)
	run(c, "v")
	if c.Mode() != Visual || c.Mode().String() != "VISUAL" {
		t.Fatalf("v: modo %v", c.Mode())
	}
	run(c, "V")
	if c.Mode() != VisualLine || c.Mode().String() != "V-LINE" || ed.Selection().Mode != editor.SelectLines {
		t.Fatalf("V: modo %v seleção %v", c.Mode(), ed.Selection().Mode)
	}
	run(c, "jd")
	if ed.Buffer().Text() != "" {
		t.Fatalf("Vjd: %q", ed.Buffer().Text())
	}
	run(c, "u")
	if ed.Buffer().Text() != "abc\ndef" || ed.Cursor() != pos(0, 0) {
		t.Fatalf("u deveria desfazer a remoção visual e voltar ao início: %q %v", ed.Buffer().Text(), ed.Cursor())
	}

	// Dragging the mouse: press at (0,0), then drag in Normal starts a
	// character selection.
	c.MoveCursor(pos(0, 0))
	c.ExtendSelection(pos(1, 1))
	if c.Mode() != Visual {
		t.Fatalf("arrastar deveria entrar no Visual: %v", c.Mode())
	}
	if r, _, _ := ed.SelectedRange(); ed.Buffer().Slice(r) != "abc\nde" {
		t.Fatalf("seleção do arraste: %q", ed.Buffer().Slice(r))
	}
}

func TestUndoReturnsToChangeStart(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		at     editor.Position
		seq    string
		wantAt editor.Position
	}{
		{"dbu", "foo bar", pos(0, 4), "dbu", pos(0, 0)},
		{"dku", "ab\ncd", pos(1, 1), "dku", pos(0, 1)},
		{"ddu mantém a coluna", "ab\ncd", pos(1, 1), "ddu", pos(1, 1)},
		{"vbdu", "foo bar", pos(0, 4), "vbdu", pos(0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := editor.New(tt.text)
			ed.MoveCursor(tt.at)
			run(New(ed), tt.seq)
			if ed.Buffer().Text() != tt.text || ed.Cursor() != tt.wantAt {
				t.Fatalf("%q: texto %q cursor %v, esperado %v", tt.seq, ed.Buffer().Text(), ed.Cursor(), tt.wantAt)
			}
		})
	}
}
