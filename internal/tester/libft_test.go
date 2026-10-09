package tester

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func needToolchain(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("os testes precisam de fork")
	}
	for _, tool := range []string{"cc", "make", "nm", "ar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s não encontrado", tool)
		}
	}
}

// edit replaces old with new in file.
type edit struct{ file, old, new string }

// fixture copies the reference libft into a temp dir and applies edits to it.
func fixture(t *testing.T, edits ...edit) string {
	t.Helper()
	dir := t.TempDir()
	if err := copyProject(filepath.Join("testdata", "libft"), dir); err != nil {
		t.Fatal(err)
	}
	for _, e := range edits {
		path := filepath.Join(dir, e.file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), e.old) {
			t.Fatalf("%s não contém %q", e.file, e.old)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(data), e.old, e.new, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runLibftAt(t *testing.T, dir string, only ...string) Report {
	t.Helper()
	rep, err := libft.Run(context.Background(), dir, Options{CC: "cc", Only: only, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// The reference libft passes every function suite and the build checks;
// it fails only what it doesn't try to follow (README, file names).
func TestLibftReferencePasses(t *testing.T) {
	needToolchain(t)
	t.Parallel()
	rep := runLibftAt(t, fixture(t))
	passed, _, _ := rep.Count()
	if passed < 700 {
		t.Errorf("só %d casos passaram; os testes encolheram?", passed)
	}
	groups := map[string]bool{}
	for _, c := range rep.Cases {
		groups[c.Group] = true
		if c.Group == "README.md" || c.Group == "arquivos" {
			continue
		}
		if c.Status.Failed() {
			t.Errorf("%s / %s: %s %s", c.Group, c.Name, c.Status, c.Detail)
		}
	}
	for _, fn := range libftFunctions {
		if !groups[fn] {
			t.Errorf("nenhum caso para %s", fn)
		}
	}
}

// Each bug planted in the reference must be caught by the right check.
func TestLibftCatchesBugs(t *testing.T) {
	needToolchain(t)
	tests := []struct {
		name   string
		edits  []edit
		group  string
		status Status
		detail string
	}{
		{"strlen off by one",
			[]edit{{"part1.c", "return (n);\n}\n\nvoid\t*ft_memset", "return (n + 1);\n}\n\nvoid\t*ft_memset"}},
			"ft_strlen", KO, "esperado 0, recebido 1"},
		{"isalpha returns non-zero instead of 1",
			[]edit{{"part1.c", "int\tft_isalpha(int c) { return (", "int\tft_isalpha(int c) { return (1024 * !!("},
				{"part1.c", "(c >= 'A' && c <= 'Z')); }", "(c >= 'A' && c <= 'Z'))); }"}},
			"ft_isalpha", KO, "exatamente 1"},
		{"split doesn't free on failure",
			[]edit{{"part2.c", "\t\t\twhile (k > 0)\n\t\t\t\tfree(r[--k]);\n", ""}},
			"ft_split", KO, "não liberou"},
		{"split doesn't check malloc",
			[]edit{{"part2.c", "\tif (!r)\n\t\treturn (NULL);\n\twhile (k < n)", "\twhile (k < n)"}},
			"ft_split", Crash, "º malloc"},
		{"strjoin forgets the +1",
			[]edit{{"part2.c", "malloc(a + b + 1)", "malloc(a + b)"}},
			"ft_strjoin", KO, "heap overflow"},
		{"substr returns NULL past the end",
			[]edit{{"part2.c", "return (ft_strdup(\"\"));", "return (NULL);"}},
			"ft_substr", KO, "devolveu NULL"},
		{"memmove always copies forward",
			[]edit{{"part1.c", "\tif (d > s)", "\tif (0 && d > s)"}},
			"ft_memmove", KO, "dst = buf+"},
		{"itoa breaks INT_MIN",
			[]edit{{"part2.c", "\tlong\tv = n;\n\n\tbuf[i] = '\\0';", "\tint\tv = n;\n\n\tbuf[i] = '\\0';"}},
			"ft_itoa", KO, "-2147483648"},
		{"strlcat loops forever",
			[]edit{{"part1.c", "\twhile (dl < size && dst[dl])\n\t\tdl++;", "\twhile (dl < size && dst[dl])\n\t\t;"}},
			"ft_strlcat", Timeout, "loop infinito"},
		{"lstsize dereferences NULL",
			[]edit{{"bonus.c", "\tfor (; lst; lst = lst->next)\n\t\tn++;", "\tdo\n\t\tn++;\n\twhile ((lst = lst->next));"}},
			"ft_lstsize", Crash, ""},
		{"calloc ignores overflow",
			[]edit{{"part1.c", "\tif (size && count > (size_t)-1 / size)\n\t\treturn (NULL);\n", ""}},
			"ft_calloc", KO, "estour"},
		{"calloc(0, 0) returns NULL",
			[]edit{{"part1.c", "\tif (size && count", "\tif (!size || !count)\n\t\treturn (NULL);\n\tif (size && count"}},
			"ft_calloc", KO, "ponteiro único"},
		{"strdup leaks",
			[]edit{{"part1.c", "\tchar\t*d = malloc(n);", "\tchar\t*d = malloc(n);\n\tchar\t*leak = malloc(1);\n\n\t(void)leak;"}},
			"ft_strdup", KO, "vazamento"},
		{"lstdelone frees twice",
			[]edit{{"bonus.c", "\tdel(lst->content);\n\tfree(lst);", "\tt_list\t*volatile again = lst;\n\n\tdel(lst->content);\n\tfree(lst);\n\tfree(again);"}},
			"ft_lstdelone", KO, "double free"},
		{"lstmap forgets del on failure",
			[]edit{{"bonus.c", "\t\t\tdel(c);\n", ""}},
			"ft_lstmap", KO, "del só foi chamada"},
		{"putstr writes to stdout",
			[]edit{{"part2.c", "void\tft_putstr_fd(char *s, int fd) { write(fd,", "void\tft_putstr_fd(char *s, int fd) { (void)fd; write(1,"}},
			"ft_putstr_fd", KO, "escreveu \"\""},
		{"global variable",
			[]edit{{"part1.c", "#include \"libft.h\"\n", "#include \"libft.h\"\n\nint\tg_calls = 1;\n"}},
			"variáveis globais", KO, "g_calls"},
		{"forbidden function",
			[]edit{{"part2.c", "#include <stdlib.h>\n", "#include <stdlib.h>\n#include <string.h>\n"},
				{"part2.c", "\tsize_t\tsl = ft_strlen(s);\n\n\tif (start", "\tsize_t\tsl = strlen(s);\n\n\tif (start"}},
			"funções permitidas", KO, "strlen"},
		{"Makefile relinks",
			[]edit{{"Makefile", "all: $(NAME)\n", "all: $(NAME)\n\t@touch $(NAME)\n"}},
			"Makefile", KO, "foi refeita"},
		{"Makefile without -Werror",
			[]edit{{"Makefile", "CFLAGS = -Wall -Wextra -Werror", "CFLAGS = -Wall -Wextra"}},
			"Makefile", KO, "falta -Werror"},
		{"Makefile without re",
			[]edit{{"Makefile", "re: fclean all\n", ""}},
			"Makefile", KO, "a regra re não existe"},
		{"missing function",
			[]edit{{"part2.c", "void\tft_putnbr_fd(int n, int fd)\n{", "void\tft_putnbr_gone(int n, int fd)\n{"},
				{"part2.c", "ft_putnbr_fd((int)(v / 10), fd);", "ft_putnbr_gone((int)(v / 10), fd);"}},
			"ft_putnbr_fd", KO, "não está na libft.a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			only := []string{tt.group}
			if !strings.HasPrefix(tt.group, "ft_") {
				only = []string{"ft_strlen"}
			}
			rep := runLibftAt(t, fixture(t, tt.edits...), only...)
			for _, c := range rep.Cases {
				if c.Group == tt.group && c.Status == tt.status && strings.Contains(c.Detail, tt.detail) {
					return
				}
			}
			var got []string
			for _, c := range rep.Cases {
				if c.Group == tt.group || c.Status.Failed() {
					got = append(got, string(c.Status)+" "+c.Group+" / "+c.Name+": "+c.Detail)
				}
			}
			t.Errorf("nenhum %s em %s com %q; casos:\n%s", tt.status, tt.group, tt.detail, strings.Join(got, "\n"))
		})
	}
}

// A project that doesn't compile fails the build check and skips the rest
// instead of erroring out.
func TestLibftCompileError(t *testing.T) {
	needToolchain(t)
	t.Parallel()
	rep := runLibftAt(t, fixture(t, edit{"part1.c", "return (n);\n}\n\nvoid\t*ft_memset", "return (n)\n}\n\nvoid\t*ft_memset"}))
	var build, skipped bool
	for _, c := range rep.Cases {
		build = build || (c.Name == "make compila libft.a" && c.Status == KO && strings.Contains(c.Detail, "error"))
		skipped = skipped || (c.Group == "testes" && c.Status == Skip)
	}
	if !build || !skipped {
		t.Errorf("esperava o make falhando e os testes pulados: %+v", rep.Cases)
	}
}

func TestCheckRoot(t *testing.T) {
	if err := libft.CheckRoot(t.TempDir()); err == nil || !strings.Contains(err.Error(), "raiz") {
		t.Errorf("CheckRoot numa pasta vazia = %v", err)
	}
	if err := libft.CheckRoot(filepath.Join("testdata", "libft")); err != nil {
		t.Errorf("CheckRoot no fixture = %v", err)
	}
}

func TestReadme(t *testing.T) {
	tests := []struct {
		readme string
		failed []string
	}{
		{"*This project has been created as part of the 42 curriculum by jdoe.*\n## Description\n## Instructions\n## Resources\n", nil},
		{"_This project has been created as part of the 42 curriculum by jdoe, asmith._\n# Description\n# Instructions\n# Resources\n", nil},
		{"This project has been created as part of the 42 curriculum by jdoe.\n## Description\n## Instructions\n## Resources\n",
			[]string{"primeira linha em itálico com os logins"}},
		{"*This project has been created as part of the 42 curriculum by jdoe.*\n## Description\n",
			[]string{"seção Instructions", "seção Resources"}},
	}
	for _, tt := range tests {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(tt.readme), 0o644); err != nil {
			t.Fatal(err)
		}
		r := &runner{proj: dir}
		r.readme("README.md")
		var failed []string
		for _, c := range r.cases {
			if c.Status != OK {
				failed = append(failed, c.Name)
			}
		}
		if strings.Join(failed, "|") != strings.Join(tt.failed, "|") {
			t.Errorf("README %q: falhou %v, esperado %v", tt.readme, failed, tt.failed)
		}
	}
}

func TestReportGroups(t *testing.T) {
	rep := Report{Cases: []Case{
		{Group: "a", Status: OK}, {Group: "b", Status: KO}, {Group: "a", Status: Crash}, {Group: "c", Status: Skip},
	}}
	groups := rep.Groups()
	if len(groups) != 3 || groups[0].Name != "a" || len(groups[0].Cases) != 2 || groups[0].Failed() != 1 || !groups[2].Skipped() {
		t.Errorf("Groups() = %+v", groups)
	}
	if p, f, s := rep.Count(); p != 1 || f != 2 || s != 1 {
		t.Errorf("Count() = %d %d %d", p, f, s)
	}
}
