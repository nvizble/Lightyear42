"""lightyear tester runtime for the Python modules.

Usage: python3 lt.py <suite.py> <project root>

A suite defines suite(t). Every check runs the student's code in its own
python3 process (so a crash, an infinite loop or an exit() fails only that
case) and prints one line per case:

    R <tab> group <tab> name <tab> OK|KO|CRASH|TIMEOUT|SKIP <tab> detail
"""

import ast
import importlib.util
import os
import shutil
import subprocess
import sys
import traceback


class Fail(Exception):
    """A failed expectation (raised by T.expect)."""


class Result:
    """What a run of the student's code produced."""

    def __init__(self, out, err, code, timed_out):
        self.out = out
        self.err = err
        self.code = code
        self.timed_out = timed_out

    @property
    def crashed(self):
        """An uncaught exception (a traceback on stderr) or a kill."""
        return self.code != 0 and ("Traceback" in self.err or self.code < 0)

    def last_error(self):
        """The last line of the traceback ("ValueError: ...")."""
        lines = [line for line in self.err.strip().splitlines() if line.strip()]
        return lines[-1] if lines else "código de saída %d" % self.code


def clip(s, n=300):
    s = str(s)
    return s if len(s) <= n else s[:n] + "…"


def show(s):
    """A string the way the report shows it: quoted, escapes visible."""
    return clip(repr(s), 400)


class T:
    """The API suites use."""

    def __init__(self, root):
        self.root = os.path.abspath(root)
        self.group_name = ""
        self.python = sys.executable or "python3"

    # ---- reporting ------------------------------------------------------

    def group(self, name):
        self.group_name = name

    def emit(self, name, status, detail=""):
        fields = [self.group_name, name, status, detail]
        fields = [str(f).replace("\t", " ").replace("\r", " ").replace("\n", " ⏎ ") for f in fields]
        sys.stdout.write("R\t" + "\t".join(fields) + "\n")
        sys.stdout.flush()

    def case(self, name, fn=None):
        """Runs fn() as one case (usable as a decorator: @t.case("name"))."""
        if fn is None:
            return lambda f: self.case(name, f)
        try:
            fn()
            self.emit(name, "OK")
        except Fail as e:
            self.emit(name, "KO", str(e))
        except Timeout as e:
            self.emit(name, "TIMEOUT", str(e))
        except Crash as e:
            self.emit(name, "CRASH", str(e))
        except Skip as e:
            self.emit(name, "SKIP", str(e))
        except Exception:
            self.emit(name, "KO", "erro no teste: " + clip(traceback.format_exc().strip().splitlines()[-1]))
        return fn

    def expect(self, cond, msg):
        if not cond:
            raise Fail(msg)

    def skip(self, msg):
        raise Skip(msg)

    # ---- files ----------------------------------------------------------

    def path(self, rel):
        return os.path.join(self.root, rel)

    def exists(self, rel):
        return os.path.isfile(self.path(rel))

    def need(self, *rels):
        """Fails the case unless every file exists."""
        missing = [r for r in rels if not self.exists(r)]
        if missing:
            raise Fail("arquivo não entregue: " + ", ".join(missing))

    def source(self, rel):
        with open(self.path(rel), encoding="utf-8") as f:
            return f.read()

    # ---- running the student's code -------------------------------------

    def _exec(self, argv, cwd, stdin, timeout):
        env = dict(os.environ, PYTHONDONTWRITEBYTECODE="1", PYTHONIOENCODING="utf-8", PYTHONHASHSEED="0")
        try:
            p = subprocess.run(argv, cwd=cwd, input=stdin, capture_output=True, text=True,
                               timeout=timeout, env=env, errors="replace")
        except subprocess.TimeoutExpired as e:
            out = e.stdout.decode(errors="replace") if isinstance(e.stdout, bytes) else (e.stdout or "")
            return Result(out, "", -9, True)
        return Result(p.stdout, p.stderr, p.returncode, False)

    def run(self, rel, args=(), stdin="", timeout=5, ok_codes=(0,), allow_crash=False):
        """Runs `python3 <rel> args...` in the file's folder. Fails the case
        on a timeout and, unless allow_crash, on an uncaught exception."""
        self.need(rel)
        path = self.path(rel)
        r = self._exec([self.python, path] + [str(a) for a in args], os.path.dirname(path), stdin, timeout)
        return self._check(r, "python3 " + rel + "".join(" " + show(str(a)) for a in args), timeout, allow_crash)

    def py(self, code, ex, stdin="", timeout=5, allow_crash=False, files=()):
        """Runs a snippet of Python with the exercise folder ex as the current
        directory and on sys.path (to import the student's modules)."""
        self.need(*[os.path.join(ex, f) for f in files])
        cwd = self.path(ex)
        if not os.path.isdir(cwd):
            raise Fail("pasta não entregue: " + ex)
        prelude = "import sys\nsys.path.insert(0, %r)\n" % cwd
        r = self._exec([self.python, "-c", prelude + code], cwd, stdin, timeout)
        return self._check(r, "o teste", timeout, allow_crash)

    def _check(self, r, what, timeout, allow_crash):
        if r.timed_out:
            raise Timeout("%s passou de %ss (loop infinito ou esperando input?)" % (what, timeout))
        if r.crashed and not allow_crash:
            raise Crash("%s terminou com erro: %s" % (what, clip(r.last_error())))
        return r

    # ---- static checks --------------------------------------------------

    def tree(self, rel):
        try:
            return ast.parse(self.source(rel), filename=rel)
        except SyntaxError as e:
            raise Fail("erro de sintaxe na linha %s: %s" % (e.lineno, e.msg))

    def authorized(self, rel, allowed=(), imports=(), methods_ok=True):
        """Fails when rel calls a builtin function that isn't in allowed or
        imports a module that isn't in imports. Calls of functions the file
        defines, and method calls (obj.method()), are fine unless
        methods_ok is False."""
        tree = self.tree(rel)
        defined = {n.name for n in ast.walk(tree) if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef))}
        params = {a.arg for n in ast.walk(tree) if isinstance(n, ast.arguments) for a in n.args + n.kwonlyargs}
        assigned = {n.id for n in ast.walk(tree) if isinstance(n, ast.Name) and isinstance(n.ctx, ast.Store)}
        builtins_names = set(dir(__builtins__)) if not isinstance(__builtins__, dict) else set(__builtins__)
        bad = set()
        for n in ast.walk(tree):
            if isinstance(n, ast.Call):
                f = n.func
                if isinstance(f, ast.Name):
                    name = f.id
                    if name in defined or name in params or name in assigned:
                        continue
                    if name in builtins_names and name not in allowed:
                        bad.add(name + "()")
                elif isinstance(f, ast.Attribute) and not methods_ok:
                    bad.add("." + f.attr + "()")
            elif isinstance(n, ast.Import):
                for a in n.names:
                    if a.name.split(".")[0] not in imports:
                        bad.add("import " + a.name)
            elif isinstance(n, ast.ImportFrom):
                if n.module and n.module.split(".")[0] not in imports:
                    bad.add("from " + n.module + " import")
        if bad:
            raise Fail("usa o que não é permitido: " + ", ".join(sorted(bad)))

    def defines(self, rel, *names):
        """Fails unless rel defines these functions/classes at the top level."""
        tree = self.tree(rel)
        top = {n.name for n in tree.body if isinstance(n, (ast.FunctionDef, ast.ClassDef, ast.AsyncFunctionDef))}
        missing = [n for n in names if n not in top]
        if missing:
            raise Fail("não define: " + ", ".join(missing))

    def typed(self, rel):
        """Fails unless every function in rel annotates its parameters and
        return type."""
        bad = []
        for n in ast.walk(self.tree(rel)):
            if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef)):
                args = [a for a in n.args.posonlyargs + n.args.args + n.args.kwonlyargs if a.arg not in ("self", "cls")]
                if n.returns is None or any(a.annotation is None for a in args):
                    bad.append(n.name)
        if bad:
            raise Fail("sem type hints completos: " + ", ".join(bad))

    def tool(self, name, args, cwd=None):
        """Runs a tool (flake8, mypy) when installed; skips the case when not."""
        exe = shutil.which(name)
        if exe is None:
            spec = importlib.util.find_spec(name)
            if spec is None:
                raise Skip("%s não está instalado (pip install %s)" % (name, name))
            argv = [self.python, "-m", name] + list(args)
        else:
            argv = [exe] + list(args)
        return self._exec(argv, cwd or self.root, "", 120)


class Timeout(Exception):
    pass


class Crash(Exception):
    pass


class Skip(Exception):
    pass


def common(t, files, typed=()):
    """The checks every module shares: files delivered, syntax, flake8,
    mypy, and full type hints on the files in typed."""
    t.group("entrega")
    for rel in files:
        t.case(rel, lambda rel=rel: t.need(rel))

    t.group("flake8")

    @t.case("o código segue o flake8")
    def _():
        present = [f for f in files if t.exists(f)]
        if not present:
            t.skip("nenhum arquivo entregue")
        r = t.tool("flake8", present)
        t.expect(r.code == 0, clip(r.out.strip() or r.err.strip(), 1500))

    t.group("mypy")

    @t.case("o mypy não acusa erros")
    def _():
        present = [f for f in files if t.exists(f)]
        if not present:
            t.skip("nenhum arquivo entregue")
        r = t.tool("mypy", ["--ignore-missing-imports", "--no-error-summary", "--cache-dir", "/dev/null"] + present)
        t.expect(r.code == 0, clip(r.out.strip() or r.err.strip(), 1500))

    for rel in typed:
        if t.exists(rel):
            t.case("type hints em todas as funções: " + rel, lambda rel=rel: t.typed(rel))

    t.group("sintaxe")
    for rel in files:
        if t.exists(rel):
            t.case(rel, lambda rel=rel: t.tree(rel))


def main():
    # Suites `from lt import ...`: make that this very module, so their
    # Fail is the one T.case catches.
    sys.modules["lt"] = sys.modules[__name__]
    suite_path, root = sys.argv[1], sys.argv[2]
    spec = importlib.util.spec_from_file_location("suite", suite_path)
    mod = importlib.util.module_from_spec(spec)
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    spec.loader.exec_module(mod)
    mod.suite(T(root))


if __name__ == "__main__":
    main()
