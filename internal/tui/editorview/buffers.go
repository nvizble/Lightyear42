package editorview

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/syntax"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// Buffers: the editor can hold several files. :e opens one (gd too, when the
// definition is elsewhere), :bn/:bp/:b N switch, :ls lists, :bd closes, and
// Ctrl-o goes back to where gd jumped from.

// buffer is one open file.
type buffer struct {
	ed  *editor.Editor
	syn *syntax.Highlighter // nil without a grammar
	lsp *lspState           // nil without a language server
}

// session is what the model's copies share.
type session struct {
	bufs     []*buffer
	cur      int
	lsp      bool                  // WithLSP: buffers get language servers
	servers  map[string]*lspServer // by server name and project root
	wins     []*window             // see windows.go
	win      int                   // the current window
	vertical bool                  // windows side by side (:vsp), not stacked
	starts   []tea.Cmd             // servers to start, for Update to return
	jumps    []jump                // where gd jumped from, for Ctrl-o
	pick     *picker               // an open list to choose from (lspedits.go)
	tip      *tip                  // the diagnostics under the mouse (diagtip.go)
	drag     *drag                 // a window border being dragged (resize.go)
	// width and height are the screen's, for the window layout: commands
	// the controller runs see the session, not the latest model.
	width, height int
}

type jump struct {
	buf *buffer
	pos editor.Position
}

func newBuffer(ed *editor.Editor) *buffer {
	b := &buffer{ed: ed, syn: syntax.For(ed.Path(), ed.Buffer().Text())}
	if b.syn != nil {
		ed.OnChange(b.syn.Edit)
	}
	return b
}

func (ses *session) current() *buffer { return ses.bufs[ses.cur] }

// synced points the model at the current buffer.
func (m Model) synced() Model {
	b := m.ses.current()
	m.ed, m.syn, m.lsp = b.ed, b.syn, b.lsp
	return m
}

// show makes buffer i the current one, in the current window.
func (m Model) show(i int) {
	m.ses.cur = i
	m.ses.wins[m.ses.win].buf = m.ses.bufs[i]
	if m.vim != nil {
		m.vim.SetEditor(m.ses.bufs[i].ed)
	}
}

// open switches to the buffer of path, opening the file when it isn't open.
func (m Model) open(path string) error {
	for i, b := range m.ses.bufs {
		if b.ed.Path() == path || sameFile(b.ed.Path(), path) {
			m.show(i)
			return nil
		}
	}
	ed, err := editor.Open(path)
	if err != nil {
		return err
	}
	b := newBuffer(ed)
	m.ses.bufs = append(m.ses.bufs, b)
	m.ses.attach(b)
	m.show(len(m.ses.bufs) - 1)
	return nil
}

// NewSideBySide is a Vim-style editor with files on the right (the first
// one shown, active) and ref read-only on the left: an exam's subject next
// to the code. :wq there saves and quits (the read-only window doesn't hold
// it open).
func NewSideBySide(ref string, files []string) (Model, error) {
	if len(files) == 0 {
		return Model{}, errors.New("nenhum arquivo para editar")
	}
	ed, err := editor.Open(files[0])
	if err != nil {
		return Model{}, err
	}
	m := NewVim(ed)
	for _, f := range files[1:] {
		if err := m.open(f); err != nil {
			return Model{}, err
		}
	}
	m.show(0)
	if res := m.split(true, ref); res.Err {
		return Model{}, errors.New(res.Message)
	}
	m.ses.current().ed.SetReadOnly(true)
	m.focus(1)
	return m.synced(), nil
}

// unsaved is the name of a modified buffer ("" when none).
func (ses *session) unsaved() string {
	for _, b := range ses.bufs {
		if b.ed.Dirty() {
			return filepath.Base(b.ed.Path())
		}
	}
	return ""
}

// Open opens path in another buffer and shows it.
func (m Model) Open(path string) (Model, error) {
	err := m.open(path)
	return m.synced(), err
}

// jumpTo moves to pos in the file at path (opening it), remembering where
// it was for Ctrl-o.
func (m Model) jumpTo(path string, pos editor.Position) error {
	m.ses.jumps = append(m.ses.jumps, jump{m.ses.current(), m.ses.current().ed.Cursor()})
	if err := m.open(path); err != nil {
		return err
	}
	m.moveCursor(pos)
	return nil
}

func (m Model) moveCursor(p editor.Position) {
	if m.vim != nil {
		m.vim.MoveCursor(p)
	} else {
		m.ses.current().ed.MoveCursor(p)
	}
}

// otherEditableWindow reports a window besides the current one showing a
// file that can be edited.
func (m Model) otherEditableWindow() bool {
	for i, w := range m.ses.wins {
		if i != m.ses.win && !w.buf.ed.ReadOnly() {
			return true
		}
	}
	return false
}

// back is Ctrl-o: to where the last jump left from.
func (m Model) back(int) vim.Result {
	for len(m.ses.jumps) > 0 {
		j := m.ses.jumps[len(m.ses.jumps)-1]
		m.ses.jumps = m.ses.jumps[:len(m.ses.jumps)-1]
		for i, b := range m.ses.bufs {
			if b == j.buf {
				m.show(i)
				m.moveCursor(j.pos)
				return vim.Result{}
			}
		}
	}
	return vim.Result{Message: "nenhum salto para voltar"}
}

// exCommands are the ex commands for buffers, and the ones that quit or
// save, which have to look at every buffer.
func (m Model) exCommands() map[string]func(string) vim.Result {
	ses := m.ses
	edit := func(arg string) vim.Result {
		if arg == "" {
			return vim.Result{Message: "E32: falta o nome do arquivo (:e arquivo)", Err: true}
		}
		if err := m.open(arg); err != nil {
			return vim.Result{Message: err.Error(), Err: true}
		}
		return vim.Result{}
	}
	step := func(d int) func(string) vim.Result {
		return func(string) vim.Result {
			m.show((ses.cur + d + len(ses.bufs)) % len(ses.bufs))
			return vim.Result{}
		}
	}
	pick := func(arg string) vim.Result {
		if n, err := strconv.Atoi(arg); err == nil && n >= 1 && n <= len(ses.bufs) {
			m.show(n - 1)
			return vim.Result{}
		}
		match := -1
		for i, b := range ses.bufs {
			if arg != "" && strings.Contains(filepath.Base(b.ed.Path()), arg) {
				if match >= 0 {
					return vim.Result{Message: "E93: mais de um buffer para " + arg, Err: true}
				}
				match = i
			}
		}
		if match < 0 {
			return vim.Result{Message: "E94: nenhum buffer " + arg, Err: true}
		}
		m.show(match)
		return vim.Result{}
	}
	list := func(string) vim.Result {
		var parts []string
		for i, b := range ses.bufs {
			mark := " "
			if i == ses.cur {
				mark = "%"
			}
			name := filepath.Base(b.ed.Path())
			if b.ed.Dirty() {
				name += " [+]"
			}
			parts = append(parts, fmt.Sprintf("%d%s %s", i+1, mark, name))
		}
		return vim.Result{Message: strings.Join(parts, "  ")}
	}
	del := func(force bool) func(string) vim.Result {
		return func(string) vim.Result {
			b := ses.current()
			switch {
			case b.ed.Dirty() && !force:
				return vim.Result{Message: "E89: " + filepath.Base(b.ed.Path()) + " tem alterações (:w salva, :bd! descarta)", Err: true}
			case len(ses.bufs) == 1:
				return vim.Result{Message: "é o único buffer (:q sai)"}
			}
			if b.lsp != nil && b.lsp.open && b.lsp.srv.client != nil {
				b.lsp.srv.client.DidClose(b.lsp.path)
			}
			if b.syn != nil {
				b.syn.Close()
			}
			ses.bufs = append(ses.bufs[:ses.cur], ses.bufs[ses.cur+1:]...)
			next := ses.bufs[min(ses.cur, len(ses.bufs)-1)]
			for _, w := range ses.wins {
				if w.buf == b {
					w.buf = next
				}
			}
			m.show(min(ses.cur, len(ses.bufs)-1))
			return vim.Result{}
		}
	}
	// unsaved is a modified buffer other than the current one ("" when none).
	unsaved := func() string {
		for i, b := range ses.bufs {
			if i != ses.cur && b.ed.Dirty() {
				return filepath.Base(b.ed.Path())
			}
		}
		return ""
	}
	quit := func(save bool) func(string) vim.Result {
		return func(string) vim.Result {
			cur := ses.current().ed
			if save {
				if err := cur.Save(); err != nil {
					return vim.Result{Message: err.Error(), Err: true}
				}
			}
			// With several windows, :q closes this one (the buffer stays),
			// unless only read-only ones would be left (a subject next to the
			// code): then it quits.
			if m.otherEditableWindow() {
				return m.closeWindow()
			}
			if cur.Dirty() {
				return vim.Result{Message: "E37: alterações não salvas (:wq salva e sai, :q! sai sem salvar)", Err: true}
			}
			if name := unsaved(); name != "" {
				return vim.Result{Message: "E162: " + name + " tem alterações (:wa salva tudo, :qa! sai sem salvar)", Err: true}
			}
			return vim.Result{Quit: true}
		}
	}
	quitAll := func(string) vim.Result {
		if ses.current().ed.Dirty() {
			return quit(false)("")
		}
		if name := unsaved(); name != "" {
			return vim.Result{Message: "E162: " + name + " tem alterações (:wa salva tudo, :qa! sai sem salvar)", Err: true}
		}
		return vim.Result{Quit: true}
	}
	force := func(string) vim.Result { return vim.Result{Quit: true} }
	saveAll := func(string) vim.Result {
		n := 0
		for _, b := range ses.bufs {
			if b.ed.Dirty() {
				if err := b.ed.Save(); err != nil {
					return vim.Result{Message: err.Error(), Err: true}
				}
				n++
			}
		}
		return vim.Result{Message: fmt.Sprintf("%d arquivo(s) salvo(s)", n)}
	}
	// :vertical resize N; the other :vertical commands aren't here.
	vertical := func(arg string) vim.Result {
		name, rest, _ := strings.Cut(arg, " ")
		if name != "resize" && name != "res" {
			return vim.Result{Message: "E492: só :vertical resize por enquanto", Err: true}
		}
		return m.resizeTo(true, strings.TrimSpace(rest))
	}
	saveQuit := func(string) vim.Result {
		if res := saveAll(""); res.Err {
			return res
		}
		return vim.Result{Quit: true}
	}
	return map[string]func(string) vim.Result{
		"e": edit, "edit": edit,
		"bn": step(1), "bnext": step(1), "bp": step(-1), "bprev": step(-1), "bprevious": step(-1),
		"b": pick, "buffer": pick,
		"ls": list, "buffers": list, "files": list,
		"bd": del(false), "bdelete": del(false), "bd!": del(true), "bdelete!": del(true),
		"q": quit(false), "quit": quit(false), "wq": quit(true), "x": quit(true),
		"qa": quitAll, "qall": quitAll,
		"q!": force, "quit!": force, "qa!": force, "qall!": force,
		"wa": saveAll, "wall": saveAll,
		"wqa": saveQuit, "wqall": saveQuit, "xa": saveQuit, "xall": saveQuit,
		"sp":     func(arg string) vim.Result { return m.split(false, arg) },
		"split":  func(arg string) vim.Result { return m.split(false, arg) },
		"vs":     func(arg string) vim.Result { return m.split(true, arg) },
		"vsp":    func(arg string) vim.Result { return m.split(true, arg) },
		"vsplit": func(arg string) vim.Result { return m.split(true, arg) },
		"clo":    func(string) vim.Result { return m.closeWindow() },
		"close":  func(string) vim.Result { return m.closeWindow() },
		"on":     func(string) vim.Result { return m.only() },
		"only":   func(string) vim.Result { return m.only() },
		"res":    func(arg string) vim.Result { return m.resizeTo(false, arg) },
		"resize": func(arg string) vim.Result { return m.resizeTo(false, arg) },
		"vert":   vertical, "vertical": vertical,
	}
}
