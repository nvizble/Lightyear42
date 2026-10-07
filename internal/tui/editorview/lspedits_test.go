package editorview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
)

func TestRenameReferencesActionsFormat(t *testing.T) {
	m := withFakeLSP(t, "int\tadd(int a);\nint\tx = add(1);")
	mainc := m.Editor().Path()
	other := filepath.Join(filepath.Dir(mainc), "other.c")
	if err := os.WriteFile(other, []byte("int\tadd(int a)\n{\n\treturn (a);\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _ = press(t, m, runes("jw"))
	cursor := m.Editor().Cursor()

	// grn opens :Rename in the command line.
	m, _ = press(t, m, runes("grn"))
	if s := screen(m); !strings.Contains(s, ":Rename ") {
		t.Fatalf("grn: %q", s)
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// The rename's edits: two files; the buffer and the cursor stay.
	at := func(l, c int) lsp.Pos { return lsp.Pos{Line: l, Col: c} }
	m, _ = m.editReply(renameMsg{edits: []lsp.FileEdit{
		{Path: mainc, Edits: []lsp.TextEdit{{Start: at(0, 4), End: at(0, 7), Text: "soma"}, {Start: at(1, 8), End: at(1, 11), Text: "soma"}}},
		{Path: other, Edits: []lsp.TextEdit{{Start: at(0, 4), End: at(0, 7), Text: "soma"}}},
	}})
	if got := m.Editor().Buffer().Text(); got != "int\tsoma(int a);\nint\tx = soma(1);" || m.Editor().Cursor() != cursor {
		t.Fatalf("rename no arquivo atual: %q %v", got, m.Editor().Cursor())
	}
	if !strings.Contains(m.message, "3 mudança(s) em 2 arquivo(s) (:wa salva)") || len(m.ses.bufs) != 2 || !m.ses.bufs[1].ed.Dirty() {
		t.Fatalf("rename em dois arquivos: %q", m.message)
	}
	if !strings.HasPrefix(m.ses.bufs[1].ed.Buffer().Text(), "int\tsoma(int a)") {
		t.Fatalf("o outro arquivo, num buffer: %q", m.ses.bufs[1].ed.Buffer().Text())
	}
	m, _ = press(t, m, runes("u"))
	if got := m.Editor().Buffer().Text(); got != "int\tadd(int a);\nint\tx = add(1);" {
		t.Fatalf("um u desfaz o rename no arquivo: %q", got)
	}

	// References: a list to pick from; Enter jumps, Ctrl-o comes back.
	m, _ = m.editReply(referencesMsg{locs: []lsp.Location{{Path: mainc, Pos: at(0, 4)}, {Path: other, Pos: at(0, 4)}}})
	if s := screen(m); !strings.Contains(s, "2 referências") || !strings.Contains(s, "other.c:1  int\tadd(int a)") && !strings.Contains(s, "other.c:1") {
		t.Fatalf("lista de referências:\n%s", s)
	}
	m, _ = press(t, m, runes("j"), tea.KeyMsg{Type: tea.KeyEnter})
	if filepath.Base(m.Editor().Path()) != "other.c" || m.Editor().Cursor() != (editor.Position{Line: 0, Column: 4}) || m.ses.pick != nil {
		t.Fatalf("enter vai à referência: %s %v", m.Editor().Path(), m.Editor().Cursor())
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlO})
	if filepath.Base(m.Editor().Path()) != "main.c" {
		t.Fatalf("ctrl+o volta: %s", m.Editor().Path())
	}

	// A code action with edits; Esc closes the list without doing anything.
	fix := lsp.Action{Title: "insert ';'", Edits: []lsp.FileEdit{{Path: mainc, Edits: []lsp.TextEdit{{Start: at(1, 15), End: at(1, 15), Text: "//ok"}}}}}
	m, _ = m.editReply(actionsMsg{actions: []lsp.Action{fix}})
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.ses.pick != nil || strings.Contains(m.Editor().Buffer().Text(), "//ok") {
		t.Fatal("esc fecha a lista sem aplicar")
	}
	m, _ = m.editReply(actionsMsg{actions: []lsp.Action{fix}})
	if !strings.Contains(screen(m), "insert ';'") {
		t.Fatalf("lista de ações:\n%s", screen(m))
	}
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.HasSuffix(m.Editor().Buffer().Text(), "add(1);//ok") {
		t.Fatalf("a ação aplica a edição: %q", m.Editor().Buffer().Text())
	}

	// Formatting C needs a .clang-format (the default style breaks the norm).
	m, _ = press(t, m, runes(":Format"), tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.message, ".clang-format") {
		t.Fatalf(":Format sem .clang-format: %q", m.message)
	}
	m, _ = m.editReply(formatMsg{path: mainc, edits: []lsp.TextEdit{{Start: at(0, 0), End: at(0, 0), Text: "// fmt\n"}}})
	if !strings.HasPrefix(m.Editor().Buffer().Text(), "// fmt\nint") {
		t.Fatalf("format aplica: %q", m.Editor().Buffer().Text())
	}
}
