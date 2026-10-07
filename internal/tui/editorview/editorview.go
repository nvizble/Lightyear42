// Package editorview is the Bubble Tea component of lightyear's embedded
// editor: it draws an editor.Editor (line numbers, text, cursor and status
// line) and turns keys and mouse into editor operations.
//
// New is a plain (non-modal) editor; NewVim hands keys to the Vim-like
// controller in internal/vim (see internal/editor/DESIGN.md).
package editorview

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/lsp"
	"github.com/nvizble/Lightyear42/internal/syntax"
	"github.com/nvizble/Lightyear42/internal/vim"
)

var (
	colorMuted  = lipgloss.AdaptiveColor{Light: "243", Dark: "245"}
	colorAccent = lipgloss.AdaptiveColor{Light: "25", Dark: "39"}
	colorGood   = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}
	colorFail   = lipgloss.AdaptiveColor{Light: "124", Dark: "196"}
	colorVisual = lipgloss.AdaptiveColor{Light: "127", Dark: "170"}

	styleGutter     = lipgloss.NewStyle().Foreground(colorMuted)
	styleGutterHere = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleTilde      = lipgloss.NewStyle().Foreground(colorAccent)
	styleCursor     = lipgloss.NewStyle().Reverse(true)
	styleSelection  = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "153", Dark: "24"})
	styleSearchHit  = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "229", Dark: "58"})
	styleMode       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(colorGood).Padding(0, 1)
	styleModeNormal = styleMode.Background(colorAccent)
	styleModeVisual = styleMode.Background(colorVisual)
	stylePending    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "130", Dark: "214"})
	styleStatus     = lipgloss.NewStyle().Foreground(colorMuted)
	styleFile       = lipgloss.NewStyle().Bold(true)
	styleError      = lipgloss.NewStyle().Foreground(colorFail)

	// syntaxStyles color the code, by syntax.Class (Plain stays as is).
	syntaxStyles = [...]lipgloss.Style{
		syntax.Keyword:  lipgloss.NewStyle().Foreground(colorVisual),
		syntax.String:   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "114"}),
		syntax.Comment:  lipgloss.NewStyle().Foreground(colorMuted).Italic(true),
		syntax.Number:   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "130", Dark: "215"}),
		syntax.Function: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "25", Dark: "75"}),
		syntax.Type:     lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "30", Dark: "80"}),
	}
)

// Model is the editor component.
type Model struct {
	// ses holds the buffers and language servers, shared by the model's
	// copies; ed, syn and lsp are the current buffer's (see synced).
	ses           *session
	ed            *editor.Editor
	width, height int
	// message is the status line note (saved, nothing to undo, errors).
	message string
	isError bool
	// quitArmed is set after Ctrl-Q with unsaved changes; a second Ctrl-Q quits.
	quitArmed bool
	done      bool
	// vim, when set, interprets keys Vim style (modes, :w, :q).
	vim *vim.Controller
	// syn colors the code (nil for files without a grammar).
	syn *syntax.Highlighter
	// lsp is the buffer's language server side (nil without one).
	lsp *lspState
}

// New wraps an editor as a plain (non-modal) editor.
func New(ed *editor.Editor) Model {
	ses := &session{bufs: []*buffer{newBuffer(ed)}, servers: map[string]*lspServer{}}
	return Model{ses: ses}.synced()
}

// NewVim wraps an editor with Vim-style modal editing, with buffers (:e,
// :bn...).
func NewVim(ed *editor.Editor) Model {
	m := New(ed)
	m.vim = vim.New(ed)
	m.vim.ExCommands = m.exCommands()
	m.vim.Commands = map[string]func(int) vim.Result{"ctrl+o": m.back}
	return m
}

// Close stops the language servers and frees the syntax highlighters; call
// it once the editor is closed.
func (m Model) Close() {
	m.closeLSP()
	for _, b := range m.ses.bufs {
		if b.syn != nil {
			b.syn.Close()
		}
	}
}

// Editor is the document being edited.
func (m Model) Editor() *editor.Editor { return m.ed }

// Done reports that the user closed the editor.
func (m Model) Done() bool { return m.done }

// Init implements tea.Model: it starts the language servers, if any.
func (m Model) Init() tea.Cmd {
	starts := m.ses.starts
	m.ses.starts = nil
	return tea.Batch(starts...)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	m = next.(Model).synced()
	// Opening a buffer of a new project starts its server.
	if starts := m.ses.starts; len(starts) > 0 {
		m.ses.starts = nil
		cmd = tea.Batch(append(starts, cmd)...)
	}
	return m, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ed.SetViewportSize(m.textWidth(), m.textHeight())
	case tea.KeyMsg:
		next, cmd := m.key(msg)
		m = next.(Model).synced() // the key may have switched buffers
		m.syncLSP()
		if st := m.lsp; st != nil {
			cmd = tea.Batch(cmd, st.pending, m.afterKey(msg))
			st.pending = nil
		}
		return m, cmd
	case tea.MouseMsg:
		m.mouse(msg)
	case lspStartedMsg, lspEventMsg:
		return m.lspMsg(msg)
	case hoverMsg, definitionMsg, completionMsg:
		return m.lspReply(msg), nil
	}
	return m, nil
}

func (m Model) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k != "ctrl+q" {
		m.quitArmed = false
	}
	m.message, m.isError = "", false
	if st := m.lsp; st != nil {
		st.hover = nil // any key closes the hover
		if st.comp != nil && m.typing() && m.completionKey(k) {
			return m, nil
		}
	}

	if m.vim != nil && k != "ctrl+s" && k != "ctrl+q" {
		var res vim.Result
		switch msg.Type {
		case tea.KeyRunes:
			res = m.vim.HandleText(string(msg.Runes))
		case tea.KeySpace:
			res = m.vim.HandleKey(" ")
		default:
			res = m.vim.HandleKey(k)
		}
		m.message, m.isError = res.Message, res.Err
		if res.Quit {
			m.done = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyRunes:
		// Pastes may carry their own line breaks.
		m.ed.Insert(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(string(msg.Runes)))
		return m, nil
	case tea.KeySpace:
		m.ed.Insert(" ")
		return m, nil
	case tea.KeyTab:
		m.ed.Insert("\t")
		return m, nil
	case tea.KeyEnter:
		m.ed.Insert("\n" + m.indentation())
		return m, nil
	}

	switch k {
	case "backspace":
		m.ed.DeleteBackward()
	case "delete":
		m.ed.DeleteForward()
	case "left":
		m.ed.Move(editor.MoveLeft)
	case "right":
		m.ed.Move(editor.MoveRight)
	case "up":
		m.ed.Move(editor.MoveUp)
	case "down":
		m.ed.Move(editor.MoveDown)
	case "home":
		m.ed.Move(editor.MoveLineStart)
	case "end":
		m.ed.Move(editor.MoveLineEnd)
	case "ctrl+home":
		m.ed.Move(editor.MoveDocStart)
	case "ctrl+end":
		m.ed.Move(editor.MoveDocEnd)
	case "pgup":
		m.ed.Move(editor.MovePageUp)
	case "pgdown":
		m.ed.Move(editor.MovePageDown)
	case "ctrl+z":
		if !m.ed.Undo() {
			m.message = "nada para desfazer"
		}
	case "ctrl+y":
		if !m.ed.Redo() {
			m.message = "nada para refazer"
		}
	case "ctrl+s":
		if err := m.ed.Save(); err != nil {
			m.message, m.isError = err.Error(), true
		} else {
			m.message = fmt.Sprintf("salvo: %s (%d linhas)", m.ed.Path(), m.ed.Buffer().LineCount())
		}
	case "ctrl+q":
		if m.ed.Dirty() && !m.quitArmed {
			m.quitArmed = true
			m.message, m.isError = "alterações não salvas: ctrl+s salva, ctrl+q de novo sai sem salvar", true
			return m, nil
		}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

// indentation is the leading whitespace of the cursor's line, carried over
// to a new line (Enter).
func (m Model) indentation() string {
	line := m.ed.Buffer().Line(m.ed.Cursor().Line)
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func (m *Model) mouse(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.ed.ScrollBy(-3)
		return
	case tea.MouseButtonWheelDown:
		m.ed.ScrollBy(3)
		return
	}
	if msg.Button != tea.MouseButtonLeft || msg.Y >= m.textHeight() {
		return
	}
	v := m.ed.Viewport()
	line := v.Top + msg.Y
	visual := max(msg.X-m.gutterWidth(), 0) + v.Left
	p := editor.Position{Line: line, Column: m.ed.ColumnAt(line, visual)}
	switch {
	case msg.Action == tea.MouseActionMotion:
		// Dragging selects (Vim mode), from where the button went down.
		if m.vim != nil {
			m.vim.ExtendSelection(p)
		}
	case msg.Action != tea.MouseActionPress:
	case m.vim != nil:
		m.vim.MoveCursor(p)
	default:
		m.ed.MoveCursor(p)
	}
}

// Layout: text area above a one-line status bar, line numbers on the left.
func (m Model) textHeight() int { return max(m.height-1, 1) }
func (m Model) textWidth() int  { return max(m.width-m.gutterWidth(), 1) }
func (m Model) gutterWidth() int {
	return len(fmt.Sprint(m.ed.Buffer().LineCount())) + 3
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	buf, v, cur := m.ed.Buffer(), m.ed.Viewport(), m.ed.Cursor()
	numWidth := m.gutterWidth() - 3

	var classes [][]syntax.Class
	if m.syn != nil {
		classes = m.syn.Lines(v.Top, v.Top+m.textHeight()-1)
	}
	rows := make([]string, 0, m.textHeight()+1)
	for r := 0; r < m.textHeight(); r++ {
		line := v.Top + r
		if line >= buf.LineCount() {
			rows = append(rows, styleTilde.Render(fmt.Sprintf("%*s", numWidth, "~")))
			continue
		}
		gutter := styleGutter.Render(fmt.Sprintf("%*d │ ", numWidth, line+1))
		if line == cur.Line {
			gutter = styleGutterHere.Render(fmt.Sprintf("%*d", numWidth, line+1)) + styleGutter.Render(" │ ")
		}
		if d := m.worst(line); d != nil {
			gutter = severityStyles[d.Severity].Render(fmt.Sprintf("%*d ● ", numWidth, line+1))
		}
		look := lineLook{cursor: -1, selFrom: -1, selTo: -1, marks: m.marks(line, buf.LineLen(line))}
		if line == cur.Line {
			look.cursor = m.ed.VisualColumn(line, cur.Column)
		}
		if from, to, ok := m.ed.SelectedColumns(line); ok {
			look.selFrom = m.ed.VisualColumn(line, from)
			look.selTo = max(m.ed.VisualColumn(line, to), look.selFrom+1) // an empty line still shows a cell
		}
		if r < len(classes) {
			look.classes = classes[r]
		}
		if m.vim != nil {
			if starts, n := m.vim.Hits(buf.Line(line)); len(starts) > 0 {
				look.hits = make([]bool, buf.LineLen(line))
				for _, s := range starts {
					for i := s; i < s+n; i++ {
						look.hits[i] = true
					}
				}
			}
		}
		rows = append(rows, gutter+renderLine(buf.Line(line), m.ed.TabSize(), v.Left, m.textWidth(), look))
	}
	m.popups(rows)
	rows = append(rows, m.statusLine())
	return strings.Join(rows, "\n")
}

// lineLook is how to draw one line: the cursor and the selection in screen
// columns (-1 for none), and by rune the syntax classes and the
// diagnostics' severities (0 for none).
type lineLook struct {
	cursor, selFrom, selTo int
	classes                []syntax.Class
	marks                  []lsp.Severity
	hits                   []bool // search matches, by rune
}

// renderLine expands tabs, shows the visible slice [left, left+width) and
// draws it as look says: cursor, then selection, search matches,
// diagnostics and syntax.
func renderLine(line string, tabSize, left, width int, look lineLook) string {
	var cells []string
	var cellClasses []syntax.Class
	var cellMarks []lsp.Severity
	var cellHits []bool
	runes := []rune(line)
	// Marks may run past the end (a missing ";"): blank cells show them.
	for i := 0; i < max(len(runes), len(look.marks)); i++ {
		r, class, mark, hit := ' ', syntax.Plain, lsp.Severity(0), false
		if i < len(runes) {
			r = runes[i]
		}
		if i < len(look.classes) {
			class = look.classes[i]
		}
		if i < len(look.marks) {
			mark = look.marks[i]
		}
		if i < len(look.hits) {
			hit = look.hits[i]
		}
		cell, n := string(r), 1
		if r == '\t' {
			cell, n = " ", tabSize-len(cells)%tabSize
		}
		for ; n > 0; n-- {
			cells = append(cells, cell)
			cellClasses = append(cellClasses, class)
			cellMarks = append(cellMarks, mark)
			cellHits = append(cellHits, hit)
		}
	}
	// The cursor or the selection may sit past the end (Insert, empty lines).
	for len(cells) < max(look.cursor+1, look.selTo) {
		cells = append(cells, " ")
	}

	// Cells with the same look are styled together: one Render per run.
	var b, run strings.Builder
	var style *lipgloss.Style
	flush := func() {
		if style != nil {
			b.WriteString(style.Render(run.String()))
		} else {
			b.WriteString(run.String())
		}
		run.Reset()
	}
	for x := left; x < min(len(cells), left+width); x++ {
		var next *lipgloss.Style
		switch {
		case x == look.cursor:
			next = &styleCursor
		case x >= look.selFrom && x < look.selTo:
			next = &styleSelection
		case x < len(cellHits) && cellHits[x]:
			next = &styleSearchHit
		case x < len(cellMarks) && cellMarks[x] != 0:
			next = &markStyles[cellMarks[x]]
		case x < len(cellClasses) && cellClasses[x] != syntax.Plain:
			next = &syntaxStyles[cellClasses[x]]
		}
		if next != style {
			flush()
			style = next
		}
		run.WriteString(cells[x])
	}
	flush()
	return b.String()
}

// statusLine: " EDIT │ first_word.c [+] │ C │ 6:5        message".
func (m Model) statusLine() string {
	if m.vim != nil && m.vim.Mode() == vim.Command {
		// The ex command line takes over the status bar, like Vim.
		return m.vim.Prompt() + m.vim.CommandLine() + styleCursor.Render(" ")
	}
	name := "[sem nome]"
	if p := m.ed.Path(); p != "" {
		name = filepath.Base(p)
	}
	if m.ed.Dirty() {
		name += " [+]"
	}
	if n := len(m.ses.bufs); n > 1 {
		name += fmt.Sprintf(" [%d/%d]", m.ses.cur+1, n)
	}
	cur := m.ed.Cursor()
	mode := styleMode.Render("EDIT")
	right := styleStatus.Render("ctrl+s salva · ctrl+z desfaz · ctrl+q sai ")
	if m.vim != nil {
		mode = styleModeNormal.Render(m.vim.Mode().String())
		right = styleStatus.Render("i insere · v seleciona · yy copia · p cola · :wq salva e sai ")
		if m.lsp != nil && m.lsp.srv.client != nil {
			right = styleStatus.Render("K info · gd definição · ]d próximo erro · :wq salva e sai ")
		}
		switch m.vim.Mode() {
		case vim.Insert:
			mode = styleMode.Render(m.vim.Mode().String())
			right = styleStatus.Render("esc volta ao normal ")
			if m.lsp != nil && m.lsp.srv.client != nil {
				right = styleStatus.Render("ctrl+n completa · esc volta ao normal ")
			}
		case vim.Visual, vim.VisualLine:
			mode = styleModeVisual.Render(m.vim.Mode().String())
			right = styleStatus.Render("y copia · d apaga · c muda · p cola · o troca a ponta · esc cancela ")
		}
	}
	if m.vim != nil && m.vim.Recording() != "" {
		mode += styleError.Render(" gravando @" + m.vim.Recording())
	}
	// The diagnostic under the cursor replaces the hints.
	if d := m.worst(cur.Line); d != nil && (m.vim == nil || m.vim.Mode() == vim.Normal) {
		msg, _, _ := strings.Cut(d.Message, "\n")
		right = severityStyles[d.Severity].Render(severityNames[d.Severity] + ": " + msg + " ")
	}
	if m.vim != nil {
		if p := m.vim.Pending(); p != "" {
			right = stylePending.Render(p + " ")
		}
	}
	lang := styleStatus.Render(" │ " + language(m.ed.Path()))
	if m.lsp != nil && m.lsp.srv.starting {
		lang += styleStatus.Render(" · " + m.lsp.srv.server.Name + "…")
	}
	if m.lsp != nil && m.lsp.srv.client != nil {
		lang += styleStatus.Render(" · " + m.lsp.srv.server.Name)
		if n := m.count(lsp.Error); n > 0 {
			lang += severityStyles[lsp.Error].Render(fmt.Sprintf(" ✖ %d", n))
		}
		if n := m.count(lsp.Warning); n > 0 {
			lang += severityStyles[lsp.Warning].Render(fmt.Sprintf(" ⚠ %d", n))
		}
	}
	left := mode + styleStatus.Render(" │ ") + styleFile.Render(name) + lang +
		styleStatus.Render(fmt.Sprintf(" │ %d:%d ", cur.Line+1, cur.Column+1))

	if m.message != "" {
		right = styleStatus.Render(m.message + " ")
		if m.isError {
			right = styleError.Render(m.message + " ")
		}
	}
	// Never drop the message on narrow terminals (it may be a warning):
	// cut it to the room left instead.
	room := m.width - lipgloss.Width(left)
	if room <= 1 {
		return ansi.Truncate(left, m.width, "…")
	}
	right = ansi.Truncate(right, room, "…")
	return left + strings.Repeat(" ", room-lipgloss.Width(right)) + right
}

// language names the file type shown in the status line.
func language(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".c", ".h":
		return "C"
	case ".cpp", ".hpp", ".cc":
		return "C++"
	case ".go":
		return "Go"
	case ".py":
		return "Python"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".md":
		return "Markdown"
	case ".sh":
		return "Shell"
	}
	return "Texto"
}
