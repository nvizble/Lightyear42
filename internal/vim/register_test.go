package vim

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestYankPutChange(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		at       editor.Position
		seq      string
		want     string
		wantAt   editor.Position
		wantMode Mode
	}{
		// y and p/P
		{"yyp duplica a linha", "a\nb", pos(0, 0), "yyp", "a\na\nb", pos(1, 0), Normal},
		{"yyP põe acima", "a\nb", pos(1, 0), "yyP", "a\nb\nb", pos(1, 0), Normal},
		{"yy3p", "a", pos(0, 0), "yy3p", "a\na\na\na", pos(1, 0), Normal},
		{"p depois da última linha", "a\nb", pos(1, 0), "yyp", "a\nb\nb", pos(2, 0), Normal},
		{"p cai no primeiro não-branco", "\tx", pos(0, 1), "yyp", "\tx\n\tx", pos(1, 1), Normal},
		{"Y é yy", "a\nb", pos(0, 0), "Yp", "a\na\nb", pos(1, 0), Normal},
		{"3Y", "1\n2\n3\n4", pos(0, 0), "3YGp", "1\n2\n3\n4\n1\n2\n3", pos(4, 0), Normal},
		{"yk sobe para a primeira linha", "a\nb", pos(1, 0), "ykP", "a\nb\na\nb", pos(0, 0), Normal},
		{"ywP", "foo bar", pos(0, 0), "ywP", "foo foo bar", pos(0, 3), Normal},
		{"ywp cola depois do cursor", "foo bar", pos(0, 0), "ywp", "ffoo oo bar", pos(0, 4), Normal},
		{"yb volta ao início", "foo bar", pos(0, 6), "ybP", "foo babar", pos(0, 5), Normal},
		{"2p", "ab", pos(0, 0), "yl2p", "aaab", pos(0, 2), Normal},
		{"p em linha vazia", "a\n", pos(0, 0), "yljp", "a\na", pos(1, 0), Normal},
		{"colar várias linhas por caractere", "ab\ncd", pos(0, 1), "vjyP", "ab\ncdb\ncd", pos(0, 1), Normal},
		// d and x fill the register too
		{"ddp troca as linhas", "1\n2\n3", pos(0, 0), "ddp", "2\n1\n3", pos(1, 0), Normal},
		{"xp troca os caracteres", "ab", pos(0, 0), "xp", "ba", pos(0, 1), Normal},
		{"dwP desfaz o dw", "foo bar", pos(0, 0), "dwP", "foo bar", pos(0, 3), Normal},
		{"dd em linha vazia guarda uma linha vazia", "a\n\nb", pos(1, 0), "ddp", "a\nb\n", pos(2, 0), Normal},
		{"D apaga até o fim", "hello world", pos(0, 5), "D", "hello", pos(0, 4), Normal},
		{"2D", "ab\ncd\nef", pos(0, 1), "2D", "a\nef", pos(0, 0), Normal},
		// c
		{"cw muda só a palavra", "foo bar", pos(0, 0), "cwx<esc>", "x bar", pos(0, 0), Normal},
		{"cw no fim da palavra muda um caractere", "foo bar", pos(0, 2), "cwx<esc>", "fox bar", pos(0, 2), Normal},
		{"2cw", "a b c", pos(0, 0), "2cwx<esc>", "x c", pos(0, 0), Normal},
		{"cw nos brancos é como dw", "a  b", pos(0, 1), "cwx<esc>", "axb", pos(0, 1), Normal},
		{"cb", "foo bar", pos(0, 4), "cbx<esc>", "xbar", pos(0, 0), Normal},
		{"c$", "hello world", pos(0, 6), "c$x<esc>", "hello x", pos(0, 6), Normal},
		{"C", "hello world", pos(0, 6), "Cx<esc>", "hello x", pos(0, 6), Normal},
		{"C em linha vazia", "", pos(0, 0), "Cx<esc>", "x", pos(0, 0), Normal},
		{"cc mantém a indentação", "\tfoo\nbar", pos(0, 2), "ccx<esc>", "\tx\nbar", pos(0, 1), Normal},
		{"cc numa linha só de indentação", "\t", pos(0, 0), "ccx<esc>", "\tx", pos(0, 1), Normal},
		{"2cc vira uma linha", "\ta\nb\nc", pos(0, 0), "2ccx<esc>", "\tx\nc", pos(0, 1), Normal},
		{"cj", "a\nb\nc", pos(1, 0), "cjx<esc>", "a\nx", pos(1, 0), Normal},
		{"c fica no INSERT", "foo", pos(0, 0), "cw", "", pos(0, 0), Insert},
		{"ch no começo não faz nada", "abc", pos(0, 0), "ch", "abc", pos(0, 0), Normal},
		{"o texto trocado vai para o register", "foo bar", pos(0, 0), "cwx<esc>p", "xfoo bar", pos(0, 3), Normal},
		// Visual
		{"vy copia e volta ao início", "foo bar", pos(0, 6), "vbyP", "foo barbar", pos(0, 6), Normal},
		{"Vy", "a\nb", pos(0, 0), "Vyjp", "a\nb\na", pos(2, 0), Normal},
		{"vc", "foo bar", pos(0, 0), "vecx<esc>", "x bar", pos(0, 0), Normal},
		{"Vc vira uma linha", "\tfoo\nbar\nbaz", pos(0, 0), "Vjcx<esc>", "\tx\nbaz", pos(0, 1), Normal},
		{"vp troca e guarda o texto trocado", "foo bar", pos(0, 0), "yewvep0P", "barfoo foo", pos(0, 2), Normal},
		{"Vp troca linhas", "a\nb\nc", pos(0, 0), "yyjVp", "a\na\nc", pos(1, 0), Normal},
		{"Vp no documento inteiro", "a\nb", pos(0, 0), "yyVGp", "a", pos(0, 0), Normal},
		{"Vp na última linha", "a\nb\nc", pos(0, 0), "yyGVp", "a\nb\na", pos(2, 0), Normal},
		{"vp com linhas divide a linha", "ab\nx", pos(1, 0), "yyggvp", "\nx\nb\nx", pos(1, 0), Normal},
		{"vd guarda o texto", "abc", pos(0, 0), "vldp", "cab", pos(0, 2), Normal},
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

func TestPutAndChangeUndoInOneStep(t *testing.T) {
	tests := []struct {
		name, text string
		at         editor.Position
		seq        string
		wantAt     editor.Position
	}{
		{"yyp + u volta à linha", "a\nb", pos(0, 0), "yypu", pos(0, 0)},
		{"ywP + u", "foo bar", pos(0, 4), "ywPu", pos(0, 4)},
		{"cw + u", "foo bar", pos(0, 0), "cwxyz<esc>u", pos(0, 0)},
		{"cc + u volta à coluna", "\tfoo\nbar", pos(0, 3), "ccx<cr>y<esc>u", pos(0, 3)},
		{"ck + u volta à primeira linha", "ab\ncd", pos(1, 1), "ckx<esc>u", pos(0, 1)},
		{"vp + u", "foo bar", pos(0, 0), "yewvepu", pos(0, 4)},
		{"Vp no documento inteiro + u", "a\nb", pos(0, 0), "yyVGpu", pos(0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := editor.New(tt.text)
			ed.MoveCursor(tt.at)
			run(New(ed), tt.seq)
			if ed.Buffer().Text() != tt.text || ed.Cursor() != tt.wantAt {
				t.Fatalf("%q: texto %q cursor %v; esperado %q %v", tt.seq, ed.Buffer().Text(), ed.Cursor(), tt.text, tt.wantAt)
			}
		})
	}
}

func TestPutWithEmptyRegister(t *testing.T) {
	for _, seq := range []string{"p", "P", "vp"} {
		ed := editor.New("abc")
		c := New(ed)
		res := run(c, seq)
		if !res.Err || !strings.Contains(res.Message, "E353") || ed.Buffer().Text() != "abc" || c.Mode() != Normal {
			t.Fatalf("%q sem nada copiado deveria avisar: %+v %q modo %v", seq, res, ed.Buffer().Text(), c.Mode())
		}
	}
	// A pending operator is canceled by D/C/Y, like any key that isn't
	// a motion.
	ed := editor.New("abc")
	run(New(ed), "dD")
	if ed.Buffer().Text() != "abc" {
		t.Fatalf("dD não deveria apagar: %q", ed.Buffer().Text())
	}
}
