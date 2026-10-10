package tester

import (
	"context"
	"strings"
	"testing"
	"time"
)

func runPrintfAt(t *testing.T, dir string) Report {
	t.Helper()
	rep, err := ftPrintf.Run(context.Background(), dir, Options{CC: "cc", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// The reference ft_printf passes the mandatory suite and the build checks;
// its bonus (the libc behind a wrapper) passes the bonus suites.
func TestPrintfReferencePasses(t *testing.T) {
	needToolchain(t)
	t.Parallel()
	rep := runPrintfAt(t, fixtureOf(t, "ft_printf"))
	groups := map[string]bool{}
	for _, c := range rep.Cases {
		groups[c.Group] = true
		if c.Group != "README.md" && c.Status.Failed() {
			t.Errorf("%s / %s: %s %s", c.Group, c.Name, c.Status, c.Detail)
		}
	}
	for _, g := range []string{"ft_printf: %p", "ft_printf: comportamento", "bônus: - 0 . e largura", "bônus: # espaço +"} {
		if !groups[g] {
			t.Errorf("nenhum caso em %s", g)
		}
	}
}

func TestPrintfCatchesBugs(t *testing.T) {
	needToolchain(t)
	tests := []struct {
		name   string
		edits  []edit
		group  string
		status Status
		detail string
	}{
		{"%u prints a signed number",
			[]edit{{"ft_printf.c", "return (pf_unsigned(va_arg(*ap, unsigned), 10, \"0123456789\"));", "return (pf_int(va_arg(*ap, int)));"}},
			"ft_printf: %u", KO, "4294967295"},
		{"%x in uppercase",
			[]edit{{"ft_printf.c", "return (pf_unsigned(va_arg(*ap, unsigned), 16, \"0123456789abcdef\"));", "return (pf_unsigned(va_arg(*ap, unsigned), 16, \"0123456789ABCDEF\"));"}},
			"ft_printf: %x", KO, "esperado"},
		{"%s NULL crashes",
			[]edit{{"ft_printf_utils.c", "\tif (!s)\n\t\ts = \"(null)\";\n", ""}},
			"ft_printf: %s", Crash, ""},
		{"%p without 0x",
			[]edit{{"ft_printf_utils.c", "\tif (pf_str(\"0x\") < 0)\n\t\treturn (-1);\n", ""}},
			"ft_printf: %p", KO, ""},
		{"%c skips the NUL byte",
			[]edit{{"ft_printf_utils.c", "\tb = (unsigned char)c;\n", "\tb = (unsigned char)c;\n\tif (!b)\n\t\treturn (0);\n"}},
			"ft_printf: %c", KO, "esperado"},
		{"INT_MIN overflows",
			[]edit{{"ft_printf_utils.c", "\tlong\tv;\n\tint\t\tr;\n\n\tv = n;", "\tint\t\tv;\n\tint\t\tr;\n\n\tv = n;"}},
			"ft_printf: %d", KO, "-2147483648"},
		{"wrong return value",
			[]edit{{"ft_printf.c", "\t\ttotal += r;", "\t\ttotal += (r > 1);"}},
			"ft_printf: %s", KO, "devolveu"},
		{"ignores write errors",
			[]edit{{"ft_printf.c", "\t\tif (r < 0)\n\t\t\treturn (va_end(ap), -1);\n", ""}},
			"ft_printf: comportamento", KO, "-1"},
		{"buffers with stdio",
			[]edit{{"ft_printf_utils.c", "#include <unistd.h>\n", "#include <stdio.h>\n#include <unistd.h>\n"},
				{"ft_printf_utils.c", "\treturn (write(1, s, n));\n}", "\treturn (fputs(s, stdout), n);\n}"}},
			"ft_printf: comportamento", KO, "bufferizou"},
		{"forbidden function",
			[]edit{{"ft_printf_utils.c", "#include <unistd.h>\n", "#include <string.h>\n#include <unistd.h>\n"},
				{"ft_printf_utils.c", "\tn = 0;\n\twhile (s[n])\n\t\tn++;\n", "\tn = (int)strlen(s);\n"}},
			"funções permitidas", KO, "strlen"},
		{"bonus rule without the bonus",
			[]edit{{"Makefile", "BONUS_OBJS = ft_printf_bonus.o ft_printf_utils.o", "BONUS_OBJS = ft_printf.o ft_printf_utils.o"}},
			"bônus: - 0 . e largura", Skip, "não foi feita"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rep := runPrintfAt(t, fixtureOf(t, "ft_printf", tt.edits...))
			for _, c := range rep.Cases {
				if c.Group == tt.group && c.Status == tt.status && strings.Contains(c.Name+" "+c.Detail, tt.detail) {
					return
				}
			}
			var got []string
			for _, c := range rep.Cases {
				if (c.Status.Failed() || c.Status == Skip) && c.Group != "README.md" && c.Group != "norminette" {
					got = append(got, string(c.Status)+" "+c.Group+" / "+c.Name+": "+c.Detail)
				}
			}
			t.Errorf("nenhum %s em %s com %q; casos:\n%s", tt.status, tt.group, tt.detail, strings.Join(got, "\n"))
		})
	}
}
