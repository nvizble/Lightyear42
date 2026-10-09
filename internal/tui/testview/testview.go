// Package testview is `lightyear test <project>`: the embedded editor with
// the project's files in a tree on the left (Space e), a button on top that
// runs the tests (Space r) and their results below (Space t). Clicking a
// failure opens the code it points at.
package testview

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/tester"
	"github.com/nvizble/Lightyear42/internal/tui/editorview"
)

var (
	colorAccent = lipgloss.AdaptiveColor{Light: "25", Dark: "39"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "243", Dark: "245"}
	colorGood   = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}
	colorFail   = lipgloss.AdaptiveColor{Light: "124", Dark: "196"}
	colorBar    = lipgloss.AdaptiveColor{Light: "254", Dark: "236"}

	styleBar     = lipgloss.NewStyle().Background(colorBar)
	styleTitle   = styleBar.Bold(true).Foreground(colorAccent)
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleButton  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(colorGood).Padding(0, 1)
	styleBusy    = styleButton.Background(lipgloss.AdaptiveColor{Light: "130", Dark: "214"})
	styleGood    = lipgloss.NewStyle().Foreground(colorGood)
	styleFail    = lipgloss.NewStyle().Foreground(colorFail)
	styleSel     = lipgloss.NewStyle().Reverse(true)
	styleCurrent = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleBorder  = lipgloss.NewStyle().Foreground(colorMuted)
)

// state is what the editor's host commands (Space e, r, t) change: they run
// inside the editor's Update, so they share it by pointer.
type state struct {
	toggleTree, togglePanel, run bool
}

// Model is the test screen.
type Model struct {
	ed      editorview.Model
	project tester.Project
	root    string
	opts    tester.Options
	cmd     *state

	width, height int
	tree          tree
	treeOpen      bool
	panel         panel
	panelOpen     bool
	// focus is where the keys go: the editor, the tree or the results.
	focus  focus
	leader bool // Space pressed in the tree or the results

	running  bool
	step     string
	started  time.Time
	progress chan string
	report   *tester.Report
	runErr   error
	took     time.Duration
	spin     int
}

type focus int

const (
	focusEditor focus = iota
	focusTree
	focusPanel
)

type (
	stepMsg string
	doneMsg struct {
		report tester.Report
		err    error
		took   time.Duration
	}
	tickMsg struct{}
)

// New builds the screen for project at root, with ed showing the first
// file; the tree starts open and focused, to pick a file.
func New(ed editorview.Model, project tester.Project, root string, opts tester.Options) Model {
	m := Model{project: project, root: root, opts: opts, cmd: &state{}, treeOpen: true, focus: focusTree}
	m.ed = ed.
		WithCommand(" e", func() { m.cmd.toggleTree = true }).
		WithCommand(" r", func() { m.cmd.run = true }).
		WithCommand(" t", func() { m.cmd.togglePanel = true })
	m.tree = newTree(root)
	m.tree.selectPath(ed.Editor().Path())
	return m
}

// Close stops the editor's language servers.
func (m Model) Close() { m.ed.Close() }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return m.ed.Init() }

// Layout: the bar on top, the tree and the editor side by side, the
// results at the bottom.
func (m Model) panelHeight() int {
	if !m.panelOpen {
		return 0
	}
	return max(m.height*2/5, 6)
}

func (m Model) treeWidth() int {
	if !m.treeOpen {
		return 0
	}
	return min(max(m.width/4, 20), 34)
}

func (m Model) editorX() int {
	if w := m.treeWidth(); w > 0 {
		return w + 1 // the border
	}
	return 0
}

func (m Model) bodyHeight() int { return max(m.height-1-m.panelHeight(), 3) }

// resize tells the editor its share of the screen.
func (m Model) resize() Model {
	next, _ := m.ed.Update(tea.WindowSizeMsg{Width: max(m.width-m.editorX(), 10), Height: m.bodyHeight()})
	m.ed = next.(editorview.Model)
	return m
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.resize(), nil
	case tea.KeyMsg:
		return m.key(msg)
	case tea.MouseMsg:
		return m.mouse(msg)
	case stepMsg:
		m.step = string(msg)
		return m, waitStep(m.progress)
	case tickMsg:
		if !m.running {
			return m, nil
		}
		m.spin++
		return m, tick()
	case doneMsg:
		m.running, m.step, m.took = false, "", msg.took
		m.report, m.runErr = &msg.report, msg.err
		m.panel = newPanel(m.report, m.runErr)
		m.tree.refresh()
		if !m.panelOpen {
			m.panelOpen = true
			m = m.resize()
		}
		return m, nil
	}
	return m.toEditor(msg)
}

// toEditor hands msg to the editor, then does what its host commands asked.
func (m Model) toEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.ed.Update(msg)
	m.ed = next.(editorview.Model)
	m, more := m.afterCommands()
	return m, tea.Batch(cmd, more)
}

func (m Model) afterCommands() (Model, tea.Cmd) {
	c := *m.cmd
	*m.cmd = state{}
	var cmd tea.Cmd
	if c.toggleTree {
		m = m.toggleTree()
	}
	if c.togglePanel {
		m = m.togglePanel()
	}
	if c.run {
		m, cmd = m.startRun()
	}
	return m, cmd
}

// toggleTree is Space e: open and go to the tree; there, close it.
func (m Model) toggleTree() Model {
	switch {
	case !m.treeOpen:
		m.treeOpen, m.focus = true, focusTree
		m.tree.refresh()
		m.tree.selectPath(m.ed.Editor().Path())
	case m.focus == focusTree:
		m.treeOpen, m.focus = false, focusEditor
	default:
		m.focus = focusTree
	}
	return m.resize()
}

// togglePanel is Space t, the same for the results.
func (m Model) togglePanel() Model {
	switch {
	case !m.panelOpen:
		m.panelOpen, m.focus = true, focusPanel
		if m.report == nil && !m.running {
			m.panel = newPanel(nil, nil)
		}
	case m.focus == focusPanel:
		m.panelOpen, m.focus = false, focusEditor
	default:
		m.focus = focusPanel
	}
	return m.resize()
}

// startRun saves the open files and runs the tests in the background.
func (m Model) startRun() (Model, tea.Cmd) {
	if m.running {
		return m, nil
	}
	if _, err := m.ed.SaveAll(); err != nil {
		m.ed = m.ed.WithMessage("não deu para salvar antes de rodar: "+err.Error(), true)
		return m, nil
	}
	m.running, m.started, m.step, m.spin = true, time.Now(), "preparando", 0
	m.progress = make(chan string, 16)
	opts, project, root, progress := m.opts, m.project, m.root, m.progress
	opts.Progress = func(s string) {
		select {
		case progress <- s:
		default:
		}
	}
	run := func() tea.Msg {
		start := time.Now()
		rep, err := project.Run(context.Background(), root, opts)
		close(progress)
		return doneMsg{report: rep, err: err, took: time.Since(start)}
	}
	return m, tea.Batch(run, waitStep(m.progress), tick())
}

func waitStep(ch chan string) tea.Cmd {
	return func() tea.Msg {
		s, ok := <-ch
		if !ok {
			return nil
		}
		return stepMsg(s)
	}
}

func tick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.focus == focusEditor {
		return m.toEditor(msg)
	}
	if m.leader {
		m.leader = false
		switch k {
		case "e":
			return m.toggleTree(), nil
		case "t":
			return m.togglePanel(), nil
		case "r":
			return m.startRun()
		}
		return m, nil
	}
	switch k {
	case " ":
		m.leader = true
		return m, nil
	case "esc":
		m.focus = focusEditor
		return m, nil
	case ":", "ctrl+q", "ctrl+c":
		// The editor quits (and asks about unsaved files).
		m.focus = focusEditor
		if k == "ctrl+c" {
			msg = tea.KeyMsg{Type: tea.KeyCtrlQ}
		}
		return m.toEditor(msg)
	}
	if m.focus == focusTree {
		return m.treeKey(k)
	}
	return m.panelKey(k)
}

func (m Model) treeKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "j", "down":
		m.tree.move(1)
	case "k", "up":
		m.tree.move(-1)
	case "g", "home":
		m.tree.move(-len(m.tree.rows))
	case "G", "end":
		m.tree.move(len(m.tree.rows))
	case "h", "left":
		m.tree.collapse()
	case "q":
		m.treeOpen, m.focus = false, focusEditor
		return m.resize(), nil
	case "enter", "l", "o", "right":
		return m.openRow(m.tree.sel)
	}
	return m, nil
}

// openRow opens the file at tree row i (or opens/closes the folder).
func (m Model) openRow(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(m.tree.rows) {
		return m, nil
	}
	m.tree.sel = i
	row := m.tree.rows[i]
	if row.dir {
		m.tree.toggle(row.path)
		return m, nil
	}
	return m.openFile(row.path, 0)
}

// openFile shows path in the editor (on line, when > 0) and focuses it.
func (m Model) openFile(path string, line int) (tea.Model, tea.Cmd) {
	ed, err := m.ed.OpenAt(path, line)
	if err != nil {
		m.ed = m.ed.WithMessage(err.Error(), true)
		return m, nil
	}
	m.ed, m.focus = ed, focusEditor
	m.tree.selectPath(path)
	// A new buffer may start a language server.
	next, cmd := m.ed.Update(nil)
	m.ed = next.(editorview.Model)
	return m, cmd
}

func (m Model) panelKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "j", "down":
		m.panel.move(1, m.panelHeight()-1)
	case "k", "up":
		m.panel.move(-1, m.panelHeight()-1)
	case "q":
		m.panelOpen, m.focus = false, focusEditor
		return m.resize(), nil
	case "r":
		return m.startRun()
	case "enter", "o":
		return m.activate(m.panel.sel)
	}
	return m, nil
}

// activate does what results line i offers: open or close a group, or open
// the code of a failure.
func (m Model) activate(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(m.panel.lines) {
		return m, nil
	}
	m.panel.sel = i
	l := m.panel.lines[i]
	if l.group != "" && l.c == nil {
		m.panel.toggle(l.group)
		return m, nil
	}
	if l.c != nil {
		if path, line := locate(m.root, *l.c, l.detail); path != "" {
			return m.openFile(path, line)
		}
	}
	return m, nil
}

func (m Model) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	press := msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft
	wheel := 0
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		wheel = -3
	case tea.MouseButtonWheelDown:
		wheel = 3
	}
	bodyEnd := 1 + m.bodyHeight()
	switch {
	case msg.Y == 0:
		if press && msg.X >= m.buttonX() && msg.X < m.buttonX()+lipgloss.Width(m.button()) {
			return m.startRun()
		}
		return m, nil
	case msg.Y >= bodyEnd:
		if wheel != 0 {
			m.panel.scroll(wheel, m.panelHeight()-1)
			return m, nil
		}
		if press {
			m.focus = focusPanel
			if row := msg.Y - bodyEnd - 1; row >= 0 {
				return m.activate(m.panel.top + row)
			}
		}
		return m, nil
	case m.treeOpen && msg.X < m.treeWidth():
		if wheel != 0 {
			m.tree.scroll(wheel, m.bodyHeight())
			return m, nil
		}
		if press {
			m.focus = focusTree
			return m.openRow(m.tree.top + msg.Y - 1)
		}
		return m, nil
	case msg.X < m.editorX():
		return m, nil // the border
	}
	if press {
		m.focus = focusEditor
	}
	msg.X -= m.editorX()
	msg.Y--
	return m.toEditor(msg)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	rows := []string{m.bar()}
	body := strings.Split(m.ed.View(), "\n")
	tw := m.treeWidth()
	var treeRows []string
	if tw > 0 {
		treeRows = m.tree.render(tw, m.bodyHeight(), m.focus == focusTree, m.ed.Editor().Path())
	}
	for i := range m.bodyHeight() {
		row := ""
		if tw > 0 {
			row = treeRows[i] + styleBorder.Render("│")
		}
		if i < len(body) {
			row += body[i]
		}
		rows = append(rows, row)
	}
	if h := m.panelHeight(); h > 0 {
		rows = append(rows, m.panel.render(m.width, h, m.focus == focusPanel, m.panelTitle())...)
	}
	return strings.Join(rows, "\n")
}

func (m Model) button() string {
	if m.running {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		return styleBusy.Render(fmt.Sprintf("%s %s… %ds", frames[m.spin%len(frames)], m.step,
			int(time.Since(m.started).Seconds())))
	}
	return styleButton.Render("▶ Rodar testes")
}

func (m Model) buttonX() int {
	return lipgloss.Width(m.barTitle()) + 1
}

func (m Model) barTitle() string {
	return styleTitle.Render(" lightyear test · " + m.project.Name + " ")
}

// bar is the top line: title, the Run button, the last result and the keys.
func (m Model) bar() string {
	left := m.barTitle() + styleBar.Render(" ") + m.button()
	if m.report != nil && !m.running {
		passed, failed, _ := m.report.Count()
		summary := styleGood.Inherit(styleBar).Render(fmt.Sprintf("  ✓ %d", passed))
		if failed > 0 {
			summary += styleFail.Inherit(styleBar).Render(fmt.Sprintf("  ✗ %d", failed))
		}
		left += summary
	}
	right := styleMuted.Inherit(styleBar).Render("Space e arquivos · Space r roda · Space t resultados ")
	room := m.width - lipgloss.Width(left)
	if room < lipgloss.Width(right) {
		right = ""
	}
	return ansi.Truncate(left+styleBar.Render(strings.Repeat(" ", max(room-lipgloss.Width(right), 0)))+right, m.width, "")
}

func (m Model) panelTitle() string {
	switch {
	case m.running:
		return "rodando: " + m.step
	case m.runErr != nil:
		return "erro"
	case m.report == nil:
		return "aperte ▶ Rodar testes (ou Space r)"
	}
	passed, failed, skipped := m.report.Count()
	s := fmt.Sprintf("%d passaram · %d falharam", passed, failed)
	if skipped > 0 {
		s += fmt.Sprintf(" · %d pulados", skipped)
	}
	return s + fmt.Sprintf(" · %.1fs", m.took.Seconds())
}
