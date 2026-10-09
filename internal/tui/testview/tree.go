package testview

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// tree is the project's files, folders first; folders open on click.
type tree struct {
	root     string
	expanded map[string]bool
	rows     []treeRow
	sel, top int
}

type treeRow struct {
	path  string
	name  string
	depth int
	dir   bool
}

func newTree(root string) tree {
	t := tree{root: root, expanded: map[string]bool{}}
	t.refresh()
	return t
}

// hidden leaves out dot files and build products.
func hidden(e os.DirEntry) bool {
	name := e.Name()
	switch {
	case strings.HasPrefix(name, "."), strings.HasSuffix(name, ".dSYM"), name == "__pycache__":
		return true
	case e.IsDir():
		return false
	}
	switch filepath.Ext(name) {
	case ".o", ".a", ".d", ".pyc":
		return true
	}
	return false
}

// refresh reads the folders again (files come and go with make and :w).
func (t *tree) refresh() {
	sel := ""
	if t.sel < len(t.rows) {
		sel = t.rows[t.sel].path
	}
	t.rows = t.rows[:0]
	t.walk(t.root, 0)
	t.selectPath(sel)
}

func (t *tree) walk(dir string, depth int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	for _, e := range entries {
		if hidden(e) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		t.rows = append(t.rows, treeRow{path: path, name: e.Name(), depth: depth, dir: e.IsDir()})
		if e.IsDir() && t.expanded[path] {
			t.walk(path, depth+1)
		}
	}
}

// selectPath selects path's row, opening the folders above it.
func (t *tree) selectPath(path string) {
	if path == "" {
		return
	}
	if rel, err := filepath.Rel(t.root, path); err == nil && !strings.HasPrefix(rel, "..") {
		opened := false
		for dir := filepath.Dir(path); dir != t.root && strings.HasPrefix(dir, t.root); dir = filepath.Dir(dir) {
			if !t.expanded[dir] {
				t.expanded[dir], opened = true, true
			}
		}
		if opened {
			t.refresh()
		}
	}
	for i, r := range t.rows {
		if r.path == path {
			t.sel = i
			return
		}
	}
}

func (t *tree) toggle(path string) {
	t.expanded[path] = !t.expanded[path]
	t.refresh()
}

// collapse closes the selected folder, or goes up to the parent's row.
func (t *tree) collapse() {
	if t.sel >= len(t.rows) {
		return
	}
	r := t.rows[t.sel]
	if r.dir && t.expanded[r.path] {
		t.toggle(r.path)
		return
	}
	t.selectPath(filepath.Dir(r.path))
}

func (t *tree) move(d int) {
	t.sel = max(0, min(len(t.rows)-1, t.sel+d))
}

func (t *tree) scroll(d, height int) {
	t.top = max(0, min(len(t.rows)-height, t.top+d))
}

// render draws height rows of width cells, keeping the selection in view.
func (t *tree) render(width, height int, focused bool, current string) []string {
	if t.sel < t.top {
		t.top = t.sel
	}
	if t.sel >= t.top+height {
		t.top = t.sel - height + 1
	}
	t.top = max(0, min(t.top, len(t.rows)-height))
	out := make([]string, height)
	for i := range out {
		line := ""
		if r := t.top + i; r < len(t.rows) {
			row := t.rows[r]
			icon := "  "
			if row.dir {
				icon = "▸ "
				if t.expanded[row.path] {
					icon = "▾ "
				}
			}
			line = ansi.Truncate(strings.Repeat("  ", row.depth)+icon+row.name, width, "…")
			pad := strings.Repeat(" ", max(width-ansi.StringWidth(line), 0))
			switch {
			case focused && r == t.sel:
				line = styleSel.Render(line + pad)
			case row.path == current:
				line = styleCurrent.Render(line) + pad
			case row.dir:
				line = styleMuted.Render(line) + pad
			default:
				line += pad
			}
		} else {
			line = strings.Repeat(" ", width)
		}
		out[i] = line
	}
	return out
}
