package syntax

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// classes renders the classes of text as one letter per character:
// k keyword, s string, c comment, n number, f function, t type, . plain.
func classes(h *Highlighter, first, last int) string {
	letters := map[Class]byte{Plain: '.', Keyword: 'k', String: 's', Comment: 'c', Number: 'n', Function: 'f', Type: 't'}
	var lines []string
	for _, line := range h.Lines(first, last) {
		b := make([]byte, len(line))
		for i, c := range line {
			b[i] = letters[c]
		}
		lines = append(lines, string(b))
	}
	return strings.Join(lines, "\n")
}

func TestHighlightLanguages(t *testing.T) {
	tests := []struct {
		path, text, want string
	}{
		{"main.c", `int	main(void) { return (0); } // ok`,
			"ttt.ffff.tttt....kkkkkk..n.....ccccc"},
		{"a.c", `#include <unistd.h>`, "kkkkkkkk.ssssssssss"},
		{"a.c", `write(1, "a\n", MAX);`, "fffff.n..sssss..nnn.."},
		{"a.h", `typedef struct s_x t_x;`, "kkkkkkk.kkkkkk.ttt.ttt."},
		{"a.cpp", `class Foo { bool ok() const; };`, "kkkkk.ttt...tttt.ff...kkkkk...."},
		{"main.go", `func main() { fmt.Println("é") }`, "kkkk.ffff.........fffffff.sss..."},
		{"x.py", `def f(x): return "s"  # c`, "kkk.f.....kkkkkk.sss..ccc"},
		{"x.rs", `fn main() { let x: i32 = 1; }`, "kk.ffff.....kkk....ttt...n..."},
	}
	for _, tt := range tests {
		t.Run(tt.path+" "+tt.text, func(t *testing.T) {
			h := For(tt.path, tt.text)
			if h == nil {
				t.Fatal("linguagem deveria ter suporte")
			}
			defer h.Close()
			if got := classes(h, 0, 0); got != tt.want {
				t.Fatalf("\n%s\n%s (obtido)\n%s (esperado)", tt.text, got, tt.want)
			}
		})
	}
	if For("notes.txt", "x") != nil || For("Makefile", "x") != nil {
		t.Fatal("arquivos sem gramática não têm highlighter")
	}
}

// Every edit, kept in step incrementally, must highlight exactly like a
// fresh parse of the resulting text.
func TestIncrementalMatchesFreshParse(t *testing.T) {
	ed := editor.New("int\tmain(void)\n{\n\treturn (0);\n}")
	h := For("main.c", ed.Buffer().Text())
	defer h.Close()
	ed.OnChange(h.Edit)

	check := func(step string) {
		t.Helper()
		fresh := For("main.c", ed.Buffer().Text())
		defer fresh.Close()
		n := ed.Buffer().LineCount() - 1
		if got, want := classes(h, 0, n), classes(fresh, 0, n); got != want {
			t.Fatalf("%s: incremental\n%s\nesperado\n%s", step, got, want)
		}
	}
	check("início")
	ed.MoveCursor(editor.Position{Line: 2, Column: 1})
	ed.Insert(`write(1, "olá", 4);` + "\n\t")
	check("insert com quebra e acento")
	ed.Insert("/* fim */")
	check("comentário")
	ed.Delete(editor.Range{Start: editor.Position{Line: 0, Column: 3}, End: editor.Position{Line: 1, Column: 1}})
	check("delete entre linhas")
	ed.Replace(editor.Range{Start: editor.Position{Line: 0, Column: 0}, End: editor.Position{Line: 0, Column: 3}}, "char")
	check("replace")
	for ed.Undo() {
		check("undo")
	}
	for ed.Redo() {
		check("redo")
	}
}

func TestLinesOutOfRange(t *testing.T) {
	h := For("a.c", "int x;\n")
	defer h.Close()
	if got := h.Lines(0, 99); len(got) != 2 || len(got[1]) != 0 {
		t.Fatalf("linhas além do fim são cortadas: %v", got)
	}
	if h.Lines(5, 9) != nil {
		t.Fatal("começar depois do fim não devolve nada")
	}
}
