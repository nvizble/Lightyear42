// Package vim turns key sequences into editor operations, Vim style: modes
// (Normal, Insert, Visual), counts, operators × motions, registers and an
// ex command line (:w, :q).
//
// It drives an editor.Editor and knows nothing about the TUI: keys arrive
// as generic names ("h", "esc", "ctrl+r", "enter"). See
// internal/editor/DESIGN.md for the roadmap.
package vim

import (
	"strings"
	"unicode/utf8"

	"github.com/nvizble/Lightyear42/internal/editor"
)

// Mode is the editing mode.
type Mode int

// Modes.
const (
	Normal Mode = iota
	Insert
	Command    // typing an ex command after ":"
	Visual     // selecting characters (v)
	VisualLine // selecting whole lines (V)
)

func (m Mode) String() string {
	switch m {
	case Insert:
		return "INSERT"
	case Command:
		return "COMMAND"
	case Visual:
		return "VISUAL"
	case VisualLine:
		return "V-LINE"
	}
	return "NORMAL"
}

// Result tells the host what happened with a key.
type Result struct {
	// Quit asks the host to close the editor.
	Quit bool
	// Message is a note for the status line; Err marks it as an error.
	Message string
	Err     bool
}

// CommandState holds a Normal-mode command still being typed, e.g. "3d2"
// waiting for a motion: count × operator × count × motion.
type CommandState struct {
	// Count is the number typed so far (0 when none): before the operator
	// while none is pending, before the motion after it.
	Count int
	// Operator is the pending operator (opNone when none).
	Operator Operator
	// Pending lists the keys typed so far, shown in the status line.
	Pending []string

	opKey   string // the operator's key, so "dd" can be told apart
	opCount int    // the count typed before the operator
	lead    string // a prefix ("g", "[", "]", "ctrl+w", "i", "a") waiting for its second key
	find    string // f, F, t, T, q or @ waiting for its character
}

// Controller interprets keys for one editor.
type Controller struct {
	ed      *editor.Editor
	mode    Mode
	state   CommandState
	cmdline string
	prompt  string // ":" for ex commands, "/" or "?" for a search
	regs    map[rune]Register
	search  search
	// lastFind is the last f/F/t/T and its character, for ; and ,.
	lastFind string

	// Repeating (repeat.go): the command being typed and the text changes
	// when it started, the last change for ".", macros and their recording.
	changes, cmdChanges int
	cmd, dot            []stroke
	replaying           int
	macro, lastMacro    rune
	macroKeys           []stroke
	macros              map[rune][]stroke

	// Commands are extra Normal-mode commands the host provides, by key
	// ("K", "gd", "]d": a language server's hover, definition...). They get
	// the count (1 when none); the controller only dispatches them.
	Commands map[string]func(count int) Result
	// ExCommands are ex commands the host provides or takes over, by name
	// (":e file", ":bn", ":q" with several buffers); they get what follows
	// the name.
	ExCommands map[string]func(arg string) Result

	watched map[*editor.Editor]bool // editors whose changes are counted
}

// New controls ed, starting in Normal mode.
func New(ed *editor.Editor) *Controller {
	c := &Controller{regs: map[rune]Register{}, macros: map[rune][]stroke{}, watched: map[*editor.Editor]bool{}}
	c.SetEditor(ed)
	return c
}

// SetEditor switches to another editor (another buffer), back in Normal
// mode; registers, macros, the search and "." carry over.
func (c *Controller) SetEditor(ed *editor.Editor) {
	if c.ed != nil {
		c.ed.ClearSelection()
		if c.mode == Insert {
			c.ed.EndGroup()
		}
	}
	c.ed, c.mode, c.state, c.cmdline = ed, Normal, CommandState{}, ""
	if !c.watched[ed] {
		c.watched[ed] = true
		ed.OnChange(func(editor.Change) { c.changes++ })
	}
}

// Mode is the current mode.
func (c *Controller) Mode() Mode { return c.mode }

// Pending is the command typed so far in Normal mode (e.g. "d").
func (c *Controller) Pending() string { return strings.Join(c.state.Pending, "") }

// CommandLine is what is being typed after the prompt.
func (c *Controller) CommandLine() string { return c.cmdline }

// Prompt is the command line's prompt: ":" (ex command), "/" or "?"
// (search).
func (c *Controller) Prompt() string {
	if c.prompt == "" {
		return ":"
	}
	return c.prompt
}

// MoveCursor moves the cursor (e.g. a mouse click) respecting the mode: it
// ends a Visual selection, and Normal mode never sits past the last
// character.
func (c *Controller) MoveCursor(p editor.Position) {
	if c.mode == Visual || c.mode == VisualLine {
		c.exitVisual()
	}
	c.ed.MoveCursor(p)
	if c.mode == Normal {
		c.clampNormal()
	}
}

// HandleKey processes one key: a single character ("h", "A", " ") or a
// named key ("esc", "enter", "ctrl+r").
func (c *Controller) HandleKey(key string) Result {
	return c.input(stroke{key: key})
}

func (c *Controller) handle(key string) Result {
	switch c.mode {
	case Insert:
		return c.insertKey(key)
	case Command:
		return c.commandKey(key)
	case Visual, VisualLine:
		return c.visualKey(key)
	}
	return c.normalKey(key)
}

// HandleText processes typed or pasted text: inserted as is in Insert mode,
// read as keys one character at a time otherwise.
func (c *Controller) HandleText(text string) Result {
	if c.mode == Insert {
		return c.input(stroke{key: text, paste: true})
	}
	var res Result
	for _, r := range text {
		if res = c.HandleKey(string(r)); res.Quit {
			break
		}
	}
	return res
}

func (c *Controller) normalKey(key string) Result {
	st := &c.state
	if key == "esc" {
		c.state = CommandState{}
		return Result{}
	}

	key, ok := c.prefix(key)
	if !ok {
		return Result{}
	}

	if keys, ok := aliases[key]; ok && st.Operator == opNone {
		var res Result
		for _, k := range keys {
			res = c.normalKey(k)
		}
		return res
	}

	if op, ok := operators[key]; ok {
		if st.Operator == opNone {
			st.Operator, st.opKey, st.opCount, st.Count = op, key, st.Count, 0
			st.Pending = append(st.Pending, key)
			return Result{}
		}
		// The operator twice (dd, 3dd, d3d) acts on count whole lines.
		same := key == st.opKey
		n := totalCount(st.opCount, st.Count)
		c.state = CommandState{}
		if same {
			line := c.ed.Cursor().Line
			last := min(line+times(n)-1, c.ed.Buffer().LineCount()-1)
			c.apply(op, Target{Pos: editor.Position{Line: last}, Linewise: true})
		}
		return Result{}
	}

	if m, ok := c.motion(key); ok {
		op, n := st.Operator, st.Count
		if op != opNone {
			n = totalCount(st.opCount, st.Count)
		}
		if op == opChange && key == "w" {
			m = changeWord
		}
		c.state = CommandState{}
		t := m(c.ed, n, op != opNone)
		switch {
		case t.Failed:
		case op != opNone:
			c.apply(op, t)
		default:
			c.moveTo(t)
		}
		return Result{}
	}

	if obj, inner, ok := lookupObject(key); ok {
		op := st.Operator
		c.state = CommandState{}
		r, linewise, found := obj(c.ed, inner)
		switch {
		case !found || op == opNone:
		case linewise:
			c.applyLines(op, r.Start.Line, r.End.Line)
		default:
			c.applyRange(op, r)
		}
		return Result{}
	}

	// Anything else cancels a pending operator, like Vim.
	if st.Operator != opNone {
		c.state = CommandState{}
		return Result{}
	}
	n, count := times(st.Count), st.Count
	c.state = CommandState{}
	switch {
	case key == ".":
		return c.repeatLast(count)
	case key == "q" || key[0] == '@' || (key[0] == 'q' && len(key) == 2):
		return c.macroKey(key, n)
	}
	if f, ok := c.Commands[key]; ok {
		return f(n)
	}
	return c.command(key, n)
}

// prefix consumes counts (1-9 start one; 0 continues it, alone it is a
// motion), the two-key prefixes ("g", "[", "]", and "i"/"a" for text
// objects after an operator or in Visual) and f/F/t/T with their
// character. It returns the key to interpret ("gg", "gd", "iw", "fa" once
// complete), or ok=false when the key was consumed.
func (c *Controller) prefix(key string) (string, bool) {
	st := &c.state
	if st.lead == "" && st.find == "" && len(key) == 1 && key[0] >= '0' && key[0] <= '9' && (key != "0" || st.Count > 0) {
		st.Count = st.Count*10 + int(key[0]-'0')
		st.Pending = append(st.Pending, key)
		return "", false
	}
	switch {
	case st.find != "":
		find := st.find
		st.find = ""
		if !isChar(key) {
			c.state = CommandState{}
			return "", false
		}
		return find + key, true
	case st.lead != "":
		key, st.lead = st.lead+key, ""
		_, motion := motions[key]
		_, command := c.Commands[key]
		_, _, object := lookupObject(key)
		switch {
		case motion || command || object:
			return key, true
		case c.commandPrefix(key):
			// Part of a longer host command ("gr" of "grn"): keep waiting.
			st.lead = key
			st.Pending = append(st.Pending, key[len(key)-1:])
			return "", false
		}
		c.state = CommandState{}
		return "", false
	case key == "q" && c.macro != 0:
		return key, true // stops recording
	case key == "f" || key == "F" || key == "t" || key == "T" || key == "q" || key == "@":
		st.find = key
		st.Pending = append(st.Pending, key)
		return "", false
	case key == "g" || key == "[" || key == "]" || key == "ctrl+w" || ((key == "i" || key == "a") && (st.Operator != opNone || c.visual())):
		st.lead = key
		st.Pending = append(st.Pending, key)
		return "", false
	}
	return key, true
}

// commandPrefix reports a host command longer than s that starts with it.
func (c *Controller) commandPrefix(s string) bool {
	for k := range c.Commands {
		if len(k) > len(s) && strings.HasPrefix(k, s) {
			return true
		}
	}
	return false
}

// OpenCommandLine starts typing an ex command with text already there (a
// host command like grn opening ":Rename ").
func (c *Controller) OpenCommandLine(text string) {
	if c.visual() {
		c.exitVisual()
	}
	c.mode, c.cmdline, c.prompt, c.state = Command, text, ":", CommandState{}
}

// motion finds the motion for key: the fixed ones, f/F/t/T with their
// character ("fa"), and ; and , repeating the last of those.
func (c *Controller) motion(key string) (Motion, bool) {
	if m, ok := motions[key]; ok {
		return m, true
	}
	switch {
	case (key == ";" || key == ",") && c.lastFind != "":
		kind, char := c.lastFind[0], []rune(c.lastFind[1:])[0]
		if key == "," {
			kind = map[byte]byte{'f': 'F', 'F': 'f', 't': 'T', 'T': 't'}[kind]
		}
		return findMotion(kind, char, true), true
	case len(key) > 1 && (key[0] == 'f' || key[0] == 'F' || key[0] == 't' || key[0] == 'T') && isChar(key[1:]):
		c.lastFind = key
		return findMotion(key[0], []rune(key[1:])[0], false), true
	}
	return nil, false
}

func (c *Controller) visual() bool {
	return c.mode == Visual || c.mode == VisualLine
}

// aliases are commands spelled as other keys, like Vim's D = d$.
var aliases = map[string][]string{"D": {"d", "$"}, "C": {"c", "$"}, "Y": {"y", "y"}}

// command runs the Normal-mode commands that aren't motions or operators.
func (c *Controller) command(key string, n int) Result {
	cur := c.ed.Cursor()
	switch key {
	case "i":
		c.enterInsert()
	case "a":
		if c.ed.Buffer().LineLen(cur.Line) > 0 {
			c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: cur.Column + 1})
		}
		c.enterInsert()
	case "I":
		c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: indentWidth(c.ed.Buffer(), cur.Line)})
		c.enterInsert()
	case "A":
		c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: c.ed.Buffer().LineLen(cur.Line)})
		c.enterInsert()
	case "o":
		indent := c.ed.Buffer().Indent(cur.Line)
		c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: c.ed.Buffer().LineLen(cur.Line)})
		c.enterInsert()
		c.ed.Insert("\n" + indent)
	case "O":
		indent := c.ed.Buffer().Indent(cur.Line)
		c.ed.MoveCursor(editor.Position{Line: cur.Line})
		c.enterInsert()
		c.ed.Insert(indent + "\n")
		c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: len([]rune(indent))})
	case "x", "delete":
		// x is "dl": count characters, never past the end of the line.
		if t := right(c.ed, n, true); !t.Failed {
			c.apply(opDelete, t)
		}
	case "u":
		for i := 0; i < n; i++ {
			if !c.ed.Undo() {
				if i == 0 {
					return Result{Message: "nada para desfazer"}
				}
				break
			}
		}
		c.clampNormal()
	case "ctrl+r":
		for i := 0; i < n; i++ {
			if !c.ed.Redo() {
				if i == 0 {
					return Result{Message: "nada para refazer"}
				}
				break
			}
		}
		c.clampNormal()
	case "p", "P":
		return c.put(key == "P", n)
	case ":", "/", "?":
		c.mode, c.cmdline, c.prompt = Command, "", key
	case "n", "N":
		return c.repeatSearch(c.search.backward != (key == "N"), n)
	case "*", "#":
		return c.searchWord(key == "#", n)
	case "v":
		c.enterVisual(Visual)
	case "V":
		c.enterVisual(VisualLine)
	}
	return Result{}
}

// moveTo moves the cursor to a motion's target (Normal mode).
func (c *Controller) moveTo(t Target) {
	if t.Vertical != 0 {
		step, n := editor.MoveDown, t.Vertical
		if n < 0 {
			step, n = editor.MoveUp, -n
		}
		for i := 0; i < n; i++ {
			c.ed.Move(step)
		}
	} else {
		c.ed.MoveCursor(t.Pos)
	}
	c.clampNormal()
}

// totalCount multiplies the counts before the operator and the motion
// ("2d3w" = 6); 0 when neither was typed.
func totalCount(opCount, count int) int {
	if opCount == 0 && count == 0 {
		return 0
	}
	return times(opCount) * times(count)
}

// Visual mode: motions extend the selection, which runs from where v/V was
// pressed to the cursor; operators (d, y, c) act on it and p replaces it.

func (c *Controller) enterVisual(mode Mode) {
	c.mode = mode
	if mode == VisualLine {
		c.ed.Select(editor.SelectLines)
	} else {
		c.ed.Select(editor.SelectChars)
	}
}

func (c *Controller) exitVisual() {
	c.ed.ClearSelection()
	c.mode = Normal
	c.state = CommandState{}
	c.clampNormal()
}

func (c *Controller) visualKey(key string) Result {
	if key == "esc" {
		c.exitVisual()
		return Result{}
	}
	key, ok := c.prefix(key)
	if !ok {
		return Result{}
	}
	n := c.state.Count
	c.state = CommandState{}
	if op, ok := operators[key]; ok {
		c.operate(op)
		return Result{}
	}
	if m, ok := c.motion(key); ok {
		if t := m(c.ed, n, false); !t.Failed {
			c.moveTo(t)
		}
		return Result{}
	}
	if obj, inner, ok := lookupObject(key); ok {
		if r, _, found := obj(c.ed, inner); found {
			c.selectRange(r)
		}
		return Result{}
	}
	switch key {
	case "v", "V":
		// The same key leaves Visual; the other one switches its kind.
		if mode := map[string]Mode{"v": Visual, "V": VisualLine}[key]; mode == c.mode {
			c.exitVisual()
		} else {
			c.enterVisual(mode)
		}
	case "o":
		c.ed.SwapSelectionEnds()
	case "x", "delete":
		c.operate(opDelete)
	case "p", "P":
		return c.replaceSelection(times(n))
	case ":":
		c.exitVisual()
		c.mode, c.cmdline, c.prompt = Command, "", ":"
	}
	return Result{}
}

// selectRange selects r (a text object) in charwise Visual mode.
func (c *Controller) selectRange(r editor.Range) {
	if !r.Start.Before(r.End) {
		return
	}
	end := editor.Position{Line: r.End.Line, Column: r.End.Column - 1}
	if r.End.Column == 0 {
		// Up to the line break before r.End.
		end = editor.Position{Line: r.End.Line - 1, Column: c.ed.Buffer().LineLen(r.End.Line - 1)}
	}
	c.mode = Visual
	c.ed.ClearSelection()
	c.ed.MoveCursor(r.Start)
	c.ed.Select(editor.SelectChars)
	c.ed.MoveCursor(end)
}

// operate runs op on the selection (whole lines in V-LINE), leaving
// Visual mode.
func (c *Controller) operate(op Operator) {
	r, linewise, ok := c.ed.SelectedRange()
	c.exitVisual()
	if !ok {
		return
	}
	if linewise {
		c.applyLines(op, r.Start.Line, r.End.Line)
		return
	}
	c.applyRange(op, r)
}

// ExtendSelection selects from the cursor to p, entering Visual mode from
// Normal (e.g. dragging the mouse).
func (c *Controller) ExtendSelection(p editor.Position) {
	switch c.mode {
	case Normal:
		c.enterVisual(Visual)
	case Visual, VisualLine:
	default:
		return
	}
	c.ed.MoveCursor(p)
	c.clampNormal()
}

func (c *Controller) enterInsert() {
	c.mode = Insert
	c.ed.BeginGroup()
}

func (c *Controller) insertKey(key string) Result {
	switch key {
	case "esc":
		c.ed.EndGroup()
		c.mode = Normal
		// Like Vim, leaving Insert steps back onto the last typed character.
		if cur := c.ed.Cursor(); cur.Column > 0 {
			c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: cur.Column - 1})
		}
	case "enter":
		c.ed.Insert("\n" + c.ed.Buffer().Indent(c.ed.Cursor().Line))
	case "tab":
		c.ed.Insert("\t")
	case "backspace":
		c.ed.DeleteBackward()
	case "delete":
		c.ed.DeleteForward()
	case "left":
		c.ed.Move(editor.MoveLeft)
	case "right":
		c.ed.Move(editor.MoveRight)
	case "up":
		c.ed.Move(editor.MoveUp)
	case "down":
		c.ed.Move(editor.MoveDown)
	default:
		if isChar(key) {
			c.ed.Insert(key)
		}
	}
	return Result{}
}

func (c *Controller) commandKey(key string) Result {
	switch key {
	case "esc":
		c.mode, c.cmdline = Normal, ""
	case "backspace":
		if c.cmdline == "" {
			c.mode = Normal
		} else {
			r := []rune(c.cmdline)
			c.cmdline = string(r[:len(r)-1])
		}
	case "enter":
		cmd, prompt := c.cmdline, c.prompt
		c.mode, c.cmdline = Normal, ""
		if prompt == "/" || prompt == "?" {
			return c.findPattern(cmd, prompt == "?", false, 1)
		}
		return c.execute(strings.TrimSpace(cmd))
	default:
		if isChar(key) {
			c.cmdline += key
		}
	}
	return Result{}
}

// execute runs an ex command: w, q, q!, wq, x, noh, or one of the host's.
func (c *Controller) execute(cmd string) Result {
	name, arg, _ := strings.Cut(cmd, " ")
	if f, ok := c.ExCommands[name]; ok {
		return f(strings.TrimSpace(arg))
	}
	switch cmd {
	case "":
		return Result{}
	case "w":
		return c.save(false)
	case "wq", "x":
		return c.save(true)
	case "q":
		if c.ed.Dirty() {
			return Result{Message: "E37: alterações não salvas (:wq salva e sai, :q! sai sem salvar)", Err: true}
		}
		return Result{Quit: true}
	case "q!":
		return Result{Quit: true}
	case "noh", "nohlsearch":
		c.search.highlight = false
		return Result{}
	}
	return Result{Message: "E492: não é um comando: " + cmd, Err: true}
}

func (c *Controller) save(quit bool) Result {
	if err := c.ed.Save(); err != nil {
		return Result{Message: err.Error(), Err: true}
	}
	return Result{Quit: quit, Message: "salvo: " + c.ed.Path()}
}

// lastColumn is the last column Normal mode may sit on: the last character
// (0 on an empty line).
func (c *Controller) lastColumn(line int) int {
	return max(c.ed.Buffer().LineLen(line)-1, 0)
}

// clampNormal pulls the cursor back onto the last character when it ends
// up past it, keeping the column j/k aim for.
func (c *Controller) clampNormal() {
	if cur := c.ed.Cursor(); cur.Column > c.lastColumn(cur.Line) {
		c.ed.PlaceCursor(editor.Position{Line: cur.Line, Column: c.lastColumn(cur.Line)})
	}
}

// isChar reports a key that types a character, as opposed to a named key
// like "esc" or "ctrl+r".
func isChar(key string) bool {
	return utf8.RuneCountInString(key) == 1
}
