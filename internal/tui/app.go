package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/services"
)

// appLoadTimeout bounds each tab load.
const appLoadTimeout = 30 * time.Second

// Exam defaults used by the app's "começar" action (same as `lightyear exam start`).
const (
	appExamRank     = "02"
	appExamDuration = 3 * time.Hour
)

// AppTab is a screen whose content is fetched and rendered on demand.
type AppTab struct {
	Title string
	// Load fetches the data and returns the rendered content. Nil means the
	// tab is unavailable; the app then shows AppOptions.Unavailable.
	Load func(ctx context.Context) (string, error)
	// LoadView, when set, is used instead of Load and also returns hotspots
	// (regions that react to the mouse).
	LoadView func(ctx context.Context) (AppView, error)
}

// Layout of the app around the tab content, used to map mouse positions:
// tab bar + rule on top, and the body padding.
const (
	appBodyTop = 2
	appPadTop  = 1
	appPadLeft = 2
)

var styleHover = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

// ExamControl is what the Exam tab drives. Implemented by *services.ExamService.
type ExamControl interface {
	Status() (exam.Session, error)
	Start(now time.Time, rank string, duration time.Duration) (exam.Session, error)
	Grade(ctx context.Context, now time.Time) (services.GradeReport, error)
	Finish() (exam.Session, error)
	Exercises() []exam.Exercise
	EditTargets(sess exam.Session) (subject string, files []string, err error)
}

// AppOptions configures the full-screen app.
type AppOptions struct {
	// Tabs are the API-backed screens, shown before the Exam tab.
	Tabs []AppTab
	// Unavailable explains why tabs without Load can't be used (e.g. not logged in).
	Unavailable string
	Exam        ExamControl
}

type appTabLoadedMsg struct {
	tab      int
	content  string
	hotspots []Hotspot
	err      error
}

type appGradedMsg struct {
	report services.GradeReport
	err    error
}

type appTickMsg time.Time

// appSpinMsg animates the spinner while a grading runs.
type appSpinMsg struct{}

// spinnerFrames is the braille spinner shown while grading.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func appSpin() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return appSpinMsg{} })
}

// appEditedMsg arrives when the editor opened with `e` exits.
type appEditedMsg struct{ err error }

// AppModel is the full-screen lightyear app: clickable tabs on top, a
// scrollable body and clickable actions in the footer.
type AppModel struct {
	opts AppOptions

	active   int
	scroll   int
	width    int
	height   int
	now      time.Time
	content  map[int]string
	hotspots map[int][]Hotspot
	hover    *Hotspot
	errs     map[int]error
	loading  map[int]bool

	examSess      *exam.Session
	examNotice    string
	grading       bool
	spin          int
	confirmFinish bool
}

// NewApp builds the app model.
func NewApp(opts AppOptions, now time.Time) AppModel {
	m := AppModel{
		opts:     opts,
		now:      now,
		content:  map[int]string{},
		hotspots: map[int][]Hotspot{},
		errs:     map[int]error{},
		loading:  map[int]bool{},
	}
	m.reloadExam()
	// Logged out: open on the Exam tab, the only one that works offline.
	if len(opts.Tabs) > 0 && !opts.Tabs[0].loadable() {
		m.active = m.examTab()
	}
	return m
}

// Init loads the first tab and starts the clock.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.load(m.active), appTick())
}

func appTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return appTickMsg(t) })
}

func (m AppModel) examTab() int { return len(m.opts.Tabs) }

func (m AppModel) titles() []string {
	titles := make([]string, 0, len(m.opts.Tabs)+1)
	for _, t := range m.opts.Tabs {
		titles = append(titles, t.Title)
	}
	return append(titles, "Exam")
}

// Update handles keys, mouse, loads and the clock.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampScroll()

	case appTickMsg:
		m.now = time.Time(msg)
		return m, appTick()

	case appTabLoadedMsg:
		m.loading[msg.tab] = false
		m.errs[msg.tab] = msg.err
		if msg.err == nil {
			m.content[msg.tab], m.hotspots[msg.tab] = msg.content, msg.hotspots
			if msg.tab == m.active {
				m.hover = nil
			}
		}
		m.clampScroll()

	case appSpinMsg:
		if !m.grading {
			return m, nil
		}
		m.spin++
		return m, appSpin()

	case appGradedMsg:
		m.grading = false
		if msg.err != nil {
			m.examNotice = styleFail.Render(msg.err.Error())
		} else {
			// The body draws the session card below, so the result stays apart.
			m.examNotice = RenderGradeResult(msg.report)
		}
		m.reloadExam()

	case appEditedMsg:
		if msg.err != nil {
			m.examNotice = styleFail.Render("Editor: " + msg.err.Error())
		} else {
			m.examNotice = styleLabel.Render("De volta do editor. Aperte g para corrigir.")
		}
		m.reloadExam()

	case tea.MouseMsg:
		return m.mouse(msg)

	case tea.KeyMsg:
		return m.key(msg.String())
	}
	return m, nil
}

func (m AppModel) key(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "right", "tab", "l":
		return m.selectTab((m.active + 1) % len(m.titles()))
	case "left", "shift+tab", "h":
		n := len(m.titles())
		return m.selectTab((m.active + n - 1) % n)
	case "down", "j":
		m.scrollBy(1)
	case "up", "k":
		m.scrollBy(-1)
	case "pgdown", " ":
		m.scrollBy(m.bodyHeight())
	case "pgup":
		m.scrollBy(-m.bodyHeight())
	case "home":
		m.scroll = 0
	case "r":
		if m.active == m.examTab() {
			m.reloadExam()
			return m, nil
		}
		return m, m.load(m.active)
	}
	if len(k) == 1 && k[0] >= '1' && int(k[0]-'1') < len(m.titles()) {
		return m.selectTab(int(k[0] - '1'))
	}
	if m.active == m.examTab() {
		return m.examKey(k)
	}
	return m, nil
}

// examKey handles the Exam tab actions: s starts, e opens the editor, g
// grades, f finishes (asking for confirmation first).
func (m AppModel) examKey(k string) (tea.Model, tea.Cmd) {
	if k != "f" {
		m.confirmFinish = false
	}
	switch k {
	case "s":
		if m.examSess != nil {
			m.examNotice = styleLabel.Render("Já existe uma prova em andamento.")
			return m, nil
		}
		if _, err := m.opts.Exam.Start(m.now, appExamRank, appExamDuration); err != nil {
			m.examNotice = styleFail.Render(err.Error())
		} else {
			m.examNotice = styleGood.Render("Prova começou. Boa sorte!")
		}
		m.reloadExam()
	case "e":
		if m.examSess == nil {
			return m, nil
		}
		subject, files, err := m.opts.Exam.EditTargets(*m.examSess)
		if err != nil {
			m.examNotice = styleFail.Render(err.Error())
			return m, nil
		}
		return m, tea.ExecProcess(editorCommand(subject, files), func(err error) tea.Msg {
			return appEditedMsg{err: err}
		})
	case "g":
		if m.examSess == nil || m.grading {
			return m, nil
		}
		m.grading, m.spin, m.examNotice = true, 0, ""
		ctrl, now := m.opts.Exam, m.now
		return m, tea.Batch(appSpin(), func() tea.Msg {
			report, err := ctrl.Grade(context.Background(), now)
			return appGradedMsg{report: report, err: err}
		})
	case "f":
		if m.examSess == nil {
			return m, nil
		}
		if !m.confirmFinish {
			m.confirmFinish = true
			m.examNotice = styleFail.Render("Encerrar a prova? Aperte f de novo para confirmar.")
			return m, nil
		}
		m.confirmFinish = false
		if sess, err := m.opts.Exam.Finish(); err != nil {
			m.examNotice = styleFail.Render(err.Error())
		} else {
			m.examNotice = fmt.Sprintf("Prova encerrada com %d/100.", sess.Score)
		}
		m.reloadExam()
	}
	return m, nil
}

func (m AppModel) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelDown:
		m.scrollBy(3)
		return m, nil
	case tea.MouseButtonWheelUp:
		m.scrollBy(-3)
		return m, nil
	}
	if msg.Action == tea.MouseActionMotion {
		m.hover = m.hotspotAt(msg.X, msg.Y)
		return m, nil
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	switch msg.Y {
	case 0:
		for i, s := range m.tabSpans() {
			if msg.X >= s.from && msg.X < s.to {
				return m.selectTab(i)
			}
		}
	case m.height - 1:
		for _, b := range m.buttons() {
			if msg.X >= b.from && msg.X < b.to {
				return m.key(b.key)
			}
		}
	default:
		// Clicking also selects, for terminals that don't report motion.
		m.hover = m.hotspotAt(msg.X, msg.Y)
	}
	return m, nil
}

// hotspotAt maps a screen position to a hotspot of the active tab.
func (m AppModel) hotspotAt(x, y int) *Hotspot {
	if y < appBodyTop || y >= appBodyTop+m.bodyHeight() {
		return nil
	}
	line, col := y-appBodyTop+m.scroll-appPadTop, x-appPadLeft
	spots := m.hotspots[m.active]
	for i := range spots {
		if h := &spots[i]; h.Line == line && col >= h.Col && col < h.Col+h.Width {
			return h
		}
	}
	return nil
}

func (m AppModel) selectTab(i int) (tea.Model, tea.Cmd) {
	m.active, m.scroll, m.confirmFinish, m.hover = i, 0, false, nil
	if i == m.examTab() {
		m.reloadExam()
		return m, nil
	}
	if _, ok := m.content[i]; ok || m.loading[i] {
		return m, nil
	}
	return m, m.load(i)
}

// load fetches a tab off the UI loop.
func (m AppModel) load(i int) tea.Cmd {
	if i >= len(m.opts.Tabs) || !m.opts.Tabs[i].loadable() || m.loading[i] {
		return nil
	}
	m.loading[i] = true
	tab := m.opts.Tabs[i]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), appLoadTimeout)
		defer cancel()
		if tab.LoadView != nil {
			view, err := tab.LoadView(ctx)
			return appTabLoadedMsg{tab: i, content: view.Content, hotspots: view.Hotspots, err: err}
		}
		content, err := tab.Load(ctx)
		return appTabLoadedMsg{tab: i, content: content, err: err}
	}
}

func (m *AppModel) reloadExam() {
	sess, err := m.opts.Exam.Status()
	switch {
	case err == nil:
		m.examSess = &sess
	case errors.Is(err, services.ErrNoExamSession):
		m.examSess = nil
	default:
		m.examSess = nil
		m.examNotice = styleFail.Render(err.Error())
	}
}

// View renders the tab bar, the body and the footer.
func (m AppModel) View() string {
	if m.width == 0 {
		return ""
	}
	rule := styleLabel.Render(strings.Repeat("─", m.width))
	lines := m.bodyLines()
	end := min(m.scroll+m.bodyHeight(), len(lines))
	body := lines[min(m.scroll, end):end]
	for len(body) < m.bodyHeight() {
		body = append(body, "")
	}
	if h := m.hover; h != nil {
		// Highlight the hovered region: keep the line, restyle its cells.
		if i := h.Line + appPadTop - m.scroll; i >= 0 && i < len(body) {
			col, line := h.Col+appPadLeft, body[i]
			body[i] = ansi.Cut(line, 0, col) + styleHover.Render(ansi.Strip(ansi.Cut(line, col, col+h.Width))) +
				ansi.Cut(line, col+h.Width, lipgloss.Width(line))
		}
	}
	for i, l := range body {
		body[i] = ansi.Truncate(l, m.width, "…")
	}
	return strings.Join([]string{m.tabBar(), rule, strings.Join(body, "\n"), rule, m.footer()}, "\n")
}

// Chrome: tab bar + rule on top, rule + footer at the bottom.
const appChromeLines = 4

func (m AppModel) bodyHeight() int { return max(m.height-appChromeLines, 1) }

func (m AppModel) bodyLines() []string {
	return strings.Split(m.body(), "\n")
}

func (m AppModel) body() string {
	pad := func(s string) string { return lipgloss.NewStyle().Padding(1, 2).Render(s) }
	if m.active == m.examTab() {
		return pad(m.examBody())
	}
	tab := m.opts.Tabs[m.active]
	if !tab.loadable() {
		return pad(styleLabel.Render(m.opts.Unavailable))
	}
	content, ok := m.content[m.active]
	switch {
	case m.loading[m.active] && !ok:
		return pad(styleLabel.Render("Carregando…"))
	case m.errs[m.active] != nil && !ok:
		return pad(styleFail.Render("Erro: "+m.errs[m.active].Error()) + "\n\n" + styleLabel.Render("Aperte r para tentar de novo."))
	}
	if m.errs[m.active] != nil {
		content = styleFail.Render("Falha ao atualizar: "+m.errs[m.active].Error()) + "\n\n" + content
	}
	return pad(content)
}

func (m AppModel) examBody() string {
	var parts []string
	switch {
	case m.grading:
		parts = append(parts, m.gradingLine())
	case m.examNotice != "":
		parts = append(parts, m.examNotice)
	}
	if m.examSess != nil {
		parts = append(parts, examCard(*m.examSess, m.now, false, ""),
			styleLabel.Render("Aperte e para abrir o subject e a sua entrega lado a lado no vim.\nNo vim, Ctrl-w w alterna entre os dois e :wq volta para cá; aí é só apertar g."))
	} else {
		parts = append(parts,
			styleTitle.Render("Simulador de provas")+"\n"+
				styleLabel.Render("Nenhuma prova em andamento. Aperte s para começar o Exam Rank 02 (3h),\nou rode `lightyear exam practice <exercício>` para treinar um só."),
			RenderExamCatalog(m.opts.Exam.Exercises()))
	}
	return strings.Join(parts, "\n\n")
}

// gradingLine is the spinner shown while grading:
// "⠋ Corrigindo…  cc -Wall -Wextra -Werror · 8 testes".
func (m AppModel) gradingLine() string {
	line := styleGood.Render(spinnerFrames[m.spin%len(spinnerFrames)]) + " " + styleValue.Render("Corrigindo…")
	info := "cc -Wall -Wextra -Werror"
	for _, ex := range m.opts.Exam.Exercises() {
		if m.examSess != nil && ex.Rank == m.examSess.Rank && ex.Name == m.examSess.Exercise {
			info += fmt.Sprintf(" · %d testes", len(ex.Tests))
		}
	}
	return line + styleLabel.Render("  "+info)
}

func (m *AppModel) scrollBy(n int) {
	m.scroll += n
	m.hover = nil
	m.clampScroll()
}

func (t AppTab) loadable() bool { return t.Load != nil || t.LoadView != nil }

func (m *AppModel) clampScroll() {
	maxScroll := max(len(m.bodyLines())-m.bodyHeight(), 0)
	m.scroll = min(max(m.scroll, 0), maxScroll)
}

var (
	styleAppLogo      = lipgloss.NewStyle().Bold(true).Foreground(colorGood)
	styleAppTab       = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)
	styleAppTabActive = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(colorGood).Padding(0, 1)
	styleAppButton    = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)
	styleAppKey       = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
)

const appLogo = " ✦ lightyear  "

type span struct{ from, to int }

// tabSpans returns the x range of each tab label in the tab bar, so clicks
// can be mapped back to tabs. Must match tabBar.
func (m AppModel) tabSpans() []span {
	x := lipgloss.Width(appLogo)
	spans := make([]span, 0, len(m.titles()))
	for i, t := range m.titles() {
		w := lipgloss.Width(styleAppTab.Render(tabLabel(i, t)))
		spans = append(spans, span{x, x + w})
		x += w + 1
	}
	return spans
}

func tabLabel(i int, title string) string { return fmt.Sprintf("%d %s", i+1, title) }

func (m AppModel) tabBar() string {
	var b strings.Builder
	b.WriteString(styleAppLogo.Render(appLogo))
	for i, t := range m.titles() {
		style := styleAppTab
		if i == m.active {
			style = styleAppTabActive
		}
		b.WriteString(style.Render(tabLabel(i, t)) + " ")
	}
	return b.String()
}

type appButton struct {
	key, label string
	span
}

// buttons lists the footer actions for the active tab, with their x ranges.
func (m AppModel) buttons() []appButton {
	var defs [][2]string
	if m.active == m.examTab() {
		if m.examSess == nil {
			defs = append(defs, [2]string{"s", "começar prova"})
		} else {
			defs = append(defs, [2]string{"e", "editar no vim"}, [2]string{"g", "grademe"}, [2]string{"f", "encerrar"})
		}
	}
	defs = append(defs, [2]string{"r", "atualizar"}, [2]string{"q", "sair"})

	x := 1
	buttons := make([]appButton, 0, len(defs))
	for _, d := range defs {
		w := lipgloss.Width(styleAppButton.Render(d[0] + " " + d[1]))
		buttons = append(buttons, appButton{key: d[0], label: d[1], span: span{x, x + w}})
		x += w + 1
	}
	return buttons
}

func (m AppModel) footer() string {
	var b strings.Builder
	b.WriteString(" ")
	for _, btn := range m.buttons() {
		b.WriteString(styleAppButton.Render(styleAppKey.Render(btn.key)+" "+btn.label) + " ")
	}
	hint := styleLabel.Render("1-" + fmt.Sprint(len(m.titles())) + " / ←→ abas · ↑↓ rolar ")
	if m.hover != nil {
		hint = styleHover.Render(m.hover.Info + " ")
	}
	gap := m.width - lipgloss.Width(b.String()) - lipgloss.Width(hint)
	if gap > 0 {
		b.WriteString(strings.Repeat(" ", gap) + hint)
	}
	return b.String()
}

// editorCommand opens the subject next to the files to turn in, using
// $VISUAL or $EDITOR (vim by default). Vim-family editors get a vertical
// split (-O): subject on the left (read-only), code on the right, with the
// cursor already in the code.
func editorCommand(subject string, files []string) *exec.Cmd {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	fields := strings.Fields(editor)
	if len(fields) == 0 {
		fields = []string{"vim"}
	}

	args := fields[1:]
	switch filepath.Base(fields[0]) {
	case "vim", "nvim", "vi", "mvim", "gvim":
		// -c runs in the first window (the subject): lock it, then move right.
		args = append(args, "-O", "-c", "setlocal nomodifiable readonly", "-c", "wincmd l")
	}
	args = append(args, subject)
	args = append(args, files...)
	return exec.Command(fields[0], args...)
}
