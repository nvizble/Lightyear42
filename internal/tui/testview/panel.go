package testview

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/nvizble/Lightyear42/internal/tester"
)

// panel lists the results: one line per group (a ✓ and its count when it
// passed), the failures of failed groups under it. A group opens and closes
// on click; a failure opens its code.
type panel struct {
	report   *tester.Report
	err      error
	open     map[string]bool // groups showing their cases
	lines    []panelLine
	sel, top int
}

type panelLine struct {
	text   string
	group  string       // set on a group's line and its cases'
	c      *tester.Case // set on a case's lines
	detail string       // the line of c.Detail it shows, if any
}

func newPanel(report *tester.Report, err error) panel {
	p := panel{report: report, err: err, open: map[string]bool{}}
	if report != nil {
		for _, g := range report.Groups() {
			p.open[g.Name] = g.Failed() > 0
		}
	}
	p.build()
	return p
}

func (p *panel) toggle(group string) {
	p.open[group] = !p.open[group]
	p.build()
}

func (p *panel) build() {
	p.lines = p.lines[:0]
	if p.err != nil {
		for line := range strings.SplitSeq(p.err.Error(), "\n") {
			p.lines = append(p.lines, panelLine{text: styleFail.Render(line)})
		}
		return
	}
	if p.report == nil {
		p.lines = append(p.lines, panelLine{text: styleMuted.Render(
			"Os testes compilam o seu projeto numa cópia (make, norminette, cada função em vários casos)")})
		return
	}
	for _, g := range p.report.Groups() {
		arrow := "▸"
		if p.open[g.Name] {
			arrow = "▾"
		}
		var head string
		switch failed := g.Failed(); {
		case failed > 0:
			head = styleFail.Render(fmt.Sprintf("%s ✗ %s", arrow, g.Name)) + styleMuted.Render(fmt.Sprintf("  %d de %d falharam", failed, len(g.Cases)))
		case g.Skipped():
			head = styleMuted.Render(fmt.Sprintf("%s – %s  (pulado)", arrow, g.Name))
		default:
			head = styleGood.Render(fmt.Sprintf("%s ✓ %s", arrow, g.Name)) + styleMuted.Render(fmt.Sprintf("  %d", len(g.Cases)))
		}
		p.lines = append(p.lines, panelLine{text: head, group: g.Name})
		if !p.open[g.Name] {
			continue
		}
		for _, c := range g.Cases {
			c := c
			mark := styleGood.Render("✓")
			switch {
			case c.Status == tester.Skip:
				mark = styleMuted.Render("–")
			case c.Status.Failed():
				mark = styleFail.Render("✗")
			}
			text := "    " + mark + " " + c.Name
			if c.Status.Failed() && c.Status != tester.KO {
				text += styleFail.Render(" [" + string(c.Status) + "]")
			}
			p.lines = append(p.lines, panelLine{text: text, group: g.Name, c: &c})
			if c.Detail == "" {
				continue
			}
			for d := range strings.SplitSeq(c.Detail, "\n") {
				style := styleMuted
				if c.Status.Failed() {
					style = styleFail
				}
				p.lines = append(p.lines, panelLine{text: "        " + style.Render(d), group: g.Name, c: &c, detail: d})
			}
		}
	}
}

func (p *panel) move(d, height int) {
	p.sel = max(0, min(len(p.lines)-1, p.sel+d))
	if p.sel < p.top {
		p.top = p.sel
	}
	if p.sel >= p.top+height {
		p.top = p.sel - height + 1
	}
}

func (p *panel) scroll(d, height int) {
	p.top = max(0, min(len(p.lines)-height, p.top+d))
}

// render draws the title line and height-1 lines of results.
func (p *panel) render(width, height int, focused bool, title string) []string {
	head := styleTitle.Render(" Resultados ") + styleBar.Render(" "+title+" ")
	if focused {
		head += styleBar.Render(styleMuted.Render(" j/k anda · enter abre · Space t fecha "))
	}
	out := []string{ansi.Truncate(head, width, "") + styleBar.Render(strings.Repeat(" ", max(width-ansi.StringWidth(head), 0)))}
	for i := range height - 1 {
		line := ""
		if r := p.top + i; r < len(p.lines) {
			line = ansi.Truncate(p.lines[r].text, width, "…")
			if focused && r == p.sel {
				line = styleSel.Render(ansi.Strip(line) + strings.Repeat(" ", max(width-ansi.StringWidth(line), 0)))
			}
		}
		out = append(out, line)
	}
	return out
}

var (
	// normLine is norminette's "(line: 12"; compilerLine a compiler's
	// "path/ft_split.c:12:".
	normLine     = regexp.MustCompile(`\(line:\s*(\d+)`)
	compilerLine = regexp.MustCompile(`([\w./-]+\.(?:c|h|py)):(\d+):`)
	traceLine    = regexp.MustCompile(`File "([^"]+\.py)", line (\d+)`)
	pythonGroup  = regexp.MustCompile(`^(ex\d+) (\S+)$`)
)

// inRoot maps a file named in an error to the project: the tests run on a
// copy (…/proj/ex0/x.py), and compilers print relative or bare names.
func inRoot(root, name string) string {
	if i := strings.LastIndex(name, "/proj/"); i >= 0 {
		name = name[i+len("/proj/"):]
	}
	for _, candidate := range []string{filepath.Join(root, name), filepath.Join(root, filepath.Base(name))} {
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

// locate is where a failure points: the file a compiler error names, the
// file norminette checked, the Makefile, the README, or the function's own
// definition. detail is the line of the failure that was picked (the first
// one when empty).
func locate(root string, c tester.Case, detail string) (string, int) {
	if detail == "" {
		detail = c.Detail
	}
	for _, re := range []*regexp.Regexp{traceLine, compilerLine} {
		if m := re.FindStringSubmatch(detail); m != nil {
			if path := inRoot(root, m[1]); path != "" {
				line, _ := strconv.Atoi(m[2])
				return path, line
			}
		}
	}
	// "ex2 ft_plot_area": the exercise's file.
	if m := pythonGroup.FindStringSubmatch(c.Group); m != nil {
		if path := filepath.Join(root, m[1], m[2]+".py"); fileExists(path) {
			return path, 0
		}
	}
	line := 0
	if m := normLine.FindStringSubmatch(detail); m != nil {
		line, _ = strconv.Atoi(m[1])
	}
	for _, name := range []string{c.Name, c.Group} {
		if path := filepath.Join(root, name); fileExists(path) {
			return path, line
		}
	}
	switch c.Group {
	case "Makefile", "norminette", "README.md":
		return "", 0
	}
	return definition(root, c.Group)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// definition finds the line where function fn is defined in the project's
// .c files (a line naming it that doesn't end in ";").
func definition(root, fn string) (string, int) {
	if !strings.HasPrefix(fn, "ft_") {
		return "", 0
	}
	if path := filepath.Join(root, fn+".c"); fileExists(path) {
		if line := findDef(path, fn); line > 0 {
			return path, line
		}
		return path, 0
	}
	var found string
	var foundLine int
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != root {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".c" {
			if line := findDef(path, fn); line > 0 {
				found, foundLine = path, line
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found, foundLine
}

func findDef(path, fn string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()
	def := regexp.MustCompile(`^[A-Za-z_].*\b` + regexp.QuoteMeta(fn) + `\s*\(`)
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		text := sc.Text()
		if def.MatchString(text) && !strings.HasSuffix(strings.TrimSpace(text), ";") {
			return n
		}
	}
	return 0
}
