package tester

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
)

//go:embed c
var suitesFS embed.FS

var lookPath = exec.LookPath

// makeTimeout bounds each make; suiteTimeout the whole C suite run (each
// case also has its own 5s alarm).
const (
	makeTimeout  = 3 * time.Minute
	suiteTimeout = 10 * time.Minute
)

// runner holds one run: a copy of the project (so make and the tests never
// touch the student's folder) and the cases so far.
type runner struct {
	src   string // the student's folder
	work  string // temp dir: proj/ (the copy) and build/
	proj  string
	opts  Options
	cases []Case
}

func newRunner(src string, opts Options) (*runner, error) {
	work, err := os.MkdirTemp("", "lightyear-test-*")
	if err != nil {
		return nil, fmt.Errorf("criar diretório temporário: %w", err)
	}
	r := &runner{src: src, work: work, proj: filepath.Join(work, "proj"), opts: opts}
	if err := copyProject(src, r.proj); err != nil {
		r.close()
		return nil, fmt.Errorf("copiar o projeto: %w", err)
	}
	return r, nil
}

func (r *runner) close() { _ = os.RemoveAll(r.work) }

func (r *runner) add(group, name string, status Status, detail string) {
	r.cases = append(r.cases, Case{Group: group, Name: name, Status: status, Detail: detail})
}

// check adds an OK case, or a KO one with detail when ok is false.
func (r *runner) check(group, name string, ok bool, detail string) {
	if ok {
		detail = ""
	}
	r.add(group, name, map[bool]Status{true: OK, false: KO}[ok], detail)
}

func (r *runner) step(s string) {
	if r.opts.Progress != nil {
		r.opts.Progress(s)
	}
}

// copyProject copies the project's files, leaving out .git and build
// products (so make starts from a clean tree).
func copyProject(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir() && d.Name() == ".git":
			return filepath.SkipDir
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case !d.Type().IsRegular():
			return nil
		}
		if ext := filepath.Ext(path); ext == ".o" || ext == ".a" || ext == ".d" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// make runs make with args in the copy. A failing make is not an error:
// ok is false and out has the output.
func (r *runner) make(parent context.Context, args ...string) (out string, ok bool, err error) {
	ctx, cancel := context.WithTimeout(parent, makeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "make", args...)
	cmd.Dir = r.proj
	cmd.Env = append(os.Environ(), "MAKEFLAGS=", "MAKELEVEL=")
	data, runErr := cmd.CombinedOutput()
	if err := parent.Err(); err != nil {
		return "", false, err
	}
	if ctx.Err() != nil {
		return string(data) + "\nmake passou de " + makeTimeout.String(), false, nil
	}
	var exitErr *exec.ExitError
	if runErr != nil && !errors.As(runErr, &exitErr) {
		return "", false, fmt.Errorf("executar make: %w", runErr)
	}
	return string(data), runErr == nil, nil
}

// hasRule reports whether the Makefile has a rule for target.
func (r *runner) hasRule(ctx context.Context, target string) (bool, error) {
	out, ok, err := r.make(ctx, "-n", target)
	if err != nil {
		return false, err
	}
	return ok || !strings.Contains(out, "No rule to make target"), nil
}

// tail keeps the last n lines of s: the end of a build log is where the error is.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = append([]string{"…"}, lines[len(lines)-n:]...)
	}
	return strings.Join(lines, "\n")
}

// norminette runs the norm on the copy: one case per file.
func (r *runner) norminette(ctx context.Context, group string) {
	if r.opts.Norminette == "" {
		r.add(group, "norminette", Skip, "norminette não está instalada (pip install norminette)")
		return
	}
	r.step("norminette")
	cmd := exec.CommandContext(ctx, r.opts.Norminette)
	cmd.Dir = r.proj
	out, _ := cmd.CombinedOutput()
	const maxErrors = 12
	var cur *Case
	errors := 0
	flush := func() {
		if cur != nil {
			r.cases = append(r.cases, *cur)
			cur = nil
		}
	}
	for line := range strings.SplitSeq(ansi.Strip(string(out)), "\n") {
		switch {
		case strings.HasSuffix(line, ": OK!"):
			flush()
			r.add(group, strings.TrimSuffix(line, ": OK!"), OK, "")
		case strings.HasSuffix(line, ": Error!"):
			flush()
			cur, errors = &Case{Group: group, Name: strings.TrimSuffix(line, ": Error!"), Status: KO}, 0
		case cur != nil && strings.TrimSpace(line) != "":
			errors++
			switch {
			case errors <= maxErrors:
				if cur.Detail != "" {
					cur.Detail += "\n"
				}
				cur.Detail += strings.Join(strings.Fields(line), " ")
			case errors == maxErrors+1:
				cur.Detail += "\n…"
			}
		}
	}
	flush()
}

// symbols lists the global symbols an archive defines (name → nm kind:
// T for code, D/B/C for data) and the ones it needs from outside, without
// the "_" macOS puts in front of C names.
func symbols(ctx context.Context, lib string) (defined map[string]string, undefined map[string]bool, err error) {
	out, err := exec.CommandContext(ctx, "nm", lib).Output()
	if err != nil {
		return nil, nil, fmt.Errorf("nm %s: %w", filepath.Base(lib), err)
	}
	defined, undefined = map[string]string{}, map[string]bool{}
	for line := range strings.SplitSeq(string(out), "\n") {
		f := strings.Fields(line)
		var kind, name string
		switch len(f) {
		case 2:
			kind, name = f[0], f[1]
		case 3:
			kind, name = f[1], f[2]
		default:
			continue
		}
		if runtime.GOOS == "darwin" {
			name = strings.TrimPrefix(name, "_")
		}
		switch kind {
		case "U":
			undefined[name] = true
		case "T", "D", "B", "S", "C", "R", "V", "W":
			defined[name] = kind
		}
	}
	for name := range defined {
		delete(undefined, name)
	}
	return defined, undefined, nil
}

// compilerSymbols are pulled in by the compiler or the runtime, not by the
// student's code.
var compilerSymbols = map[string]bool{
	"__stack_chk_fail": true, "__stack_chk_guard": true, "dyld_stub_binder": true,
	"_GLOBAL_OFFSET_TABLE_": true, "__chkstk_darwin": true, "_DYNAMIC": true,
}

// forbidden adds the case for functions used from outside the allowed list.
func (r *runner) forbidden(group string, undefined map[string]bool, allowed ...string) {
	var bad []string
	for name := range undefined {
		if !compilerSymbols[name] && !strings.HasPrefix(name, "ltmp") && !slices.Contains(allowed, name) {
			bad = append(bad, name)
		}
	}
	slices.Sort(bad)
	r.check(group, "só usa "+strings.Join(allowed, ", "), len(bad) == 0,
		"usa função não permitida: "+strings.Join(bad, ", "))
}

// suite is one C file of tests, for one function.
type suite struct {
	dir   string // under c/
	name  string // the function; the file is <name>.c with lt_suite_<name>()
	group string
}

// runSuites compiles each suite against the student's headers (one by one:
// a header that breaks one suite doesn't take the others down), links the
// ones that compiled with the library and adds the cases they report.
func (r *runner) runSuites(ctx context.Context, suites []suite, includes []string, link []string) error {
	build := filepath.Join(r.work, "build")
	if err := writeEmbedded(build); err != nil {
		return err
	}
	r.step("compilando os testes")
	flags := []string{"-c", "-g", "-w", "-I", filepath.Join(build, "c")}
	for _, inc := range includes {
		flags = append(flags, "-I", inc)
	}
	type result struct {
		obj, out string
		ok       bool
	}
	results := make([]result, len(suites))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for i, s := range suites {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			src := filepath.Join(build, "c", s.dir, s.name+".c")
			obj := filepath.Join(build, s.name+".o")
			cmd := exec.CommandContext(ctx, r.opts.CC, append(slices.Clone(flags), "-I", filepath.Join(build, "c", s.dir), src, "-o", obj)...)
			out, err := cmd.CombinedOutput()
			results[i] = result{obj: obj, out: string(out), ok: err == nil}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}

	var main strings.Builder
	main.WriteString("#include \"lt.h\"\n")
	var objs, calls []string
	for i, s := range suites {
		if !results[i].ok {
			r.add(s.group, "os testes compilam com o seu header", KO, tail(results[i].out, 8))
			continue
		}
		objs = append(objs, results[i].obj)
		fmt.Fprintf(&main, "void lt_suite_%s(void);\n", s.name)
		calls = append(calls, fmt.Sprintf("\tlt_suite_%s();\n", s.name))
	}
	if len(objs) == 0 {
		return nil
	}
	main.WriteString("int main(int argc, char **argv)\n{\n\tlt_init(argc, argv);\n")
	main.WriteString(strings.Join(calls, ""))
	main.WriteString("\treturn (0);\n}\n")
	mainSrc := filepath.Join(build, "lt_main.c")
	if err := os.WriteFile(mainSrc, []byte(main.String()), 0o600); err != nil {
		return err
	}
	bin := filepath.Join(build, "lt_tests")
	args := append([]string{"-g", "-w", "-I", filepath.Join(build, "c"), filepath.Join(build, "c", "lt.c"), mainSrc}, objs...)
	args = append(args, link...)
	args = append(args, "-o", bin)
	if out, err := exec.CommandContext(ctx, r.opts.CC, args...).CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.add("testes", "linkar os testes com a sua biblioteca", KO, tail(string(out), 10))
		return nil
	}

	r.step("rodando os testes")
	return r.runBinary(ctx, bin)
}

// runBinary runs the linked suites and turns their result lines into cases.
func (r *runner) runBinary(ctx context.Context, bin string) error {
	ctx, cancel := context.WithTimeout(ctx, suiteTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin)
	cmd.Dir = filepath.Dir(bin)
	cmd.Env = append(os.Environ(), "TMPDIR="+filepath.Dir(bin))
	if r.opts.Timeout > 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("LT_TIMEOUT=%d", max(int(r.opts.Timeout.Seconds()), 1)))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("rodar os testes: %w", err)
	}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		f := strings.SplitN(sc.Text(), "\t", 5)
		if len(f) != 5 || f[0] != "R" {
			continue
		}
		r.add(f[1], f[2], Status(f[3]), f[4])
	}
	_, _ = io.Copy(io.Discard, stdout)
	err = cmd.Wait()
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			r.add("testes", "terminar em "+suiteTimeout.String(), Timeout, "os testes não terminaram a tempo")
			return nil
		}
		return ctx.Err()
	}
	if err != nil {
		r.add("testes", "o executável dos testes terminou bem", KO, fmt.Sprintf("%v\n%s", err, tail(stderr.String(), 5)))
	}
	return nil
}

// writeEmbedded writes the runtime and suites under dir/c.
func writeEmbedded(dir string) error {
	return fs.WalkDir(suitesFS, "c", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := suitesFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
}

// findFile returns the directory holding name inside root (the project's
// headers may live in a subfolder like includes/).
func findFile(root, name string) (string, bool) {
	var found string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == name {
			found = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	return found, found != ""
}
