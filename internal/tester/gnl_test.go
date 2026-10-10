package tester

import (
	"context"
	"strings"
	"testing"
	"time"
)

func runGNLAt(t *testing.T, dir string) Report {
	t.Helper()
	rep, err := getNextLine.Run(context.Background(), dir, Options{CC: "cc", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// The reference get_next_line (and its bonus) passes every check but the
// README and the norm, which it doesn't try to follow.
func TestGNLReferencePasses(t *testing.T) {
	needToolchain(t)
	t.Parallel()
	rep := runGNLAt(t, fixtureOf(t, "get_next_line"))
	passed, _, _ := rep.Count()
	if passed < 250 {
		t.Errorf("só %d casos passaram", passed)
	}
	for _, c := range rep.Cases {
		if c.Group != "README.md" && c.Status.Failed() {
			t.Errorf("%s / %s: %s %s", c.Group, c.Name, c.Status, c.Detail)
		}
	}
}

func TestGNLCatchesBugs(t *testing.T) {
	needToolchain(t)
	tests := []struct {
		name   string
		edits  []edit
		group  string // prefix of the group
		status Status
		detail string
	}{
		{"never frees the old stash",
			[]edit{{"get_next_line.c", "\tfree(*stash);\n\t*stash = rest;", "\t*stash = rest;"}},
			"BUFFER_SIZE=42", KO, "vazamento"},
		{"reads to the end of the file first",
			[]edit{{"get_next_line.c", "while (!gnl_chr(stash, '\\n') && n > 0)", "while (n > 0)"}},
			"BUFFER_SIZE=42", Timeout, "esperou o fim do arquivo"},
		{"doesn't check malloc",
			[]edit{{"get_next_line_utils.c", "\tr = malloc(len + 1);\n\tif (!r)\n\t\treturn (NULL);\n", "\tr = malloc(len + 1);\n"}},
			"BUFFER_SIZE=42", Crash, "º malloc"},
		{"drops the newline",
			[]edit{{"get_next_line.c", "\t\tlen = (size_t)(nl - *stash) + 1;", "\t\tlen = (size_t)(nl - *stash);"}},
			"BUFFER_SIZE=42", KO, "linha 1"},
		{"no default BUFFER_SIZE",
			[]edit{{"get_next_line.h", "# ifndef BUFFER_SIZE\n#  define BUFFER_SIZE 42\n# endif\n", ""}},
			"compilação", KO, "BUFFER_SIZE"},
		{"uses lseek",
			[]edit{{"get_next_line.c", "\tif (fd < 0 || BUFFER_SIZE <= 0)", "\tif (fd < 0 || BUFFER_SIZE <= 0 || lseek(fd, 0, SEEK_CUR) < -1)"}},
			"compilação", KO, "lseek"},
		{"global variable",
			[]edit{{"get_next_line_utils.c", "#include \"get_next_line.h\"\n", "#include \"get_next_line.h\"\n\nint\tg_reads;\n"}},
			"compilação", KO, "g_reads"},
		{"bonus with two statics",
			[]edit{{"get_next_line_bonus.c", "\tstatic char\t*stash[4096];", "\tstatic char\t*stash[4096];\n\tstatic int\tcalls;\n\n\tif (++calls < 0)\n\t\treturn (NULL);"}},
			"bônus", KO, "variáveis static"},
		{"bonus mixes fds",
			[]edit{{"get_next_line_bonus.c", "\tstash[fd] = fill(fd, stash[fd]);\n\tif (!stash[fd] || !*stash[fd])\n\t{\n\t\tfree(stash[fd]);\n\t\tstash[fd] = NULL;\n\t\treturn (NULL);\n\t}\n\treturn (take(&stash[fd]));",
				"stash[0] = fill(fd, stash[0]);\n\tif (!stash[0] || !*stash[0])\n\t{\n\t\tfree(stash[0]);\n\t\tstash[0] = NULL;\n\t\treturn (NULL);\n\t}\n\treturn (take(&stash[0]));"}},
			"bônus: vários fds", KO, "fd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rep := runGNLAt(t, fixtureOf(t, "get_next_line", tt.edits...))
			for _, c := range rep.Cases {
				if strings.HasPrefix(c.Group, tt.group) && c.Status == tt.status && strings.Contains(c.Name+" "+c.Detail, tt.detail) {
					return
				}
			}
			var got []string
			for _, c := range rep.Cases {
				if c.Status.Failed() && c.Group != "README.md" && c.Group != "norminette" {
					got = append(got, string(c.Status)+" "+c.Group+" / "+c.Name+": "+c.Detail)
				}
			}
			t.Errorf("nenhum %s em %s com %q; falhas:\n%s", tt.status, tt.group, tt.detail, strings.Join(got, "\n"))
		})
	}
}
