package exam

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Python exercises: the grader writes a harness that imports the function
// and prints repr() of what each call returns (or the exception it raises),
// one line per call, and compares that with the reference's output.

// pyHarnessFile is the harness's name, chosen not to collide with a submission.
const pyHarnessFile = "lightyear_main.py"

const pyHarness = `import ast
import sys

sys.path.insert(0, ".")
from %s import %s as function  # noqa: E402

for call in sys.argv[1:]:
    args = eval("(" + call + ",)")  # our tests, not the submission
    try:
        out = repr(function(*args))
    except Exception as e:  # the type is compared, not the message
        out = "raise " + type(e).__name__
    print(out, flush=True)
`

// pyChecker reports what a file uses from the forbidden list: builtins
// called by name ("sorted"), methods by their name ("list.sort" → .sort()),
// and modules imported ("itertools").
const pyChecker = `import ast
import sys

path, forbidden = sys.argv[1], [f.removesuffix("()") for f in sys.argv[2:]]
names = {f for f in forbidden if "." not in f}
methods = {f.rsplit(".", 1)[1] for f in forbidden if "." in f}
try:
    tree = ast.parse(open(path, encoding="utf-8").read(), path)
except SyntaxError:
    sys.exit(0)  # running it reports the error
for n in ast.walk(tree):
    if isinstance(n, ast.Call):
        f = n.func
        if isinstance(f, ast.Name) and f.id in names:
            print("%s() na linha %d" % (f.id, n.lineno))
        elif isinstance(f, ast.Attribute) and f.attr in methods:
            print(".%s() na linha %d" % (f.attr, n.lineno))
    elif isinstance(n, ast.Import):
        for a in n.names:
            if a.name.split(".")[0] in names:
                print("import %s na linha %d" % (a.name, n.lineno))
    elif isinstance(n, ast.ImportFrom) and (n.module or "").split(".")[0] in names:
        print("from %s import na linha %d" % (n.module, n.lineno))
`

func (g Grader) python() string {
	if g.Python != "" {
		return g.Python
	}
	return "python3"
}

// buildPython writes the sources and the harness into dir; "building" is
// only that (a syntax error shows up as a traceback when it runs).
func (g Grader) buildPython(_ context.Context, dir string, sources map[string]string, ex Exercise) (string, string, error) {
	files := make(map[string]string, len(sources)+len(ex.Provided)+1)
	maps.Copy(files, sources)
	maps.Copy(files, ex.Provided)
	module := strings.TrimSuffix(ex.Files[0], ".py")
	files[pyHarnessFile] = fmt.Sprintf(pyHarness, module, ex.Function)
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			return "", "", fmt.Errorf("preparar a correção: %w", err)
		}
	}
	return filepath.Join(dir, pyHarnessFile), "", nil
}

// forbiddenUse lists what src uses from forbidden ("" when nothing).
func (g Grader) forbiddenUse(ctx context.Context, src string, forbidden []string) (string, error) {
	f, err := os.CreateTemp("", "lightyear-check-*.py")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.WriteString(src); err != nil {
		return "", err
	}
	_ = f.Close()
	out, err := exec.CommandContext(ctx, g.python(), append([]string{"-c", pyChecker, f.Name()}, forbidden...)...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("checar o uso proibido: %s", exitErr.Stderr)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// pythonTrace explains a failed Python test: the call that failed and,
// when the submission raised, the exception instead of the output.
func pythonTrace(ex Exercise, calls []string, want, got, stderr string) string {
	wantLines := strings.Split(strings.TrimRight(want, "\n"), "\n")
	gotLines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	for i, call := range calls {
		w, g := "", ""
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w == g {
			continue
		}
		head := fmt.Sprintf("%s(%s)\nesperado: %s\n", ex.Function, call, w)
		if i >= len(gotLines) || got == "" {
			if line := lastErrorLine(stderr); line != "" {
				return head + "erro: " + line
			}
		}
		return head + "recebido: " + g
	}
	return fmt.Sprintf("esperado:\n%s\nrecebido:\n%s", want, got)
}

// lastErrorLine is the exception line of a traceback ("ValueError: ...").
func lastErrorLine(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" && !strings.HasPrefix(line, "File ") && !strings.HasPrefix(line, "^") {
			return line
		}
	}
	return ""
}
