package editorview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
)

func TestSpread(t *testing.T) {
	tests := []struct {
		total  int
		shares []float64
		least  int
		want   string
	}{
		{100, []float64{1, 1}, 12, "[50 50]"},
		{79, []float64{1, 1}, 12, "[40 39]"},
		{80, []float64{1, 3}, 12, "[20 60]"},
		{30, []float64{0.1, 1}, 12, "[12 18]"},     // the least wins over the share
		{10, []float64{1, 1}, 12, "[5 5]"},         // no room for the least: just split
		{90, []float64{1, 1, 1}, 12, "[30 30 30]"}, // three windows
	}
	for _, tt := range tests {
		if got := fmt.Sprint(spread(tt.total, tt.shares, tt.least)); got != tt.want {
			t.Errorf("spread(%d, %v): %s, esperado %s", tt.total, tt.shares, got, tt.want)
		}
	}
}

// sideBySide is the exam layout on an 80-column screen: the subject on the
// left, the code (active) on the right.
func sideBySide(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	subject := filepath.Join(dir, "subject.txt")
	if err := os.WriteFile(subject, []byte("Assignment name: first_word\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := NewSideBySide(subject, []string{filepath.Join(dir, "first_word.c")})
	if err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	return next.(Model)
}

func widths(m Model) string {
	var w []int
	for _, r := range m.rects() {
		w = append(w, r.w)
	}
	return fmt.Sprint(w)
}

func TestResizeWithKeys(t *testing.T) {
	m := sideBySide(t)
	if widths(m) != "[40 39]" {
		t.Fatalf("começa meio a meio: %s", widths(m))
	}
	ctrlW := tea.KeyMsg{Type: tea.KeyCtrlW}
	// 10 Ctrl-w > grows the code (the last window takes from the subject).
	m, _ = press(t, m, runes("10"), ctrlW, runes(">"))
	if widths(m) != "[30 49]" {
		t.Fatalf("10 ctrl+w >: %s", widths(m))
	}
	m, _ = press(t, m, runes("5"), ctrlW, runes("<"))
	if widths(m) != "[35 44]" {
		t.Fatalf("5 ctrl+w <: %s", widths(m))
	}
	m, _ = press(t, m, ctrlW, runes("|"))
	if widths(m) != "[12 67]" {
		t.Fatalf("ctrl+w | deixa o subject no mínimo: %s", widths(m))
	}
	m, _ = press(t, m, ctrlW, runes("="))
	if widths(m) != "[40 39]" {
		t.Fatalf("ctrl+w = iguala: %s", widths(m))
	}
	m, _ = press(t, m, ctrlW, runes("+"))
	if widths(m) != "[40 39]" {
		t.Fatalf("ctrl+w + (altura) não mexe lado a lado: %s", widths(m))
	}
	m, _ = press(t, m, runes(":vertical resize 50"), tea.KeyMsg{Type: tea.KeyEnter})
	if widths(m) != "[29 50]" {
		t.Fatalf(":vertical resize 50: %s", widths(m))
	}
	m, _ = press(t, m, runes(":vert res -10"), tea.KeyMsg{Type: tea.KeyEnter})
	if widths(m) != "[39 40]" {
		t.Fatalf(":vert res -10: %s", widths(m))
	}
	if m, _ = press(t, m, runes(":resize x"), tea.KeyMsg{Type: tea.KeyEnter}); !m.isError {
		t.Fatal(":resize sem número deveria explicar o uso")
	}
	// The terminal grows: the windows keep their proportion.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 10})
	m = next.(Model)
	if widths(m) != "[78 81]" { // 159 × 39/79 = 78.5
		t.Fatalf("o terminal cresce e a proporção fica: %s", widths(m))
	}
}

func TestResizeWithTheMouse(t *testing.T) {
	m := sideBySide(t)
	left := tea.MouseButtonLeft
	// The border is the │ right after the subject (column 40).
	if string([]rune(ansi.Strip(strings.Split(m.View(), "\n")[0]))[40:43]) != "│1 " {
		t.Fatalf("a borda deveria estar na coluna 40: %q", ansi.Strip(strings.Split(m.View(), "\n")[0]))
	}
	m, _ = press(t, m,
		tea.MouseMsg{X: 40, Y: 3, Action: tea.MouseActionPress, Button: left},
		tea.MouseMsg{X: 25, Y: 3, Action: tea.MouseActionMotion, Button: left})
	if widths(m) != "[25 54]" || m.ses.drag == nil {
		t.Fatalf("arrastar a borda: %s", widths(m))
	}
	if !strings.Contains(m.View(), styleGutterHere.Render("│")) {
		t.Fatal("a borda arrastada fica em destaque")
	}
	m, _ = press(t, m, tea.MouseMsg{X: 2, Y: 3, Action: tea.MouseActionMotion, Button: left})
	if widths(m) != "[12 67]" {
		t.Fatalf("arrastar até a beira para no mínimo: %s", widths(m))
	}
	m, _ = press(t, m,
		tea.MouseMsg{X: 30, Y: 3, Action: tea.MouseActionMotion, Button: left},
		tea.MouseMsg{X: 30, Y: 3, Action: tea.MouseActionRelease, Button: left},
		tea.MouseMsg{X: 50, Y: 3, Action: tea.MouseActionMotion, Button: left})
	if widths(m) != "[30 49]" || m.ses.drag != nil {
		t.Fatalf("soltar termina o arraste: %s", widths(m))
	}
	// A press in the text still places the cursor (the code, on the right).
	m, _ = press(t, m, tea.MouseMsg{X: 60, Y: 0, Action: tea.MouseActionPress, Button: left})
	if filepath.Base(m.Editor().Path()) != "first_word.c" || m.ses.drag != nil {
		t.Fatalf("clique no texto não é arraste: %s", m.Editor().Path())
	}

	// The layout can be kept for a later editor.
	shares := m.Shares()
	again := sideBySide(t).WithShares(shares)
	if widths(again) != "[30 49]" {
		t.Fatalf("WithShares repete o layout: %s", widths(again))
	}
}

func TestResizeStacked(t *testing.T) {
	next, _ := NewVim(editor.New("a\nb\nc")).Update(tea.WindowSizeMsg{Width: 60, Height: 13})
	m := next.(Model)
	m, _ = press(t, m, runes(":sp"), tea.KeyMsg{Type: tea.KeyEnter})
	heights := func() string {
		var h []int
		for _, r := range m.rects() {
			h = append(h, r.h)
		}
		return fmt.Sprint(h)
	}
	if heights() != "[5 5]" {
		t.Fatalf(":sp divide a altura: %s", heights())
	}
	m, _ = press(t, m, runes("2"), tea.KeyMsg{Type: tea.KeyCtrlW}, runes("+"))
	if heights() != "[7 3]" {
		t.Fatalf("2 ctrl+w +: %s", heights())
	}
	// Dragging the title line under the top window.
	m, _ = press(t, m,
		tea.MouseMsg{X: 10, Y: 7, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		tea.MouseMsg{X: 10, Y: 3, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft},
		tea.MouseMsg{X: 10, Y: 3, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	if heights() != "[3 7]" {
		t.Fatalf("arrastar o título: %s", heights())
	}
	m, _ = press(t, m, runes(":resize 4"), tea.KeyMsg{Type: tea.KeyEnter})
	if heights() != "[4 6]" {
		t.Fatalf(":resize 4: %s", heights())
	}
	// Closing a window gives its space back.
	m, _ = press(t, m, runes(":close"), tea.KeyMsg{Type: tea.KeyEnter})
	if heights() != "[12]" {
		t.Fatalf(":close devolve o espaço: %s", heights())
	}
}
