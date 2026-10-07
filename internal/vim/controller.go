// Package vim turns key sequences into editor operations, Vim style: modes,
// pending commands and an ex command line (:w, :q).
//
// It drives an editor.Editor and knows nothing about the TUI: keys arrive
// as generic names ("h", "esc", "ctrl+r", "enter"). See
// internal/editor/DESIGN.md for the roadmap (counts, operators × motions,
// visual mode and registers come in the next phases).
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
	Command // typing an ex command after ":"
)

func (m Mode) String() string {
	switch m {
	case Insert:
		return "INSERT"
	case Command:
		return "COMMAND"
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

// CommandState holds a command still being typed (e.g. "d" waiting for
// "d"). Counts and motions join it in the command-parser phase.
type CommandState struct {
	Pending []string
}

// Controller interprets keys for one editor.
type Controller struct {
	ed      *editor.Editor
	mode    Mode
	state   CommandState
	cmdline string
}

// New controls ed, starting in Normal mode.
func New(ed *editor.Editor) *Controller {
	return &Controller{ed: ed}
}

// Mode is the current mode.
func (c *Controller) Mode() Mode { return c.mode }

// Pending is the command typed so far in Normal mode (e.g. "d").
func (c *Controller) Pending() string { return strings.Join(c.state.Pending, "") }

// CommandLine is the ex command being typed after ":".
func (c *Controller) CommandLine() string { return c.cmdline }

// MoveCursor moves the cursor (e.g. a mouse click) respecting the mode:
// Normal mode never sits past the last character.
func (c *Controller) MoveCursor(p editor.Position) {
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
	if len(c.state.Pending) > 0 {
		return c.pendingKey(key)
	}
	cur := c.ed.Cursor()
	switch key {
	case "h", "left":
		if cur.Column > 0 {
			c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: cur.Column - 1})
		}
	case "l", "right":
		if cur.Column < c.lastColumn(cur.Line) {
			c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: cur.Column + 1})
		}
	case "j", "down":
		c.ed.Move(editor.MoveDown)
		c.clampNormal()
	case "k", "up":
		c.ed.Move(editor.MoveUp)
		c.clampNormal()
	case "i":
		c.enterInsert()
	case "a":
		if c.ed.Buffer().LineLen(cur.Line) > 0 {
			c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: cur.Column + 1})
		}
		c.enterInsert()
	case "I":
		c.ed.MoveCursor(editor.Position{Line: cur.Line, Column: len([]rune(c.ed.Buffer().Indent(cur.Line)))})
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
		if c.ed.Buffer().LineLen(cur.Line) > 0 {
			c.ed.Delete(editor.Range{Start: cur, End: editor.Position{Line: cur.Line, Column: cur.Column + 1}})
			c.clampNormal()
		}
	case "d":
		c.state.Pending = append(c.state.Pending, key)
	case "u":
		if !c.ed.Undo() {
			return Result{Message: "nada para desfazer"}
		}
		c.clampNormal()
	case "ctrl+r":
		if !c.ed.Redo() {
			return Result{Message: "nada para refazer"}
		}
		c.clampNormal()
	case ":":
		c.mode, c.cmdline = Command, ""
	}
	return Result{}
}

// pendingKey completes a command waiting for more keys; anything that
// doesn't complete it cancels it, like Vim.
func (c *Controller) pendingKey(key string) Result {
	pending := c.Pending()
	c.state = CommandState{}
	if pending == "d" && key == "d" {
		c.deleteLine()
	}
	return Result{}
}

// deleteLine removes the cursor's line (dd) and lands on the first
// non-blank character of the line that takes its place.
func (c *Controller) deleteLine() {
	buf, line := c.ed.Buffer(), c.ed.Cursor().Line
	r := editor.Range{Start: editor.Position{Line: line}, End: editor.Position{Line: line + 1}}
	switch {
	case line+1 < buf.LineCount():
		// Remove the line and its newline.
	case line > 0:
		// Last line: take the newline before it instead.
		r = editor.Range{Start: editor.Position{Line: line - 1, Column: buf.LineLen(line - 1)}, End: editor.Position{Line: line, Column: buf.LineLen(line)}}
	default:
		// The only line: empty it.
		r.End = editor.Position{Line: 0, Column: buf.LineLen(0)}
	}
	c.ed.Delete(r)
	target := min(line, buf.LineCount()-1)
	c.ed.MoveCursor(editor.Position{Line: target, Column: len([]rune(buf.Indent(target)))})
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
