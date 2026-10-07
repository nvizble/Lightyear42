package exam

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// harnessFile is the name the function-exercise harness is compiled under,
// chosen so it never collides with a file the student turns in.
const harnessFile = "lightyear_main.c"

// refTimeout bounds each run of the reference solution, and the student's
// first run. It is generous on purpose: on macOS the first run of a freshly
// built binary can take seconds while Gatekeeper scans it, which must not be
// graded as a timeout. Later runs are judged by the per-test Timeout.
const refTimeout = 10 * time.Second

// maxOutput caps the stdout kept per run, so a program stuck printing in a
// loop cannot exhaust memory before the timeout kills it.
const maxOutput = 1 << 20

// Result is the verdict of one grading.
type Result struct {
	Passed bool
	// Trace explains the first failure (compile error, wrong output, crash,
	// timeout). Empty when the submission passed.
	Trace string
}

// Grader compiles a submission and the reference solution and compares
// their stdout over the exercise tests, like the 42 exam grader.
type Grader struct {
	// CC is the C compiler, invoked with -Wall -Wextra -Werror.
	CC string
	// Timeout bounds each test run.
	Timeout time.Duration
}

// NewGrader returns a grader using "cc" and a 2s timeout per test.
func NewGrader() Grader {
	return Grader{CC: "cc", Timeout: 2 * time.Second}
}

// Grade checks the files in submissionDir against the exercise.
//
// A failing submission is reported in Result; the error is reserved for
// problems unrelated to the student's code (no compiler, broken reference).
func (g Grader) Grade(ctx context.Context, ex Exercise, submissionDir string) (Result, error) {
	if _, err := exec.LookPath(g.CC); err != nil {
		return Result{}, fmt.Errorf("compilador %q não encontrado: %w", g.CC, err)
	}

	work, err := os.MkdirTemp("", "lightyear-exam-*")
	if err != nil {
		return Result{}, fmt.Errorf("criar diretório temporário: %w", err)
	}
	defer func() { _ = os.RemoveAll(work) }()

	refBin, out, err := g.build(ctx, filepath.Join(work, "ref"), ex.ref, ex)
	if err != nil {
		return Result{}, err
	}
	if refBin == "" {
		return Result{}, fmt.Errorf("solução de referência de %s não compila:\n%s", ex.Name, out)
	}

	sources := make(map[string]string, len(ex.Files))
	for _, name := range ex.Files {
		data, err := os.ReadFile(filepath.Join(submissionDir, name))
		if err != nil {
			return Result{Trace: fmt.Sprintf("arquivo não entregue: %s", name)}, nil
		}
		sources[name] = string(data)
	}
	userBin, out, err := g.build(ctx, filepath.Join(work, "user"), sources, ex)
	if err != nil {
		return Result{}, err
	}
	if userBin == "" {
		return Result{Trace: "erro de compilação:\n" + out}, nil
	}

	for i, args := range ex.Tests {
		want, err := run(ctx, refBin, args, refTimeout)
		if err != nil {
			return Result{}, fmt.Errorf("solução de referência de %s falhou com %q: %w", ex.Name, args, err)
		}
		timeout := g.Timeout
		if i == 0 {
			timeout = max(timeout, refTimeout)
		}
		got, err := run(ctx, userBin, args, timeout)
		if err != nil {
			return Result{Trace: fmt.Sprintf("%s %s\n%v", ex.Name, quoteArgs(args), err)}, nil
		}
		if got != want {
			return Result{Trace: fmt.Sprintf("%s %s\nesperado: %q\nrecebido: %q", ex.Name, quoteArgs(args), want, got)}, nil
		}
	}
	return Result{Passed: true}, nil
}

// build writes the sources, then the exercise's provided files (which win
// over a submitted file of the same name) and harness, into dir and compiles
// the .c files among them; headers are only written, for #include. A compile
// error is not an error: it yields an empty binary path and the compiler output.
func (g Grader) build(ctx context.Context, dir string, sources map[string]string, ex Exercise) (bin, output string, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", fmt.Errorf("preparar compilação: %w", err)
	}

	files := make(map[string]string, len(sources)+len(ex.Provided)+1)
	maps.Copy(files, sources)
	maps.Copy(files, ex.Provided)
	if ex.harness != "" {
		files[harnessFile] = ex.harness
	}

	args := []string{"-Wall", "-Wextra", "-Werror", "-o", "a.out"}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			return "", "", fmt.Errorf("preparar compilação: %w", err)
		}
		if strings.HasSuffix(name, ".c") {
			args = append(args, name)
		}
	}

	cmd := exec.CommandContext(ctx, g.CC, args...)
	cmd.Dir = dir
	combined, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", string(combined), nil
		}
		return "", "", fmt.Errorf("executar %s: %w", g.CC, err)
	}
	return filepath.Join(dir, "a.out"), "", nil
}

// run executes bin with args and returns its stdout. Timeouts and crashes
// are errors; a non-zero exit status is not (the exam only compares output).
func run(ctx context.Context, bin string, args []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout cappedBuffer
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = filepath.Dir(bin)
	cmd.Stdout = &stdout
	// Children that keep stdout open must not hang the grader past the timeout.
	cmd.WaitDelay = time.Second

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout: passou de %s", timeout)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == -1 {
			return "", fmt.Errorf("crash: %s", exitErr.ProcessState)
		}
		err = nil
	}
	return stdout.String(), err
}

// cappedBuffer keeps at most maxOutput bytes and silently drops the rest.
type cappedBuffer struct {
	bytes.Buffer
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := maxOutput - b.Len(); room < len(p) {
		b.Buffer.Write(p[:max(room, 0)])
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

// quoteArgs renders argv the way it would be typed in a shell.
func quoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = fmt.Sprintf("%q", a)
	}
	return strings.Join(quoted, " ")
}
