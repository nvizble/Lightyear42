package vim

import (
	"slices"
	"strconv"
	"strings"
)

// Repeating: "." replays the keys of the last command that changed the text
// (an insert session included), and macros replay what was typed between
// q{a-z} and q with @{a-z} (@@ runs the last one again).

// stroke is one input: a key, or text pasted in Insert mode.
type stroke struct {
	key   string
	paste bool
}

// maxReplay bounds nested replays (a macro calling itself).
const maxReplay = 20

// input runs one stroke, recording it for macros and for ".".
func (c *Controller) input(s stroke) Result {
	if c.macro != 0 && c.replaying == 0 {
		c.macroKeys = append(c.macroKeys, s)
	}
	if c.mode == Normal && len(c.state.Pending) == 0 {
		c.cmd, c.cmdChanges = nil, c.changes // a new command starts
	}
	c.cmd = append(c.cmd, s)

	var res Result
	if s.paste {
		c.ed.Insert(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(s.key))
	} else {
		res = c.handle(s.key)
	}

	if c.mode == Normal && len(c.state.Pending) == 0 && c.changes != c.cmdChanges && repeatable(c.cmd) {
		c.dot = slices.Clone(c.cmd)
	}
	return res
}

// repeatable leaves out commands that change the text but aren't edits to
// repeat: undo, redo, "." and macros.
func repeatable(cmd []stroke) bool {
	for _, s := range cmd {
		if !s.paste && len(s.key) == 1 && s.key[0] >= '0' && s.key[0] <= '9' {
			continue // the count
		}
		return s.paste || (s.key != "u" && s.key != "ctrl+r" && s.key != "." && !strings.HasPrefix(s.key, "@"))
	}
	return false
}

// repeatLast is ".": a count replaces the command's own.
func (c *Controller) repeatLast(count int) Result {
	strokes := c.dot
	if strokes == nil {
		return Result{}
	}
	if count > 0 {
		i := 0
		for i < len(strokes) && !strokes[i].paste && len(strokes[i].key) == 1 && strokes[i].key[0] >= '0' && strokes[i].key[0] <= '9' {
			i++
		}
		var digits []stroke
		for _, d := range strconv.Itoa(count) {
			digits = append(digits, stroke{key: string(d)})
		}
		strokes = append(digits, strokes[i:]...)
	}
	return c.replay(strokes)
}

// replay feeds strokes back, stopping at the first error.
func (c *Controller) replay(strokes []stroke) Result {
	if c.replaying >= maxReplay {
		return Result{Message: "repetição recursiva demais", Err: true}
	}
	c.replaying++
	cmd, changes := c.cmd, c.cmdChanges
	defer func() {
		c.replaying--
		c.cmd, c.cmdChanges = cmd, changes
	}()
	var res Result
	for _, s := range strokes {
		if res = c.input(s); res.Err || res.Quit {
			break
		}
	}
	return res
}

// macroKey handles "q{reg}" (start recording), "q" (stop) and "@{reg}",
// "@@" (play count times).
func (c *Controller) macroKey(key string, count int) Result {
	switch {
	case key == "q":
		c.macros[c.macro] = slices.Clone(c.macroKeys[:len(c.macroKeys)-1]) // without this q
		c.macro = 0
	case key[0] == 'q':
		if reg := rune(key[1]); reg >= 'a' && reg <= 'z' {
			c.macro, c.macroKeys = reg, nil
		}
	default: // "@x"
		reg := rune(key[1])
		if reg == '@' {
			reg = c.lastMacro
		}
		keys, ok := c.macros[reg]
		if !ok {
			return Result{Message: "nenhuma macro em @" + string(reg), Err: true}
		}
		c.lastMacro = reg
		var res Result
		for i := 0; i < count; i++ {
			if res = c.replay(keys); res.Err {
				break
			}
		}
		return res
	}
	return Result{}
}

// Recording is the register a macro is being recorded into ("" when none).
func (c *Controller) Recording() string {
	if c.macro == 0 {
		return ""
	}
	return string(c.macro)
}
