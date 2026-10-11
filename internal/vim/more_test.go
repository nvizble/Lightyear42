package vim

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// moreCase is one key sequence on a document: the text and cursor after it.
type moreCase struct {
	name   string
	text   string
	at     editor.Position
	seq    string
	want   string
	wantAt editor.Position
}

func runMore(t *testing.T, tests []moreCase, height int) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := editor.New(tt.text)
			if height > 0 {
				ed.SetViewportSize(80, height)
			}
			ed.MoveCursor(tt.at)
			c := New(ed)
			run(c, tt.seq)
			if got := ed.Buffer().Text(); got != tt.want || ed.Cursor() != tt.wantAt {
				t.Fatalf("%q: texto %q cursor %v; esperado %q %v", tt.seq, got, ed.Cursor(), tt.want, tt.wantAt)
			}
		})
	}
}

func TestMoreMotions(t *testing.T) {
	code := "int\tmain(void)\n{\n\treturn (f(a[1], b));\n}"
	runMore(t, []moreCase{
		{"End vai ao último caractere", "hello world", pos(0, 0), "<end>", "hello world", pos(0, 10)},
		{"Home vai à coluna 0", "  hello", pos(0, 5), "<home>", "  hello", pos(0, 0)},
		{"d<End> apaga até o fim", "hello world", pos(0, 6), "d<end>", "hello ", pos(0, 5)},
		{"Ctrl-End e Ctrl-Home", "a\n b\n  c", pos(0, 0), "<c-end>", "a\n b\n  c", pos(2, 2)},
		{"Ctrl-Home volta ao começo", "a\n b\nc", pos(2, 0), "<c-home>", "a\n b\nc", pos(0, 0)},
		{"W pula a palavra com pontuação", "a.b c.d e", pos(0, 0), "W", "a.b c.d e", pos(0, 4)},
		{"w para na pontuação", "a.b c.d e", pos(0, 0), "w", "a.b c.d e", pos(0, 1)},
		{"B volta uma WORD", "a.b c.d e", pos(0, 8), "B", "a.b c.d e", pos(0, 4)},
		{"E vai ao fim da WORD", "a.b c.d e", pos(0, 0), "E", "a.b c.d e", pos(0, 2)},
		{"dW apaga a WORD", "a.b c.d", pos(0, 0), "dW", "c.d", pos(0, 0)},
		{"ge volta ao fim da palavra anterior", "one two three", pos(0, 8), "ge", "one two three", pos(0, 6)},
		{"ge atravessa linhas", "one\ntwo", pos(1, 0), "ge", "one\ntwo", pos(0, 2)},
		{"gE pula pontuação", "a.b c", pos(0, 4), "gE", "a.b c", pos(0, 2)},
		{"ge no começo não anda", "one", pos(0, 0), "ge", "one", pos(0, 0)},
		{"g_ vai ao último não-branco", "abc   ", pos(0, 0), "g_", "abc   ", pos(0, 2)},
		{"| vai à coluna", "abcdef", pos(0, 0), "4|", "abcdef", pos(0, 3)},
		{"| conta tabs como espaço na tela", "\tx", pos(0, 1), "1|", "\tx", pos(0, 0)},
		{"+ desce ao primeiro não-branco", "a\n  b", pos(0, 0), "+", "a\n  b", pos(1, 2)},
		{"Enter também", "a\n  b", pos(0, 0), "<cr>", "a\n  b", pos(1, 2)},
		{"- sobe ao primeiro não-branco", "  a\nb", pos(1, 0), "-", "  a\nb", pos(0, 2)},
		{"_ fica na linha", "  a", pos(0, 2), "_", "  a", pos(0, 2)},
		{"d+ apaga duas linhas", "a\nb\nc", pos(0, 0), "d+", "c", pos(0, 0)},
		{"} vai à próxima linha vazia", "a\nb\n\nc", pos(0, 0), "}", "a\nb\n\nc", pos(2, 0)},
		{"} sem linha vazia vai ao fim", "a\nbc", pos(0, 0), "}", "a\nbc", pos(1, 1)},
		{"{ volta à linha vazia", "a\n\nb\nc", pos(3, 0), "{", "a\n\nb\nc", pos(1, 0)},
		{"2} pula dois parágrafos", "a\n\nb\n\nc", pos(0, 0), "2}", "a\n\nb\n\nc", pos(3, 0)},
		{"d} apaga o parágrafo", "a\nb\n\nc", pos(0, 0), "d}", "\nc", pos(0, 0)},
		{"% vai ao par", code, pos(2, 1), "%", code, pos(2, 19)},
		{"% volta do par", code, pos(2, 19), "%", code, pos(2, 8)},
		{"% atravessa linhas", code, pos(1, 0), "%", code, pos(3, 0)},
		{"% com colchete aninhado", "f(a[1])", pos(0, 3), "%", "f(a[1])", pos(0, 5)},
		{"d% apaga até o par", "x(a b)y", pos(0, 1), "d%", "xy", pos(0, 1)},
		{"% sem par não anda", "abc", pos(0, 0), "%", "abc", pos(0, 0)},
		{"50% vai à metade", "1\n2\n3\n4", pos(0, 0), "50%", "1\n2\n3\n4", pos(1, 0)},
		{"Backspace e espaço andam", "abc", pos(0, 1), " <bs><bs>", "abc", pos(0, 0)},
	}, 0)
}

func TestScreenMotions(t *testing.T) {
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "  line"
	}
	doc := strings.Join(lines, "\n")
	runMore(t, []moreCase{
		{"H vai ao topo da tela", doc, pos(5, 0), "H", doc, pos(0, 2)},
		{"L vai ao fim da tela (sem rolar)", doc, pos(0, 0), "L", doc, pos(6, 2)},
		{"M vai ao meio", doc, pos(0, 0), "M", doc, pos(4, 2)},
		{"Ctrl-d desce meia tela", doc, pos(0, 0), "<c-d>", doc, pos(5, 2)},
		{"Ctrl-u sobe meia tela", doc, pos(20, 2), "<c-u>", doc, pos(15, 2)},
		{"Ctrl-f desce uma tela", doc, pos(0, 0), "<c-f>", doc, pos(8, 2)},
		{"PgDn também", doc, pos(0, 0), "<pgdn>", doc, pos(8, 2)},
		{"Ctrl-b volta uma tela", doc, pos(20, 0), "<c-b>", doc, pos(12, 2)},
		{"Ctrl-u no topo não anda", doc, pos(0, 0), "<c-u>", doc, pos(0, 0)},
		{"Ctrl-d no fim não passa", doc, pos(49, 0), "<c-d>", doc, pos(49, 0)},
	}, 10)

	t.Run("zt, zz e zb movem a tela", func(t *testing.T) {
		ed := editor.New(doc)
		ed.SetViewportSize(80, 10)
		ed.MoveCursor(pos(20, 0))
		c := New(ed)
		for _, tc := range []struct {
			seq string
			top int
		}{{"zt", 17}, {"zz", 15}, {"zb", 14}} {
			run(c, tc.seq)
			if got := ed.Viewport().Top; got != tc.top || ed.Cursor() != pos(20, 0) {
				t.Errorf("%s: topo %d cursor %v, esperado topo %d e o cursor parado", tc.seq, got, ed.Cursor(), tc.top)
			}
		}
	})
	t.Run("Ctrl-e rola a tela e o cursor só se precisar", func(t *testing.T) {
		ed := editor.New(doc)
		ed.SetViewportSize(80, 10)
		ed.MoveCursor(pos(5, 0))
		c := New(ed)
		run(c, "<c-e><c-e>")
		if ed.Viewport().Top != 2 || ed.Cursor().Line != 5 {
			t.Fatalf("topo %d cursor %v", ed.Viewport().Top, ed.Cursor())
		}
		run(c, "<c-e><c-e><c-e><c-e>")
		if ed.Viewport().Top != 6 || ed.Cursor().Line != 9 {
			t.Fatalf("o cursor devia descer junto: topo %d cursor %v", ed.Viewport().Top, ed.Cursor())
		}
		run(c, "<c-y>")
		if ed.Viewport().Top != 5 {
			t.Fatalf("Ctrl-y: topo %d", ed.Viewport().Top)
		}
	})
}

func TestMoreEdits(t *testing.T) {
	runMore(t, []moreCase{
		{"J junta com espaço", "a\n  b", pos(0, 0), "J", "a b", pos(0, 1)},
		{"3J junta três", "a\nb\nc\nd", pos(0, 0), "3J", "a b c\nd", pos(0, 3)},
		{"J antes de ) não põe espaço", "f(a\n)", pos(0, 0), "J", "f(a)", pos(0, 2)},
		{"J com linha vazia", "a\n\nb", pos(0, 0), "J", "a\nb", pos(0, 0)},
		{"J na última linha não faz nada", "a\nb", pos(1, 0), "J", "a\nb", pos(1, 0)},
		{"gJ junta sem mexer", "a\n  b", pos(0, 0), "gJ", "a  b", pos(0, 1)},
		{"J desfaz de uma vez", "a\nb\nc", pos(0, 0), "3Ju", "a\nb\nc", pos(0, 0)},
		{"r troca o caractere", "abc", pos(0, 1), "rx", "axc", pos(0, 1)},
		{"3r troca três", "abcd", pos(0, 0), "3rx", "xxxd", pos(0, 2)},
		{"r além do fim não faz nada", "ab", pos(0, 1), "3rx", "ab", pos(0, 1)},
		{"r e ponto repetem", "abcd", pos(0, 0), "rxl.", "xxcd", pos(0, 1)},
		{"~ troca a caixa e anda", "aBc", pos(0, 0), "~~", "Abc", pos(0, 2)},
		{"3~ troca três", "abcd", pos(0, 0), "3~", "ABCd", pos(0, 3)},
		{"~ no último fica nele", "ab", pos(0, 1), "~", "aB", pos(0, 1)},
		{"s troca um caractere", "abc", pos(0, 1), "sX<esc>", "aXc", pos(0, 1)},
		{"S troca a linha", "  abc", pos(0, 3), "Sx<esc>", "  x", pos(0, 2)},
		{"X apaga antes do cursor", "abc", pos(0, 2), "X", "ac", pos(0, 1)},
		{">> indenta com tab", "a\nb", pos(0, 0), ">>", "\ta\nb", pos(0, 1)},
		{"3>> indenta três", "a\nb\nc", pos(0, 0), "3>>", "\ta\n\tb\n\tc", pos(0, 1)},
		{">j indenta duas", "a\nb\nc", pos(0, 0), ">j", "\ta\n\tb\nc", pos(0, 1)},
		{">> não indenta linha vazia", "a\n\nb", pos(0, 0), ">2j", "\ta\n\n\tb", pos(0, 1)},
		{"<< tira um tab", "\t\ta", pos(0, 2), "<<", "\ta", pos(0, 1)},
		{"<< tira espaços", "      a", pos(0, 6), "<<", "  a", pos(0, 2)},
		{"<< sem indentação não faz nada", "a", pos(0, 0), "<<", "a", pos(0, 0)},
		{">> e u", "a", pos(0, 0), ">>u", "a", pos(0, 0)},
		{"gUw passa a palavra para maiúsculas", "hello world", pos(0, 0), "gUw", "HELLO world", pos(0, 0)},
		{"guiw para minúsculas", "HELLO", pos(0, 2), "guiw", "hello", pos(0, 0)},
		{"g~~ troca a linha", "aBc", pos(0, 0), "g~~", "AbC", pos(0, 0)},
		{"gUU a linha", "  abc", pos(0, 0), "gUU", "  ABC", pos(0, 2)},
		{"guu a linha", "ABC", pos(0, 1), "guu", "abc", pos(0, 0)},
		{"gU não mexe no registro", "abc", pos(0, 0), "yiwgUiwP", "abcABC", pos(0, 2)},
		{"Ctrl-a soma 1", "x = 41;", pos(0, 0), "<c-a>", "x = 42;", pos(0, 5)},
		{"5 Ctrl-a soma 5", "n9", pos(0, 0), "5<c-a>", "n14", pos(0, 2)},
		{"Ctrl-x subtrai", "x = 0", pos(0, 0), "<c-x>", "x = -1", pos(0, 5)},
		{"Ctrl-a com negativo", "-5", pos(0, 0), "10<c-a>", "5", pos(0, 0)},
		{"Ctrl-a dentro do número", "1234", pos(0, 2), "<c-a>", "1235", pos(0, 3)},
		{"Ctrl-a sem número não faz nada", "abc", pos(0, 0), "<c-a>", "abc", pos(0, 0)},
		{"marca e volta exata", "abc\ndef", pos(0, 2), "majj`a", "abc\ndef", pos(0, 2)},
		{"marca e volta à linha", "  abc\ndef", pos(0, 4), "mbj'b", "  abc\ndef", pos(0, 2)},
		{"d'a apaga até a marca", "a\nb\nc", pos(0, 0), "majjd'a", "", pos(0, 0)},
		{"marca que não existe não anda", "abc", pos(0, 1), "`z", "abc", pos(0, 1)},
	}, 0)
}

func TestMoreInsertAndVisual(t *testing.T) {
	runMore(t, []moreCase{
		{"End e Home no insert", "abc", pos(0, 1), "i<end>X<home>Y<esc>", "YabcX", pos(0, 0)},
		{"Ctrl-w apaga a palavra", "foo bar", pos(0, 7), "A<c-w><esc>", "foo ", pos(0, 3)},
		{"Ctrl-w com espaço antes", "foo bar  ", pos(0, 9), "A<c-w><esc>", "foo ", pos(0, 3)},
		{"Ctrl-w no começo junta", "a\nb", pos(1, 0), "i<c-w><esc>", "ab", pos(0, 0)},
		{"Ctrl-u apaga até a indentação", "\tabc", pos(0, 4), "A<c-u><esc>", "\t", pos(0, 0)},
		{"Ctrl-left e Ctrl-right", "one two", pos(0, 7), "A<c-left>X<c-right>Y<esc>", "one XtwoY", pos(0, 8)},
		{"Ctrl-Home no insert", "a\nb", pos(1, 1), "a<c-home>X<esc>", "Xa\nb", pos(0, 0)},
		{"U no visual", "abc", pos(0, 0), "vlU", "ABc", pos(0, 0)},
		{"u no visual", "ABC", pos(0, 0), "Vu", "abc", pos(0, 0)},
		{"~ no visual", "aBc", pos(0, 0), "v$~", "AbC", pos(0, 0)},
		{"> no visual indenta as linhas", "a\nb\nc", pos(0, 0), "Vj>", "\ta\n\tb\nc", pos(0, 1)},
		{"< no visual", "\ta\n\tb", pos(0, 0), "vj<", "a\nb", pos(0, 0)},
		{"J no visual junta as linhas", "a\nb\nc", pos(0, 0), "VjjJ", "a b c", pos(0, 3)},
		{"s no visual troca a seleção", "abc", pos(0, 0), "vlsX<esc>", "Xc", pos(0, 0)},
		{"End no visual estende", "abc", pos(0, 0), "v<end>d", "", pos(0, 0)},
	}, 0)
}

func TestZZAndZQ(t *testing.T) {
	ed := editor.New("a")
	c := New(ed)
	var got []string
	c.ExCommands = map[string]func(string) Result{
		"x":  func(string) Result { got = append(got, "x"); return Result{Quit: true} },
		"q!": func(string) Result { got = append(got, "q!"); return Result{Quit: true} },
	}
	if res := run(c, "ZZ"); !res.Quit {
		t.Fatal("ZZ deveria sair")
	}
	if res := run(c, "ZQ"); !res.Quit {
		t.Fatal("ZQ deveria sair")
	}
	if strings.Join(got, ",") != "x,q!" {
		t.Fatalf("ZZ/ZQ chamaram %v", got)
	}
}
