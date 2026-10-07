package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBufferInsertDelete(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		at      Position
		insert  string
		want    string
		wantEnd Position
	}{
		{"meio da linha", "abcd", Position{0, 2}, "XY", "abXYcd", Position{0, 4}},
		{"várias linhas", "abcd", Position{0, 2}, "1\n2\n3", "ab1\n2\n3cd", Position{2, 1}},
		{"nova linha no fim", "ab", Position{0, 2}, "\n", "ab\n", Position{1, 0}},
		{"acentos e emoji contam 1", "olá 🚀!", Position{0, 5}, "x", "olá 🚀x!", Position{0, 6}},
		{"posição fora é ajustada", "ab\ncd", Position{9, 9}, "!", "ab\ncd!", Position{1, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuffer(tt.text)
			start := b.Clamp(tt.at)
			end := b.Insert(tt.at, tt.insert)
			if b.Text() != tt.want || end != tt.wantEnd {
				t.Fatalf("Insert = %q (fim %v), esperado %q (fim %v)", b.Text(), end, tt.want, tt.wantEnd)
			}
			// Deleting what was inserted gives the original text back.
			if got := b.Delete(Range{Start: end, End: start}); got != tt.insert || b.Text() != NewBuffer(tt.text).Text() {
				t.Fatalf("Delete devolveu %q e deixou %q", got, b.Text())
			}
		})
	}
}

func TestBufferBasics(t *testing.T) {
	b := NewBuffer("um\r\ndois\ntrês\n")
	if b.LineCount() != 4 || b.Line(2) != "três" || b.LineLen(2) != 4 || b.Line(3) != "" {
		t.Fatalf("linhas erradas: %d %q", b.LineCount(), b.Text())
	}
	if b.Text() != "um\ndois\ntrês\n" {
		t.Fatalf("CRLF deveria virar LF e o \\n final ser mantido: %q", b.Text())
	}
	if got := b.Slice(Range{Start: Position{0, 1}, End: Position{2, 2}}); got != "m\ndois\ntr" {
		t.Fatalf("Slice = %q", got)
	}
	if got := b.End(); got != (Position{3, 0}) {
		t.Fatalf("End = %v", got)
	}
}

func TestUndoRedoCoalescing(t *testing.T) {
	e := New("")
	for _, r := range "abc" {
		e.Insert(string(r))
	}
	if !e.Undo() || e.Buffer().Text() != "" || e.Cursor() != (Position{}) {
		t.Fatalf("digitação contínua deveria desfazer de uma vez: %q %v", e.Buffer().Text(), e.Cursor())
	}
	if !e.Redo() || e.Buffer().Text() != "abc" || e.Cursor() != (Position{0, 3}) {
		t.Fatalf("redo: %q %v", e.Buffer().Text(), e.Cursor())
	}

	// A cursor jump starts a new step.
	e.MoveCursor(Position{0, 0})
	e.Insert(">")
	e.Undo()
	if e.Buffer().Text() != "abc" {
		t.Fatalf("o salto deveria separar os passos: %q", e.Buffer().Text())
	}

	// Enter ends the step: one line of typing per undo.
	e = New("")
	for _, s := range []string{"a", "b", "\n", "c", "d"} {
		e.Insert(s)
	}
	e.Undo()
	if e.Buffer().Text() != "ab\n" {
		t.Fatalf("primeiro undo deveria tirar só a segunda linha: %q", e.Buffer().Text())
	}

	// A backspace run is one step too.
	e = New("abcd")
	e.MoveCursor(Position{0, 4})
	e.DeleteBackward()
	e.DeleteBackward()
	if e.Buffer().Text() != "ab" {
		t.Fatalf("backspace: %q", e.Buffer().Text())
	}
	e.Undo()
	if e.Buffer().Text() != "abcd" || e.Cursor() != (Position{0, 4}) {
		t.Fatalf("backspaces deveriam voltar juntos: %q %v", e.Buffer().Text(), e.Cursor())
	}

	// A new edit clears redo.
	e.Undo()
	e.Insert("z")
	if e.Redo() {
		t.Fatal("nova edição deveria limpar o redo")
	}
}

func TestReplaceIsOneStep(t *testing.T) {
	e := New("hello world")
	e.Replace(Range{Start: Position{0, 6}, End: Position{0, 11}}, "42")
	if e.Buffer().Text() != "hello 42" || e.Cursor() != (Position{0, 8}) {
		t.Fatalf("replace: %q %v", e.Buffer().Text(), e.Cursor())
	}
	e.Undo()
	if e.Buffer().Text() != "hello world" {
		t.Fatalf("um undo deveria desfazer o replace inteiro: %q", e.Buffer().Text())
	}
	e.Redo()
	if e.Buffer().Text() != "hello 42" {
		t.Fatalf("um redo deveria refazer o replace inteiro: %q", e.Buffer().Text())
	}
}

func TestDeleteJoinsLines(t *testing.T) {
	e := New("ab\ncd")
	e.MoveCursor(Position{1, 0})
	e.DeleteBackward()
	if e.Buffer().Text() != "abcd" || e.Cursor() != (Position{0, 2}) {
		t.Fatalf("backspace no início da linha: %q %v", e.Buffer().Text(), e.Cursor())
	}
	e.DeleteForward()
	if e.Buffer().Text() != "abd" {
		t.Fatalf("delete: %q", e.Buffer().Text())
	}
	e = New("ab\ncd")
	e.MoveCursor(Position{0, 2})
	e.DeleteForward()
	if e.Buffer().Text() != "abcd" {
		t.Fatalf("delete no fim da linha: %q", e.Buffer().Text())
	}
	// Nothing before the start, nothing after the end.
	e.MoveCursor(Position{})
	e.DeleteBackward()
	e.MoveCursor(e.Buffer().End())
	e.DeleteForward()
	if e.Buffer().Text() != "abcd" {
		t.Fatalf("bordas: %q", e.Buffer().Text())
	}
}

func TestMovementKeepsVisualColumn(t *testing.T) {
	// Tabs span 4 columns: "\tx" puts x at column 4.
	e := New("\tx = 1;\nab\n\tint y;")
	e.MoveCursor(Position{0, 1}) // on "x", screen column 4
	e.Move(MoveDown)
	if e.Cursor() != (Position{1, 2}) {
		t.Fatalf("linha curta: cursor deveria ir pro fim (1,2), está em %v", e.Cursor())
	}
	e.Move(MoveDown)
	if e.Cursor() != (Position{2, 1}) {
		t.Fatalf("deveria voltar à coluna visual 4 (depois do tab): %v", e.Cursor())
	}
	e.Move(MoveLineEnd)
	e.Move(MoveRight) // end of document: stays
	if e.Cursor() != (Position{2, 7}) {
		t.Fatalf("fim: %v", e.Cursor())
	}
	e.Move(MoveDocStart)
	e.Move(MoveLeft)
	if e.Cursor() != (Position{}) {
		t.Fatalf("início: %v", e.Cursor())
	}
	e.Move(MoveLineEnd)
	e.Move(MoveRight) // wraps to next line
	if e.Cursor() != (Position{1, 0}) {
		t.Fatalf("direita no fim da linha deveria ir pra próxima: %v", e.Cursor())
	}
}

func TestViewportFollowsCursor(t *testing.T) {
	var text string
	for i := 0; i < 100; i++ {
		text += "linha\n"
	}
	e := New(text)
	e.SetViewportSize(20, 10)
	e.MoveCursor(Position{50, 0})
	v := e.Viewport()
	if 50 < v.Top || 50 >= v.Top+v.Height {
		t.Fatalf("cursor fora da tela: %+v", v)
	}
	if 50 >= v.Top+v.Height-scrollMargin {
		t.Fatalf("deveria manter margem embaixo: %+v", v)
	}
	e.Move(MoveDocStart)
	if e.Viewport().Top != 0 {
		t.Fatalf("voltar ao topo: %+v", e.Viewport())
	}
	e.ScrollBy(500)
	if e.Viewport().Top != e.Buffer().LineCount()-1 {
		t.Fatalf("rolagem limitada ao fim: %+v", e.Viewport())
	}
}

func TestOpenSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rendu", "first_word", "first_word.c")
	e, err := Open(path) // missing file: empty, created on save
	if err != nil {
		t.Fatal(err)
	}
	if e.Buffer().Text() != "" || e.Dirty() {
		t.Fatal("arquivo novo deveria abrir vazio e limpo")
	}
	e.Insert("int main(void)\n{\n}\n")
	if !e.Dirty() {
		t.Fatal("edição deveria marcar alterações")
	}
	if err := e.Save(); err != nil {
		t.Fatal(err)
	}
	if e.Dirty() {
		t.Fatal("salvar deveria limpar as alterações")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "int main(void)\n{\n}\n" {
		t.Fatalf("arquivo salvo: %q %v", data, err)
	}
	again, err := Open(path)
	if err != nil || again.Buffer().Text() != string(data) {
		t.Fatalf("reabrir: %q %v", again.Buffer().Text(), err)
	}
	if err := New("x").Save(); err == nil {
		t.Fatal("documento sem arquivo não deveria salvar")
	}
}
