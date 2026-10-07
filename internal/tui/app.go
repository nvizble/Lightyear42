package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/nvizble/Lightyear42/internal/tui/editorview"
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
	// Activate, when set, runs when a hotspot is clicked or picked in the
	// search (e.g. open a subject); the returned text goes to the footer.
	Activate func(ctx context.Context, h Hotspot) (string, error)
}

// appActivateTimeout bounds a hotspot action (e.g. downloading a PDF).
const appActivateTimeout = 3 * time.Minute

// appActivatedMsg carries the outcome of a tab's Activate.
type appActivatedMsg struct {
	status string
	err    error
}

// Layout of the app around the tab content, used to map mouse positions:
// tab bar + rule on top, and the body padding.
const (
	appBodyTop = 2
	appPadTop  = 1
	appPadLeft = 2
)

var (
	styleHover  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	styleSearch = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "162", Dark: "213"})
)

// ExamControl is what the Exam tab drives. Implemented by *services.ExamService.
type ExamControl interface {
	Status() (exam.Session, error)
	Start(now time.Time, rank string, duration time.Duration) (exam.Session, error)
	Grade(ctx context.Context, now time.Time) (services.GradeReport, error)
	Finish() (exam.Session, error)
	Practice(now time.Time, name string) (exam.Session, error)
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
	// EditorLSP turns on the language server (clangd: errors and
	// completion) in the Exam tab's editor.
	EditorLSP bool
	// EditorScheme is the editor's saved color scheme, and SaveEditorScheme
	// keeps the one :colorscheme picks (both optional).
	EditorScheme     func() string
	SaveEditorScheme func(name string)
	// Notify, when set, runs every NotifyEvery while the app is open (like
	// `lightyear notify watch`): it pushes new evaluations to the phone and
	// says what it sent, for the footer ("" when nothing).
	Notify      func(ctx context.Context) (string, error)
	NotifyEvery time.Duration
}

// appNotifiedMsg carries the outcome of a Notify check.
type appNotifiedMsg struct {
	status string
	err    error
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
	// "/" search over the active tab's hotspots: searching while typing,
	// query kept after Enter to keep the matches highlighted.
	searching bool
	query     string
	// exactID is set when a suggestion was picked: match that hotspot only.
	exactID string
	// status is the footer message of the last hotspot action.
	status     string
	activating bool
	// sel is the highlighted suggestion while searching.
	sel     int
	errs    map[int]error
	loading map[int]bool

	// editor is the Exam tab's editor while it is open (over the app);
	// editorShares keeps its window sizes for the next time it opens.
	editor        *editorview.Model
	editorShares  []float64
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

// Init loads the first tab, starts the clock and the notify checks.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.load(m.active), appTick(), m.notifyAfter(0))
}

// notifyAfter runs the Notify check after wait (none without Notify).
func (m AppModel) notifyAfter(wait time.Duration) tea.Cmd {
	check := m.opts.Notify
	if check == nil {
		return nil
	}
	return tea.Tick(wait, func(time.Time) tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), appLoadTimeout)
		defer cancel()
		status, err := check(ctx)
		return appNotifiedMsg{status: status, err: err}
	})
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
	if m.editor == nil {
		return m.update(msg)
	}
	// The Exam tab's editor is open: it takes the keys and the mouse; the
	// rest (the clock, loads, its language server) goes to both.
	var appCmd tea.Cmd
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
	default:
		next, cmd := m.update(msg)
		m, appCmd = next.(AppModel), cmd
	}
	next, edCmd := m.editor.Update(msg)
	ed := next.(editorview.Model)
	if ed.Done() {
		// Its :q closes the editor, not the app.
		m.editorShares = ed.Shares()
		ed.Close()
		m.editor = nil
		m.examNotice = styleLabel.Render("De volta do editor. Aperte g para corrigir.")
		m.reloadExam()
		return m, appCmd
	}
	m.editor = &ed
	return m, tea.Batch(appCmd, edCmd)
}

func (m AppModel) update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case appNotifiedMsg:
		switch {
		case msg.err != nil:
			m.status = styleFail.Render("notify: " + msg.err.Error())
		case msg.status != "":
			m.status = msg.status
		}
		return m, m.notifyAfter(m.opts.NotifyEvery)

	case appActivatedMsg:
		m.activating = false
		m.status = msg.status
		if msg.err != nil {
			m.status = styleFail.Render(msg.err.Error())
		}

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
	if m.searching {
		return m.searchKey(k)
	}
	switch k {
	case "/":
		if len(m.spots()) > 0 {
			m.searching, m.query, m.exactID, m.sel, m.hover = true, "", "", 0, nil
		}
		return m, nil
	case "esc":
		m.query, m.exactID = "", ""
		return m, nil
	case "enter":
		// After a search pick, Enter runs the tab's action on the result.
		if found := m.matches(); len(found) == 1 && m.exactID != "" {
			return m.activate(found[0])
		}
		return m, nil
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

// examKey handles the Exam tab actions: s starts, e opens the editor (E
// the external $EDITOR), g grades, f finishes (asking for confirmation
// first).
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
	case "e", "E":
		if m.examSess == nil {
			return m, nil
		}
		subject, files, err := m.opts.Exam.EditTargets(*m.examSess)
		if err != nil {
			m.examNotice = styleFail.Render(err.Error())
			return m, nil
		}
		if k == "E" {
			return m, tea.ExecProcess(editorCommand(subject, files), func(err error) tea.Msg {
				return appEditedMsg{err: err}
			})
		}
		return m.openEditor(subject, files)
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
	// Suggestions while searching: hovering highlights, clicking picks.
	if box, first := m.suggestionBox(); len(box) > 0 && msg.X >= appPadLeft && msg.X < appPadLeft+lipgloss.Width(box[0]) {
		if i := msg.Y - first; i >= 0 && i < len(box)-2 {
			if msg.Action == tea.MouseActionMotion {
				m.sel = i
				return m, nil
			}
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				return m.pickSuggestion(m.suggestions()[i])
			}
		}
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
		// Clicking runs the tab's action when it has one; otherwise it
		// selects, for terminals that don't report motion.
		h := m.hotspotAt(msg.X, msg.Y)
		if h != nil && m.active == m.examTab() {
			return m.practice(h.id())
		}
		if h != nil && m.opts.tabAt(m.active).Activate != nil {
			return m.activate(h)
		}
		m.hover = h
	}
	return m, nil
}

// searchKey edits the search query: typing filters live and suggests
// people, ↑↓ choose a suggestion, Enter/Tab pick it, Esc clears the search.
func (m AppModel) searchKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.searching, m.query, m.exactID = false, "", ""
		return m, nil
	case "up", "ctrl+p":
		if n := len(m.suggestions()); n > 0 {
			m.sel = (m.sel + n - 1) % n
		}
		return m, nil
	case "down", "ctrl+n":
		if n := len(m.suggestions()); n > 0 {
			m.sel = (m.sel + 1) % n
		}
		return m, nil
	case "enter", "tab":
		if sugs := m.suggestions(); len(sugs) > 0 {
			return m.pickSuggestion(sugs[min(m.sel, len(sugs)-1)])
		}
		m.searching = false
		return m, nil
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
	case "space":
		m.query += " "
	default:
		for _, r := range k {
			if !unicode.IsPrint(r) {
				return m, nil
			}
		}
		m.query += k
	}
	m.exactID, m.sel = "", 0
	if found := m.matches(); len(found) > 0 {
		m.scrollTo(found[0].Line)
	}
	return m, nil
}

// maxSuggestions caps the suggestion box while searching.
const maxSuggestions = 6

// suggestions lists the people matching the query while typing, one per
// login: those whose login starts with it first, then alphabetically.
func (m AppModel) suggestions() []*Hotspot {
	if !m.searching {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(m.query))
	if q == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []*Hotspot
	spots := m.spots()
	for i := range spots {
		id := spots[i].id()
		if spots[i].Search == "" || seen[id] || !strings.Contains(strings.ToLower(spots[i].Search), q) {
			continue
		}
		seen[id] = true
		out = append(out, &spots[i])
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].label()), strings.ToLower(out[j].label())
		if pa, pb := strings.HasPrefix(a, q), strings.HasPrefix(b, q); pa != pb {
			return pa
		}
		return a < b
	})
	if len(out) > maxSuggestions {
		out = out[:maxSuggestions]
	}
	return out
}

// pickSuggestion searches for exactly that hotspot, closes the box and, on
// tabs with an action, runs it.
func (m AppModel) pickSuggestion(h *Hotspot) (AppModel, tea.Cmd) {
	m.query, m.exactID, m.searching = h.label(), h.id(), false
	m.scrollTo(h.Line)
	if m.active == m.examTab() {
		return m.practice(h.id())
	}
	if m.opts.tabAt(m.active).Activate != nil {
		return m.activate(h)
	}
	return m, nil
}

// activate runs the active tab's action on a hotspot, off the UI loop.
func (m AppModel) activate(h *Hotspot) (AppModel, tea.Cmd) {
	act := m.opts.tabAt(m.active).Activate
	if act == nil || m.activating {
		return m, nil
	}
	m.activating, m.status = true, "Abrindo "+h.label()+"…"
	spot := *h
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), appActivateTimeout)
		defer cancel()
		status, err := act(ctx, spot)
		return appActivatedMsg{status: status, err: err}
	}
}

// practice starts practicing one exercise, picked in the Exam tab's
// catalog: untimed, like `lightyear exam practice`.
func (m AppModel) practice(name string) (AppModel, tea.Cmd) {
	if _, err := m.opts.Exam.Practice(m.now, name); err != nil {
		m.examNotice = styleFail.Render(err.Error())
	} else {
		m.examNotice = styleGood.Render("Treino de " + name + " começou, sem tempo. Aperte e para abrir no editor.")
	}
	m.hover, m.query, m.exactID, m.scroll = nil, "", "", 0
	m.reloadExam()
	return m, nil
}

// spots are the active tab's hotspots; the Exam tab's follow what it shows.
func (m AppModel) spots() []Hotspot {
	if m.active == m.examTab() {
		return m.examView().Hotspots
	}
	return m.hotspots[m.active]
}

// tabAt returns the API tab i, or an empty tab (e.g. for the Exam tab).
func (o AppOptions) tabAt(i int) AppTab {
	if i >= 0 && i < len(o.Tabs) {
		return o.Tabs[i]
	}
	return AppTab{}
}

// suggestionBox renders the suggestion list and the screen row of its
// first entry (the box sits at the bottom of the body, over the content).
func (m AppModel) suggestionBox() (lines []string, firstRow int) {
	sugs := m.suggestions()
	if len(sugs) == 0 {
		return nil, 0
	}
	width := 0
	for _, h := range sugs {
		width = max(width, lipgloss.Width(h.label()))
	}
	rows := make([]string, 0, len(sugs))
	for i, h := range sugs {
		rest := strings.Replace(h.Info, " · "+h.label(), "", 1)
		login := h.label() + strings.Repeat(" ", width-lipgloss.Width(h.label()))
		if i == min(m.sel, len(sugs)-1) {
			rows = append(rows, styleSearch.Render("▸ "+login)+"  "+styleValue.Render(rest))
		} else {
			rows = append(rows, "  "+styleValue.Render(login)+"  "+styleLabel.Render(rest))
		}
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorMuted).Padding(0, 1).
		Render(strings.Join(rows, "\n"))
	lines = strings.Split(box, "\n")
	return lines, appBodyTop + m.bodyHeight() - len(lines) + 1
}

// matches lists the active tab's hotspots matching the search query
// (case-insensitive substring of Search, or Info when Search is empty).
func (m AppModel) matches() []*Hotspot {
	q := strings.ToLower(strings.TrimSpace(m.query))
	if q == "" {
		return nil
	}
	if m.exactID != "" {
		var found []*Hotspot
		spots := m.spots()
		for i := range spots {
			if spots[i].id() == m.exactID {
				found = append(found, &spots[i])
			}
		}
		return found
	}
	var found []*Hotspot
	spots := m.spots()
	for i := range spots {
		text := spots[i].Search
		if text == "" {
			text = spots[i].Info
		}
		if strings.Contains(strings.ToLower(text), q) {
			found = append(found, &spots[i])
		}
	}
	return found
}

// scrollTo brings a content line into view, centered when it was off screen.
func (m *AppModel) scrollTo(line int) {
	if line+appPadTop >= m.scroll && line+appPadTop < m.scroll+m.bodyHeight() {
		return
	}
	m.scroll = line + appPadTop - m.bodyHeight()/2
	m.clampScroll()
}

// hotspotAt maps a screen position to a hotspot of the active tab.
func (m AppModel) hotspotAt(x, y int) *Hotspot {
	if y < appBodyTop || y >= appBodyTop+m.bodyHeight() {
		return nil
	}
	line, col := y-appBodyTop+m.scroll-appPadTop, x-appPadLeft
	spots := m.spots()
	for i := range spots {
		if h := &spots[i]; h.Line == line && col >= h.Col && col < h.Col+h.Width {
			return h
		}
	}
	return nil
}

func (m AppModel) selectTab(i int) (tea.Model, tea.Cmd) {
	m.active, m.scroll, m.confirmFinish, m.hover = i, 0, false, nil
	m.searching, m.query, m.exactID, m.status = false, "", "", ""
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
	if m.editor != nil {
		return m.editor.View()
	}
	rule := styleLabel.Render(strings.Repeat("─", m.width))
	lines := m.bodyLines()
	end := min(m.scroll+m.bodyHeight(), len(lines))
	body := lines[min(m.scroll, end):end]
	for len(body) < m.bodyHeight() {
		body = append(body, "")
	}
	// Highlight search matches, then the hovered region on top: keep the
	// line, restyle the cells.
	highlight := func(h *Hotspot, style lipgloss.Style) {
		if i := h.Line + appPadTop - m.scroll; i >= 0 && i < len(body) {
			col, line := h.Col+appPadLeft, body[i]
			body[i] = ansi.Cut(line, 0, col) + style.Render(ansi.Strip(ansi.Cut(line, col, col+h.Width))) +
				ansi.Cut(line, col+h.Width, lipgloss.Width(line))
		}
	}
	for _, h := range m.matches() {
		highlight(h, styleSearch)
	}
	if m.hover != nil {
		highlight(m.hover, styleHover)
	}
	// Suggestions float over the bottom of the body while typing.
	if box, _ := m.suggestionBox(); len(box) > 0 {
		start := len(body) - len(box)
		for i, row := range box {
			if j := start + i; j >= 0 {
				line := body[j]
				body[j] = ansi.Cut(line, 0, appPadLeft) + row + ansi.Cut(line, appPadLeft+lipgloss.Width(row), lipgloss.Width(line))
			}
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
	// Indent only (no right padding): a single long line must not widen,
	// and so truncate, every other line of the tab.
	pad := func(s string) string {
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			lines[i] = strings.Repeat(" ", appPadLeft) + l
		}
		return strings.Repeat("\n", appPadTop) + strings.Join(lines, "\n")
	}
	if m.active == m.examTab() {
		return pad(m.examView().Content)
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

// examView is the Exam tab: the session in progress, or the catalog, whose
// exercises are clicked to practice them.
func (m AppModel) examView() AppView {
	var parts []string
	switch {
	case m.grading:
		parts = append(parts, m.gradingLine())
	case m.examNotice != "":
		parts = append(parts, m.examNotice)
	}
	if m.examSess != nil {
		parts = append(parts, examCard(*m.examSess, m.now, false, ""),
			styleLabel.Render("Aperte e para abrir o subject (só leitura) e a sua entrega lado a lado no editor,\ncom os erros do compilador e autocomplete. Ctrl-w w alterna entre os dois, arrastar\na borda (ou Ctrl-w > <) muda o tamanho e :wq salva e volta para cá; aí é só apertar g.\nE abre no seu $EDITOR (vim por padrão)."))
		return AppView{Content: strings.Join(parts, "\n\n")}
	}
	parts = append(parts, styleTitle.Render("Simulador de provas")+"\n"+
		styleLabel.Render("Nenhuma prova em andamento. Aperte s para começar o Exam Rank 02 (3h),\nou clique num exercício para treinar só ele, sem tempo (/ busca)."))
	head := strings.Join(parts, "\n\n") + "\n\n"
	catalog := ExamCatalogView(m.opts.Exam.Exercises(), m.width-appPadLeft)
	for i := range catalog.Hotspots {
		catalog.Hotspots[i].Line += strings.Count(head, "\n")
	}
	return AppView{Content: head + catalog.Content, Hotspots: catalog.Hotspots}
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
	if m.opts.Notify != nil {
		// The schedule is being watched for the phone (notify).
		on := styleLabel.Render("notify ligado ")
		if gap := m.width - lipgloss.Width(b.String()) - lipgloss.Width(on); gap > 0 {
			b.WriteString(strings.Repeat(" ", gap) + on)
		}
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
			defs = append(defs, [2]string{"e", "editar"}, [2]string{"E", "$EDITOR"}, [2]string{"g", "grademe"}, [2]string{"f", "encerrar"})
		}
	}
	if len(m.spots()) > 0 {
		defs = append(defs, [2]string{"/", "buscar"})
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
	if m.searching {
		b.WriteString(styleAppKey.Render(" buscar: ") + styleSearch.Render(m.query) + styleValue.Render("▌") +
			styleLabel.Render("   ↑↓ escolhe · enter seleciona · esc cancela "))
	} else {
		for _, btn := range m.buttons() {
			b.WriteString(styleAppButton.Render(styleAppKey.Render(btn.key)+" "+btn.label) + " ")
		}
	}
	hint := styleLabel.Render("1-" + fmt.Sprint(len(m.titles())) + " / ←→ abas · ↑↓ rolar ")
	switch found := m.matches(); {
	case m.hover != nil:
		hint = styleHover.Render(m.hover.Info + " ")
	case m.status != "" && !m.searching:
		hint = styleValue.Render(m.status + " ")
	case strings.TrimSpace(m.query) == "":
	case len(found) == 0:
		hint = styleFail.Render(fmt.Sprintf("ninguém online com %q ", strings.TrimSpace(m.query)))
	case m.searching:
		// While typing, the suggestion box has the details.
		hint = styleSearch.Render(fmt.Sprintf("%d encontrados ", len(found)))
	case len(found) == 1:
		hint = styleSearch.Render(found[0].Info + " ")
	default:
		hint = styleSearch.Render(fmt.Sprintf("%d encontrados · %s ", len(found), found[0].Info))
	}
	gap := m.width - lipgloss.Width(b.String()) - lipgloss.Width(hint)
	if gap > 0 {
		b.WriteString(strings.Repeat(" ", gap) + hint)
	}
	return b.String()
}

// openEditor opens lightyear's editor over the app: the subject read-only
// on the left, the files to turn in on the right.
func (m AppModel) openEditor(subject string, files []string) (tea.Model, tea.Cmd) {
	ed, err := editorview.NewSideBySide(subject, files)
	if err != nil {
		m.examNotice = styleFail.Render("Editor: " + err.Error())
		return m, nil
	}
	if m.opts.EditorLSP {
		ed = ed.WithLSP()
	}
	ed = ed.WithShares(m.editorShares) // the subject's width as it was left
	if m.opts.EditorScheme != nil {
		ed = ed.WithColorscheme(m.opts.EditorScheme())
	}
	ed = ed.OnColorscheme(m.opts.SaveEditorScheme)
	start := ed.Init() // before any Update, like Bubble Tea does
	next, _ := ed.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	ed = next.(editorview.Model)
	m.editor = &ed
	return m, start
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
