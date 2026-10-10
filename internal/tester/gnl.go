package tester

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// gnlSizes are the BUFFER_SIZEs the suite runs with: the tiny ones, the
// usual one, one bigger than most lines and the subject's 10000000.
var gnlSizes = []int{1, 2, 5, 42, 1000, 10000000}

var gnlFiles = []string{"get_next_line.c", "get_next_line_utils.c"}

var gnlBonusFiles = []string{"get_next_line_bonus.c", "get_next_line_utils_bonus.c"}

var getNextLine = Project{
	Name:    "get_next_line",
	Subject: "42cursus-get_next_line",
	Markers: []string{"get_next_line.c", "get_next_line_utils.c", "get_next_line.h"},
	run:     runGNL,
}

func runGNL(ctx context.Context, r *runner) error {
	if _, err := lookPath(r.opts.CC); err != nil {
		return fmt.Errorf("compilador %q não encontrado: %w", r.opts.CC, err)
	}
	r.readme("README.md")
	r.norminette(ctx, "norminette")

	if !r.gnlChecks(ctx, "compilação", gnlFiles, false) {
		r.add("testes", "rodar os testes", Skip, "o projeto não compilou com -Wall -Wextra -Werror")
		return nil
	}
	build := filepath.Join(r.work, "build")
	if err := writeEmbedded(build); err != nil {
		return err
	}
	for _, size := range gnlSizes {
		r.step(fmt.Sprintf("rodando os testes (BUFFER_SIZE=%d)", size))
		if err := r.runGNLSuite(ctx, size, gnlFiles, "gnl", ""); err != nil {
			return err
		}
	}

	var missing []string
	for _, f := range append(slices.Clone(gnlBonusFiles), "get_next_line_bonus.h") {
		if _, err := os.Stat(filepath.Join(r.proj, f)); err != nil {
			missing = append(missing, f)
		}
	}
	if len(missing) == 3 {
		r.add("bônus", "get_next_line_bonus.c, _utils_bonus.c e _bonus.h", Skip, "o bônus não foi feito")
		return nil
	}
	if len(missing) > 0 {
		r.add("bônus", "arquivos do bônus", KO, "falta "+strings.Join(missing, ", "))
		return nil
	}
	if !r.gnlChecks(ctx, "bônus", gnlBonusFiles, true) {
		return nil
	}
	for _, size := range []int{1, 42, 10000000} {
		r.step(fmt.Sprintf("rodando o bônus (BUFFER_SIZE=%d)", size))
		if err := r.runGNLSuite(ctx, size, gnlBonusFiles, "gnl", "bônus: "); err != nil {
			return err
		}
		if err := r.runGNLSuite(ctx, size, gnlBonusFiles, "gnl_bonus", ""); err != nil {
			return err
		}
	}
	return nil
}

// gnlChecks compiles files with -Wall -Wextra -Werror, with and without
// -D BUFFER_SIZE, and checks the objects: only read, malloc and free, no
// globals and, for the bonus, a single static variable.
func (r *runner) gnlChecks(ctx context.Context, group string, files []string, bonus bool) bool {
	objs := filepath.Join(r.work, "objs-"+group)
	if err := os.MkdirAll(objs, 0o755); err != nil {
		r.add(group, "compila", KO, err.Error())
		return false
	}
	var built []string
	ok := true
	for _, define := range [][]string{{"-D", "BUFFER_SIZE=42"}, nil} {
		for _, f := range files {
			obj := filepath.Join(objs, strings.TrimSuffix(f, ".c")+".o")
			args := append([]string{"-Wall", "-Wextra", "-Werror", "-c"}, define...)
			cmd := exec.CommandContext(ctx, r.opts.CC, append(args, f, "-o", obj)...)
			cmd.Dir = r.proj
			out, err := cmd.CombinedOutput()
			name := "compila com -Wall -Wextra -Werror -D BUFFER_SIZE=42"
			if define == nil {
				name = "compila sem -D BUFFER_SIZE (precisa de um valor padrão)"
			}
			if err != nil {
				r.add(group, name+": "+f, KO, tail(string(out), 8))
				ok = false
				continue
			}
			if define != nil {
				built = append(built, obj)
			}
		}
	}
	if !ok {
		return false
	}
	r.add(group, "compila com -Wall -Wextra -Werror, com e sem -D BUFFER_SIZE", OK, "")

	defined := map[string]string{}
	undefined := map[string]bool{}
	statics := 0
	var staticNames []string
	for _, obj := range built {
		d, u, err := symbols(ctx, obj)
		if err != nil {
			r.add(group, "nm", KO, err.Error())
			return false
		}
		for k, v := range d {
			defined[k] = v
		}
		for k := range u {
			undefined[k] = true
		}
		names, err := localData(ctx, obj)
		if err != nil {
			r.add(group, "nm", KO, err.Error())
			return false
		}
		statics += len(names)
		staticNames = append(staticNames, names...)
	}
	for name := range defined {
		delete(undefined, name)
	}
	r.forbidden(group, undefined, "read", "malloc", "free")
	r.globals(group, defined)
	if bonus {
		r.check(group, "só uma variável static", statics <= 1,
			fmt.Sprintf("%d variáveis static: %s (o bônus pede uma só)", statics, strings.Join(staticNames, ", ")))
	}
	return true
}

// localData lists the static variables of an object file (nm's local
// data and bss symbols).
func localData(ctx context.Context, obj string) ([]string, error) {
	out, err := exec.CommandContext(ctx, "nm", obj).Output()
	if err != nil {
		return nil, fmt.Errorf("nm %s: %w", filepath.Base(obj), err)
	}
	var names []string
	for line := range strings.SplitSeq(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || (f[1] != "b" && f[1] != "d") {
			continue
		}
		name := f[2]
		if runtime.GOOS == "darwin" {
			name = strings.TrimPrefix(name, "_")
		}
		if strings.HasPrefix(name, "ltmp") || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "l_") {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// runGNLSuite builds suite (c/get_next_line/<suite>.c) with the student's
// files and -D BUFFER_SIZE=size, and runs it.
func (r *runner) runGNLSuite(ctx context.Context, size int, files []string, suite, prefix string) error {
	build := filepath.Join(r.work, "build")
	c := filepath.Join(build, "c")
	bin := filepath.Join(build, fmt.Sprintf("%s_%d", suite, size))
	main := filepath.Join(build, suite+"_main.c")
	src := fmt.Sprintf("#include \"lt.h\"\nvoid lt_suite_%s(void);\nint main(int argc, char **argv)\n{\n\tlt_init(argc, argv);\n\tlt_suite_%s();\n\treturn (0);\n}\n", suite, suite)
	if err := os.WriteFile(main, []byte(src), 0o600); err != nil {
		return err
	}
	args := []string{"-g", "-w", fmt.Sprintf("-DBUFFER_SIZE=%d", size), "-DLT_PREFIX=\"" + prefix + "\"",
		"-I", c, "-I", r.proj, filepath.Join(c, "lt.c"), filepath.Join(c, "get_next_line", suite+".c"), main}
	for _, f := range files {
		args = append(args, filepath.Join(r.proj, f))
	}
	out, err := exec.CommandContext(ctx, r.opts.CC, append(args, "-o", bin)...).CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.add(fmt.Sprintf("%sBUFFER_SIZE=%d", prefix, size), "compilar os testes com o seu código", KO, tail(string(out), 10))
		return nil
	}
	return r.runBinary(ctx, bin)
}
