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
	g       bool   // "g" typed, waiting for the second key ("gg")
}

// Controller interprets keys for one editor.
type Controller struct {
	ed      *editor.Editor
	mode    Mode
	state   CommandState
	cmdline string
	regs    map[rune]Register
}

// New controls ed, starting in Normal mode.
func New(ed *editor.Editor) *Controller {
	return &Controller{ed: ed, regs: map[rune]Register{}}
}

// Mode is the current mode.
func (c *Controller) Mode() Mode { return c.mode }

// Pending is the command typed so far in Normal mode (e.g. "d").
func (c *Controller) Pending() string { return strings.Join(c.state.Pending, "") }

// CommandLine is the ex command being typed after ":".
func (c *Controller) CommandLine() string { return c.cmdline }

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
		c.ed.Insert(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(text))
		return Result{}
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

	if m, ok := motions[key]; ok {
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

	// Anything else cancels a pending operator, like Vim.
	if st.Operator != opNone {
		c.state = CommandState{}
		return Result{}
	}
	n := times(st.Count)
	c.state = CommandState{}
	return c.command(key, n)
}

// prefix consumes counts (1-9 start one; 0 continues it, alone it is a
// motion) and the "g" prefix. It returns the key to interpret ("gg" after
// two "g"s), or ok=false when the key was consumed.
func (c *Controller) prefix(key string) (string, bool) {
	st := &c.state
	if !st.g && len(key) == 1 && key[0] >= '0' && key[0] <= '9' && (key != "0" || st.Count > 0) {
		st.Count = st.Count*10 + int(key[0]-'0')
		st.Pending = append(st.Pending, key)
		return "", false
	}
	switch {
	case st.g:
		st.g = false
		if key != "g" {
			c.state = CommandState{}
			return "", false
		}
		return "gg", true
	case key == "g":
		st.g = true
		st.Pending = append(st.Pending, key)
		return "", false
	}
	return key, true
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
	case ":":
		c.mode, c.cmdline = Command, ""
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
	if m, ok := motions[key]; ok {
		if t := m(c.ed, n, false); !t.Failed {
			c.moveTo(t)
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
		c.mode, c.cmdline = Command, ""
	}
	return Result{}
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
		cmd := strings.TrimSpace(c.cmdline)
		c.mode, c.cmdline = Normal, ""
		return c.execute(cmd)
	default:
		if isChar(key) {
			c.cmdline += key
		}
	}
	return Result{}
}

// execute runs an ex command: w, q, q!, wq, x.
func (c *Controller) execute(cmd string) Result {
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
