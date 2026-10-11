"""Python Module 05 bis (The Codex): venvs, dependencies, .env configuration,
the alchemy package and its import mysteries."""

import ast
import builtins
import importlib.metadata
import importlib.util
import os
import re
import shutil
import subprocess
import sys
import tempfile

from lt import common, show

try:
    import tomllib
except ImportError:  # Python 3.10
    tomllib = None

CONSTRUCT = "ex0/construct.py"
LOADING = "ex0/loading.py"
REQ0 = "ex0/requirements.txt"
PYPROJECT = "ex0/pyproject.toml"
ORACLE = "ex1/oracle.py"
REQ1 = "ex1/requirements.txt"
ENV_EXAMPLE = "ex1/.env.example"
GITIGNORE = "ex1/.gitignore"

A1 = "ft_alembic_1.py"   # the subject wants a mypy error here, on purpose
INIT = "alchemy/__init__.py"
ELEM = "alchemy/elements.py"
POTIONS = "alchemy/potions.py"
TINIT = "alchemy/transmutation/__init__.py"
RECIPES = "alchemy/transmutation/recipes.py"
GINIT = "alchemy/grimoire/__init__.py"
LIGHT_BOOK = "alchemy/grimoire/light_spellbook.py"
LIGHT_VAL = "alchemy/grimoire/light_validator.py"
DARK_BOOK = "alchemy/grimoire/dark_spellbook.py"
DARK_VAL = "alchemy/grimoire/dark_validator.py"
PACKAGE = [INIT, ELEM, POTIONS, TINIT, RECIPES, GINIT, LIGHT_BOOK, LIGHT_VAL, DARK_BOOK, DARK_VAL]
SCRIPTS = ["ft_alembic_0.py", A1, "ft_distillation_0.py", "ft_transmutation_0.py",
           "ft_kaboom_0.py", "ft_kaboom_1.py"]
PY = [CONSTRUCT, LOADING, ORACLE, "elements.py"] + PACKAGE + SCRIPTS
OTHER = [REQ0, PYPROJECT, REQ1, ENV_EXAMPLE, GITIGNORE]

BUILTINS = set(dir(builtins)) - {"eval", "exec"}
# lt.authorized ignores relative imports; typing is tolerated for annotations.
MODULES = {"alchemy", "elements", "typing"}
VARS = ["MATRIX_MODE", "DATABASE_URL", "API_KEY", "LOG_LEVEL", "ZION_ENDPOINT"]

FIRE = "Fire element created"
WATER = "Water element created"
EARTH = "Earth element created"
AIR = "Air element created"
HEAL = "Healing potion brewed with '%s' and '%s'" % (EARTH, AIR)
STRENGTH = "Strength potion brewed with '%s' and '%s'" % (FIRE, WATER)
GOLD = "Recipe transmuting Lead to Gold: brew '%s' and '%s' mixed with '%s'" % (AIR, STRENGTH, FIRE)
LIGHT = {"earth", "air", "fire", "water"}
DARK = ("bat", "frog", "arsenic", "eyeball")


def norm(s):
    """Lenient on quotes, case and spacing; strict on the words."""
    s = re.sub(r"[\"'`‘’“”]", "", str(s)).lower()
    return " ".join(s.split())


def low(s):
    return s.lower()


def need_pkg(t, *names):
    for n in names:
        if importlib.util.find_spec(n) is None:
            pip = "python-dotenv" if n == "dotenv" else n
            t.skip("%s não está instalado (pip install %s)" % (pip, pip))


# ---- imports (ast) -----------------------------------------------------


def imports(t, rel, top_only=False):
    """[(kind, module, names)] for every import in rel; a relative module
    keeps its dots. top_only skips imports inside functions and classes."""
    out = []

    def visit(node):
        for n in ast.iter_child_nodes(node):
            if top_only and isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
                continue
            if isinstance(n, ast.Import):
                out.extend(("import", a.name, ()) for a in n.names)
            elif isinstance(n, ast.ImportFrom):
                out.append(("from", "." * n.level + (n.module or ""), tuple(a.name for a in n.names)))
            visit(n)

    visit(t.tree(rel))
    return out


def code(imps):
    lines = ["import " + m if k == "import" else "from %s import %s" % (m, ", ".join(ns)) for k, m, ns in imps]
    return show("; ".join(lines) or "nenhum import")


def has(imps, kind, module, name=None):
    return any(k == kind and m == module and (name is None or name in ns) for k, m, ns in imps)


def mentions(imps, name):
    return any(name in m or name in ns for k, m, ns in imps)


def need_import(t, rel, ok, want):
    t.need(rel)
    imps = imports(t, rel)
    t.expect(ok(imps), "esperado `%s`, achei: %s" % (want, code(imps)))
    return imps


def uses_attr(t, rel, obj, attr):
    return any(isinstance(n, ast.Attribute) and n.attr == attr and isinstance(n.value, ast.Name)
               and n.value.id == obj for n in ast.walk(t.tree(rel)))


def static(t, rels, imports_ok=MODULES):
    for rel in rels:
        t.case("só importa o permitido, sem eval/exec: " + rel,
               lambda rel=rel: (t.need(rel), t.authorized(rel, allowed=BUILTINS, imports=imports_ok)))


# ---- running -----------------------------------------------------------


def value(t, body, files=()):
    """Runs body from the project root; it calls done(x) and x comes back."""
    pre = "import contextlib, io\n\ndef done(x):\n    print('@@' + repr(x))\n\n"
    r = t.py(pre + body, ex=".", files=files)
    for line in reversed(r.out.splitlines()):
        if line.startswith("@@"):
            return ast.literal_eval(line[2:])
    t.expect(False, "o teste não produziu resultado (saída: %s)" % show(r.out[-200:]))


def call(t, module, func, *args):
    """module.func(*args) imported from the root, its prints silenced."""
    body = ("with contextlib.redirect_stdout(io.StringIO()):\n"
            "    from %s import %s as f\n"
            "    r = f(*%r)\n"
            "done(r)\n" % (module, func, args))
    return value(t, body)


def same(t, got, want, what):
    t.expect(isinstance(got, str), "%s deveria retornar uma str, retornou %s" % (what, show(got)))
    t.expect(norm(got) == norm(want), "%s: esperado %s, veio %s" % (what, show(want), show(got)))


def run_shows(t, rel, *wants, allow_crash=False):
    r = t.run(rel, allow_crash=allow_crash)
    for w in wants:
        t.expect(norm(w) in norm(r.out), "esperado %s na saída de %s, veio %s" % (show(w), rel, show(r.out)))
    return r


def as_main(t, ex, rel, setup):
    """Runs rel (inside ex) as __main__ after the Python code in setup."""
    t.need(rel)
    path = t.path(rel)
    body = "import os, runpy, sys\n%s\nsys.argv = [%r]\nrunpy.run_path(%r, run_name='__main__')\n" % (
        setup, path, path)
    return t.py(body, ex=ex)


def no_hardcode(t, rel, *words):
    t.need(rel)
    src = low(t.source(rel))
    found = [w for w in words if w in src]
    t.expect(not found, "%s escreve %s direto no código: deveria chamar as funções importadas"
             % (rel, show(found)))


# ---- general rules -----------------------------------------------------

MUTATORS = {"append", "insert", "extend", "remove", "pop", "clear", "__setitem__", "__iadd__"}


def is_sys_path(n):
    return (isinstance(n, ast.Attribute) and n.attr == "path" and isinstance(n.value, ast.Name)
            and n.value.id == "sys")


def touches_sys_path(tree):
    for n in ast.walk(tree):
        if isinstance(n, ast.Call) and isinstance(n.func, ast.Attribute) and n.func.attr in MUTATORS \
                and is_sys_path(n.func.value):
            return True
        if isinstance(n, (ast.Assign, ast.AugAssign, ast.AnnAssign)):
            targets = n.targets if isinstance(n, ast.Assign) else [n.target]
            for tg in targets:
                if is_sys_path(tg) or (isinstance(tg, ast.Subscript) and is_sys_path(tg.value)):
                    return True
        if isinstance(n, ast.Delete) and any(isinstance(tg, ast.Subscript) and is_sys_path(tg.value)
                                             for tg in n.targets):
            return True
    return False


def git(args, cwd):
    if shutil.which("git") is None:
        return None
    return subprocess.run(["git", "-c", "core.excludesFile=/dev/null"] + args, cwd=cwd,
                          capture_output=True, text=True)


def rules(t):
    t.group("regras gerais")

    @t.case("ninguém modifica sys.path")
    def _():
        bad = [rel for rel in PY if t.exists(rel) and touches_sys_path(t.tree(rel))]
        t.expect(not bad, "o subject proíbe modificar sys.path: %s" % show(bad))

    def tracked():
        r = git(["rev-parse", "--is-inside-work-tree"], t.root)
        if r is None:
            t.skip("git não está instalado")
        if r.returncode != 0:
            t.skip("a pasta não é um repositório git")
        return git(["ls-files"], t.root).stdout.splitlines()

    @t.case("nenhum .env versionado no git")
    def _():
        bad = [f for f in tracked() if os.path.basename(f) == ".env"]
        t.expect(not bad, "o .env tem segredos e não pode ir para o repositório: %s" % show(bad))

    @t.case("nenhum ambiente virtual versionado no git")
    def _():
        bad = sorted({os.path.dirname(f) or "." for f in tracked() if os.path.basename(f) == "pyvenv.cfg"})
        t.expect(not bad, "o venv não deve ir para o repositório: %s" % show(bad))


# ---- requirements / pyproject -------------------------------------------

REQ_LINE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]*(\[[^\]]*\])?\s*"
                      r"((===|==|>=|<=|~=|!=|>|<)\s*[\w.*+!-]+\s*(,\s*(===|==|>=|<=|~=|!=|>|<)\s*[\w.*+!-]+\s*)*)?"
                      r"(;.*)?$")


def pkg(name):
    return re.sub(r"[-_.]+", "-", name).lower()


def req_names(t, rel):
    t.need(rel)
    names, bad = set(), []
    for line in t.source(rel).splitlines():
        line = re.sub(r"(^|\s)#.*", "", line).strip()
        if not line or line.startswith("-"):
            continue
        if not REQ_LINE.match(line):
            bad.append(line)
        else:
            names.add(pkg(re.match(r"[A-Za-z0-9._-]+", line).group()))
    t.expect(not bad, "%s: linha que o pip não entende: %s" % (rel, show(bad[:3])))
    return names


def toml(t):
    t.need(PYPROJECT)
    if tomllib is None:
        t.skip("tomllib precisa do Python 3.11+")
    try:
        return tomllib.loads(t.source(PYPROJECT))
    except tomllib.TOMLDecodeError as e:
        t.expect(False, "pyproject.toml não é um TOML válido: %s" % e)


def poetry_deps(data):
    poetry = data.get("tool", {}).get("poetry", {})
    deps = {pkg(k) for k in poetry.get("dependencies", {})}
    for group in poetry.get("group", {}).values():
        deps |= {pkg(k) for k in group.get("dependencies", {})}
    for spec in data.get("project", {}).get("dependencies", []):
        m = re.match(r"\s*([A-Za-z0-9._-]+)", spec)
        if m:
            deps.add(pkg(m.group(1)))
    return deps


# ---- ex0: construct ----------------------------------------------------

FAKE_VENV = "/lt_lab/lt_sealed_env"
FAKE_PY = FAKE_VENV + "/bin/python"

CONSTRUCT_SETUP = """
import site
venv = %r
for k in ("VIRTUAL_ENV", "CONDA_PREFIX", "CONDA_DEFAULT_ENV", "POETRY_ACTIVE", "PIPENV_ACTIVE"):
    os.environ.pop(k, None)
if hasattr(sys, "real_prefix"):
    del sys.real_prefix
if venv == "REAL":
    if sys.prefix != sys.base_prefix:
        os.environ["VIRTUAL_ENV"] = sys.prefix
elif venv:
    sys.prefix = sys.exec_prefix = venv
    sys.executable = venv + "/bin/python"
    os.environ["VIRTUAL_ENV"] = venv
else:
    sys.prefix, sys.exec_prefix = sys.base_prefix, sys.base_exec_prefix
    sys.executable = getattr(sys, "_base_executable", None) or sys.executable
site.PREFIXES[:] = [sys.prefix, sys.exec_prefix]
print("@@" + sys.executable, file=sys.stderr)
"""


def construct(t, venv):
    r = as_main(t, "ex0", CONSTRUCT, CONSTRUCT_SETUP % venv)
    exe = [line[2:] for line in r.err.splitlines() if line.startswith("@@")][0]
    r.err = "\n".join(line for line in r.err.splitlines() if not line.startswith("@@"))
    return r, exe


def venv_lines(out, exe):
    """The output without the Python path and the site-packages lines (they
    contain the venv path even when the venv isn't recognised)."""
    return "\n".join(line.replace(exe, "") for line in out.splitlines()
                     if "site-packages" not in line and "dist-packages" not in line)


def shows_python(t, out, exe):
    t.expect(exe in out or os.path.realpath(exe) in out,
             "esperado o Python atual (%s) na saída, veio %s" % (show(exe), show(out)))


def instructions(t, out):
    t.expect(re.search(r"-m\s+venv", out) and "activate" in low(out),
             "fora de um venv, esperado as instruções 'python3 -m venv ...' e 'source .../bin/activate', veio %s"
             % show(out))


def ex0_construct(t):
    t.group("ex0 construct")
    t.case("só usa os, sys, site, importlib (sem eval/exec)", lambda: t.authorized(
        CONSTRUCT, allowed=BUILTINS, imports={"os", "sys", "site", "importlib", "pandas", "numpy", "typing"}))

    @t.case("fora de um venv: 'None detected' e instruções para criar/ativar")
    def _():
        r, exe = construct(t, "")
        t.expect(r.out.strip(), "não imprimiu nada")
        instructions(t, r.out)

    @t.case("fora de um venv: mostra o Python atual")
    def _():
        r, exe = construct(t, "")
        shows_python(t, r.out, exe)

    @t.case("dentro de um venv: mostra o nome do venv")
    def _():
        r, exe = construct(t, FAKE_VENV)
        t.expect("lt_sealed_env" in venv_lines(r.out, exe), "com o venv %s ativo, esperado o nome dele "
                 "('lt_sealed_env') na saída, veio %s" % (show(FAKE_VENV), show(r.out)))

    @t.case("dentro de um venv: mostra o caminho do venv e o Python dele")
    def _():
        r, exe = construct(t, FAKE_VENV)
        t.expect(FAKE_VENV in venv_lines(r.out, exe),
                 "esperado o caminho do venv %s na saída, veio %s" % (show(FAKE_VENV), show(r.out)))
        shows_python(t, r.out, exe)

    @t.case("dentro de um venv: mostra onde os pacotes são instalados (site-packages)")
    def _():
        r, exe = construct(t, FAKE_VENV)
        t.expect("site-packages" in r.out or "dist-packages" in r.out,
                 "esperado o caminho do site-packages do venv na saída, veio %s" % show(r.out))

    @t.case("dentro de um venv: não manda criar outro venv")
    def _():
        r, exe = construct(t, FAKE_VENV)
        t.expect(not re.search(r"-m\s+venv", r.out), "o venv já está ativo, mas a saída manda criar um: %s"
                 % show(r.out))

    @t.case("saída claramente diferente dentro e fora do venv")
    def _():
        out_open = construct(t, "")[0].out
        out_sealed = construct(t, FAKE_VENV)[0].out
        t.expect(norm(out_open) != norm(out_sealed), "a saída é a mesma nos dois casos: %s" % show(out_open))

    @t.case("rodando de verdade: reconhece o ambiente atual")
    def _():
        r = t.run(CONSTRUCT)
        if sys.prefix != sys.base_prefix:
            r = construct(t, "REAL")[0]
            name = os.path.basename(sys.prefix)
            t.expect(name in r.out, "rodando no venv %s, esperado o nome dele na saída, veio %s"
                     % (show(name), show(r.out)))
        else:
            instructions(t, r.out)


# ---- ex0: loading ------------------------------------------------------

STUB = '''__version__ = "%s"


class _Any:
    def __getattr__(self, name):
        return _Any()

    def __call__(self, *a, **k):
        return _Any()

    def __getitem__(self, k):
        return _Any()

    def __iter__(self):
        return iter(())

    def __len__(self):
        return 0

    def __str__(self):
        return "0"

    __repr__ = __str__

    def __format__(self, spec):
        return "0"

    def __int__(self):
        return 0

    def __float__(self):
        return 0.0

    def _op(self, *a):
        return self

    __add__ = __radd__ = __sub__ = __rsub__ = __mul__ = __rmul__ = _op
    __truediv__ = __rtruediv__ = __floordiv__ = __pow__ = __mod__ = __neg__ = _op


def __getattr__(name):
    return _Any()
'''
VERSIONS = {"pandas": "7.7.7", "numpy": "8.8.8"}

LOADING_SETUP = """
import importlib
sys.path[:] = [p for p in sys.path if "site-packages" not in p and "dist-packages" not in p]
if %r:
    sys.path.append(%r)
sys.path_importer_cache.clear()
importlib.invalidate_caches()
"""


def loading(t, *present):
    """Runs loading.py with only the fake reagents in present available
    (site-packages hidden: the real ones are out of reach)."""
    stubs = tempfile.mkdtemp(prefix="lt_reagents_")
    try:
        for name in present:
            os.makedirs(os.path.join(stubs, name))
            with open(os.path.join(stubs, name, "__init__.py"), "w") as f:
                f.write(STUB % VERSIONS[name])
            info = os.path.join(stubs, "%s-%s.dist-info" % (name, VERSIONS[name]))
            os.makedirs(info)
            with open(os.path.join(info, "METADATA"), "w") as f:
                f.write("Metadata-Version: 2.1\nName: %s\nVersion: %s\n" % (name, VERSIONS[name]))
        return as_main(t, "ex0", LOADING, LOADING_SETUP % (bool(present), stubs))
    finally:
        shutil.rmtree(stubs, ignore_errors=True)


def reported_ok(out, name):
    o = low(out)
    return "[ok] " + name in o or re.search(name + r"\W{0,3}\(?v?\d+\.\d+", o) is not None


def install_help(t, both, names):
    lo = low(both)
    t.expect("pip install" in lo, "esperado o comando 'pip install ...' para o que falta, veio %s" % show(both))
    t.expect(re.search(r"poetry (add|install)", lo), "esperado o comando 'poetry add ...' / 'poetry install', veio %s"
             % show(both))
    for n in names:
        t.expect(n in lo, "esperado o nome do reagente que falta (%s) na mensagem, veio %s" % (n, show(both)))


def ex0_loading(t):
    t.group("ex0 loading")
    t.case("só usa os, sys, site, importlib, pandas, numpy (sem eval/exec)", lambda: t.authorized(
        LOADING, allowed=BUILTINS, imports={"os", "sys", "site", "importlib", "pandas", "numpy", "typing"}))

    @t.case("verifica pandas e numpy")
    def _():
        t.need(LOADING)
        src = t.source(LOADING)
        t.expect("pandas" in src and "numpy" in src, "loading.py deveria importar pandas e numpy")

    @t.case("sem nenhum reagente: não quebra e avisa o que falta")
    def _():
        r = loading(t)
        both = r.out + r.err
        t.expect(not reported_ok(r.out, "pandas") and not reported_ok(r.out, "numpy"),
                 "pandas/numpy não estão disponíveis, mas a saída diz que estão: %s" % show(r.out))
        install_help(t, both, ["pandas", "numpy"])

    @t.case("só o pandas faltando: mostra a versão do numpy e o comando para o pandas")
    def _():
        r = loading(t, "numpy")
        both = r.out + r.err
        t.expect("8.8.8" in r.out, "o numpy (versão 8.8.8 neste teste) está disponível: esperado a versão na saída, "
                 "veio %s" % show(r.out))
        t.expect(not reported_ok(r.out, "pandas"), "o pandas não está disponível, mas a saída diz que está: %s"
                 % show(r.out))
        install_help(t, both, ["pandas"])

    @t.case("com os dois reagentes: mostra a versão de cada um")
    def _():
        r = loading(t, "pandas", "numpy")
        for name, ver in VERSIONS.items():
            t.expect(ver in r.out, "esperado %s (%s neste teste) na saída, veio %s" % (name, ver, show(r.out)))

    @t.case("explica a diferença entre pip e Poetry")
    def _():
        r = loading(t, "pandas", "numpy")
        lo = low(r.out)
        t.expect("pip" in lo and "poetry" in lo and any(w in lo for w in ("requirements", "pyproject", "lock")),
                 "esperado na saída a diferença entre pip (requirements.txt) e Poetry (pyproject.toml/poetry.lock), "
                 "veio %s" % show(r.out))

    @t.case("com os pacotes reais instalados: mostra as versões instaladas")
    def _():
        need_pkg(t, "pandas", "numpy")
        r = t.run(LOADING, timeout=30)
        for name in ("pandas", "numpy"):
            ver = importlib.metadata.version(name)
            t.expect(ver in r.out, "esperado %s %s na saída, veio %s" % (name, ver, show(r.out)))

    @t.case("rodando no ambiente atual: não quebra")
    def _():
        t.run(LOADING, timeout=30)


def ex0_deps(t):
    t.group("ex0 requirements.txt / pyproject.toml")

    @t.case("requirements.txt declara pandas e numpy, no formato do pip")
    def _():
        names = req_names(t, REQ0)
        miss = [n for n in ("pandas", "numpy") if n not in names]
        t.expect(not miss, "faltou declarar %s em %s (tem %s)" % (show(miss), REQ0, show(sorted(names))))

    t.case("pyproject.toml é um TOML válido", lambda: toml(t))

    @t.case("pyproject.toml tem o nome do projeto ([tool.poetry] ou [project])")
    def _():
        data = toml(t)
        name = data.get("tool", {}).get("poetry", {}).get("name") or data.get("project", {}).get("name")
        t.expect(name, "esperado name = \"...\" em [tool.poetry] (ou [project])")

    @t.case("pyproject.toml declara pandas e numpy para o Poetry")
    def _():
        deps = poetry_deps(toml(t))
        miss = [n for n in ("pandas", "numpy") if n not in deps]
        t.expect(not miss, "faltou %s em [tool.poetry.dependencies] (ou [project] dependencies); achei %s"
                 % (show(miss), show(sorted(deps))))

    @t.case("pyproject.toml declara a versão do Python")
    def _():
        data = toml(t)
        ok = "python" in data.get("tool", {}).get("poetry", {}).get("dependencies", {}) \
            or "requires-python" in data.get("project", {})
        t.expect(ok, "esperado python = \"^3.10\" em [tool.poetry.dependencies] (ou requires-python em [project])")


# ---- ex1: oracle -------------------------------------------------------

ORACLE_SETUP = """
import shutil, tempfile
src = %r
tmp = tempfile.mkdtemp(prefix="lt_oracle_")
for n in os.listdir(src):
    if n != ".env" and os.path.isfile(os.path.join(src, n)):
        shutil.copy(os.path.join(src, n), tmp)
dotenv = %r
if dotenv is not None:
    with open(os.path.join(tmp, ".env"), "w") as f:
        f.write(dotenv)
for k in %r:
    os.environ.pop(k, None)
os.environ.update(%r)
os.chdir(tmp)
import atexit
atexit.register(shutil.rmtree, tmp, True)
"""

DEV = {"MATRIX_MODE": "development", "DATABASE_URL": "postgresql://localhost:5432/zion",
       "API_KEY": "fake-api-key-for-tests", "LOG_LEVEL": "DEBUG", "ZION_ENDPOINT": "https://zion.example.com/api"}
WARN = ("missing", "not set", "warn", "default", "not found", "not configured", "unset", "no .env", "not defined")


def dotenv_text(d):
    return "".join("%s=%s\n" % kv for kv in d.items())


def oracle_main(t, env=None, dotenv=None):
    """oracle.py run from a copy of ex1 (without the student's own .env),
    with only env set and dotenv as its .env file."""
    need_pkg(t, "dotenv")
    t.need(ORACLE)
    setup = ORACLE_SETUP % (t.path("ex1"), dotenv, VARS, env or {})
    body = ("import os, runpy, sys\n%s\npath = os.path.join(tmp, 'oracle.py')\nsys.argv = [path]\n"
            "runpy.run_path(path, run_name='__main__')\n" % setup)
    return t.py(body, ex="ex1")


def mode_counts(out):
    lo = low(out)
    return lo.count("production"), lo.count("development")


def secrets_in_code(tree):
    bad = []
    for n in ast.walk(tree):
        if isinstance(n, (ast.Assign, ast.AnnAssign)):
            targets = n.targets if isinstance(n, ast.Assign) else [n.target]
            v = n.value
            if isinstance(v, ast.Constant) and isinstance(v.value, str) and v.value.strip() \
                    and not re.fullmatch(r"[A-Z_][A-Z0-9_]*", v.value):
                for tg in targets:
                    if isinstance(tg, ast.Name) and re.search(r"KEY|SECRET|PASSWORD|PASSWD|TOKEN", tg.id.upper()):
                        bad.append("%s = %r" % (tg.id, v.value))
        if isinstance(n, ast.Constant) and isinstance(n.value, str) \
                and re.search(r"://[^/\s:@]+:[^/\s@]+@", n.value):
            bad.append(repr(n.value))
    return bad


def ignored(text, name):
    with tempfile.TemporaryDirectory() as d:
        with open(os.path.join(d, ".gitignore"), "w") as f:
            f.write(text)
        git(["init", "-q"], d)
        return git(["check-ignore", "-q", name], d).returncode == 0


def ex1_static(t):
    t.group("ex1 configuração")
    t.case("só usa os, sys, dotenv (sem eval/exec)", lambda: t.authorized(
        ORACLE, allowed=BUILTINS, imports={"os", "sys", "dotenv", "typing"}))

    t.case("usa o python-dotenv", lambda: need_import(
        t, ORACLE, lambda i: mentions(i, "dotenv"), "from dotenv import load_dotenv"))

    @t.case("lê as 5 variáveis do subject")
    def _():
        t.need(ORACLE)
        src = t.source(ORACLE)
        miss = [v for v in VARS if v not in src]
        t.expect(not miss, "oracle.py não usa %s" % show(miss))

    @t.case("nenhum segredo escrito no código")
    def _():
        t.need(ORACLE)
        bad = secrets_in_code(t.tree(ORACLE))
        t.expect(not bad, "segredo fixo no código (deveria vir do ambiente/.env): %s" % show(bad))

    @t.case("requirements.txt declara python-dotenv")
    def _():
        names = req_names(t, REQ1)
        t.expect("python-dotenv" in names, "faltou python-dotenv em %s (tem %s)" % (REQ1, show(sorted(names))))

    @t.case(".env.example tem as 5 variáveis, no formato CHAVE=valor")
    def _():
        t.need(ENV_EXAMPLE)
        keys, bad = set(), []
        for line in t.source(ENV_EXAMPLE).splitlines():
            s = line.strip()
            if not s or s.startswith("#"):
                continue
            m = re.match(r"(export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=", s)
            if m:
                keys.add(m.group(2))
            else:
                bad.append(s)
        t.expect(not bad, "linha que não é CHAVE=valor no .env.example: %s" % show(bad[:3]))
        miss = [v for v in VARS if v not in keys]
        t.expect(not miss, "faltou %s no .env.example" % show(miss))

    def gitignore():
        t.need(GITIGNORE)
        if shutil.which("git") is None:
            t.skip("git não está instalado")
        return t.source(GITIGNORE)

    t.case(".gitignore ignora o .env", lambda: t.expect(
        ignored(gitignore(), ".env"), "o .env não é ignorado pelo %s: os segredos iriam para o git" % GITIGNORE))
    t.case(".gitignore não ignora o .env.example", lambda: t.expect(
        not ignored(gitignore(), ".env.example"), "o .env.example é o modelo que vai para o git, mas %s ignora ele"
        % GITIGNORE))


def ex1_oracle(t):
    t.group("ex1 oracle")

    @t.case("sem configuração nenhuma: não quebra e avisa")
    def _():
        r = oracle_main(t)
        both = low(r.out + r.err)
        t.expect(r.out.strip(), "não imprimiu nada")
        t.expect(any(w in both for w in WARN), "sem .env e sem variáveis, esperado um aviso de configuração faltando, "
                 "veio %s" % show(r.out + r.err))

    @t.case("só MATRIX_MODE definido: não quebra e avisa do resto")
    def _():
        r = oracle_main(t, {"MATRIX_MODE": "development"})
        both = low(r.out + r.err)
        t.expect(any(w in both for w in WARN), "faltando 4 variáveis, esperado um aviso, veio %s" % show(r.out + r.err))

    @t.case("carrega a configuração do .env")
    def _():
        r = oracle_main(t, dotenv=dotenv_text(dict(DEV, LOG_LEVEL="CRITICAL")))
        t.expect("CRITICAL" in r.out, "o .env tem LOG_LEVEL=CRITICAL, esperado isso na saída, veio %s" % show(r.out))

    @t.case("o .env.example copiado como .env funciona")
    def _():
        t.need(ENV_EXAMPLE)
        oracle_main(t, dotenv=t.source(ENV_EXAMPLE))

    @t.case("carrega a configuração das variáveis de ambiente")
    def _():
        r = oracle_main(t, dict(DEV, LOG_LEVEL="CRITICAL"))
        t.expect("CRITICAL" in r.out, "com LOG_LEVEL=CRITICAL no ambiente, esperado isso na saída, veio %s"
                 % show(r.out))

    @t.case("o ambiente tem prioridade sobre o .env (LOG_LEVEL)")
    def _():
        r = oracle_main(t, {"LOG_LEVEL": "CRITICAL"}, dotenv_text(dict(DEV, LOG_LEVEL="DEBUG")))
        t.expect("CRITICAL" in r.out, ".env com LOG_LEVEL=DEBUG e ambiente com LOG_LEVEL=CRITICAL: esperado CRITICAL, "
                 "veio %s" % show(r.out))

    @t.case("development e production: diferença visível na saída")
    def _():
        dev = oracle_main(t, DEV).out
        prod = oracle_main(t, dict(DEV, MATRIX_MODE="production")).out
        t.expect(norm(dev) != norm(prod), "a saída é a mesma em development e production: %s" % show(dev))
        t.expect(mode_counts(prod)[0] > mode_counts(dev)[0] and mode_counts(dev)[1] > mode_counts(prod)[1],
                 "esperado o modo atual na saída: development=%s / production=%s" % (show(dev), show(prod)))

    @t.case("MATRIX_MODE=production no ambiente vence o .env (exemplo do subject)")
    def _():
        text = dotenv_text(DEV)
        base = oracle_main(t, dotenv=text).out
        r = oracle_main(t, {"MATRIX_MODE": "production", "API_KEY": "secret123"}, text)
        t.expect(mode_counts(r.out)[0] > mode_counts(base)[0],
                 ".env em development e ambiente com MATRIX_MODE=production: esperado o modo production, veio %s"
                 % show(r.out))

    @t.case("o API_KEY vem da configuração (muda a saída quando falta)")
    def _():
        full = oracle_main(t, DEV).out
        rest = dict(DEV)
        del rest["API_KEY"]
        without = oracle_main(t, rest).out
        t.expect(norm(full) != norm(without), "a saída é igual com e sem API_KEY: o segredo está fixo no código? %s"
                 % show(full))

    @t.case("não mostra o valor do API_KEY na saída")
    def _():
        r = oracle_main(t, dotenv=dotenv_text(DEV))
        r2 = oracle_main(t, {"MATRIX_MODE": "production", "API_KEY": "secret123"})
        for res, key in ((r, DEV["API_KEY"]), (r2, "secret123")):
            t.expect(key not in res.out + res.err, "o segredo %s apareceu na saída" % show(key))

    @t.case("production sem API_KEY: não quebra")
    def _():
        oracle_main(t, {"MATRIX_MODE": "production"})

    @t.case("MATRIX_MODE inválido: não quebra")
    def _():
        oracle_main(t, dict(DEV, MATRIX_MODE="banana"))

    @t.case(".env com valores vazios: não quebra")
    def _():
        oracle_main(t, dotenv="".join("%s=\n" % v for v in VARS))


# ---- ex2: alembic ------------------------------------------------------


def ex2(t):
    t.group("ex2 alembic")
    a0 = "ft_alembic_0.py"

    t.case("elements.py define create_fire e create_water",
           lambda: (t.need("elements.py"), t.defines("elements.py", "create_fire", "create_water")))
    t.case("alchemy/elements.py define create_earth e create_air",
           lambda: (t.need(ELEM), t.defines(ELEM, "create_earth", "create_air")))
    t.case("create_fire() retorna 'Fire element created'",
           lambda: same(t, call(t, "elements", "create_fire"), FIRE, "create_fire()"))
    t.case("create_water() retorna 'Water element created'",
           lambda: same(t, call(t, "elements", "create_water"), WATER, "create_water()"))
    t.case("alchemy.elements.create_earth() retorna 'Earth element created'",
           lambda: same(t, call(t, "alchemy.elements", "create_earth"), EARTH, "create_earth()"))
    t.case("alchemy.elements.create_air() retorna 'Air element created'",
           lambda: same(t, call(t, "alchemy.elements", "create_air"), AIR, "create_air()"))

    names = ("with contextlib.redirect_stdout(io.StringIO()):\n"
             "    import alchemy\n"
             "done(sorted(n for n in dir(alchemy) if not n.startswith('__')))\n")

    @t.case("alchemy expõe create_air")
    def _():
        got = value(t, names, files=(INIT,))
        t.expect("create_air" in got, "esperado alchemy.create_air, alchemy tem %s" % show(got))
        same(t, call(t, "alchemy", "create_air"), AIR, "alchemy.create_air()")

    @t.case("alchemy NÃO expõe create_earth")
    def _():
        got = value(t, names, files=(INIT,))
        t.expect("create_earth" not in got, "o subject quer create_earth fora de `import alchemy`, "
                 "mas alchemy tem %s" % show(got))

    @t.case("`from alchemy import create_earth` dá ImportError")
    def _():
        t.need(INIT)
        r = t.py("from alchemy import create_earth", ex=".", allow_crash=True)
        t.expect("ImportError" in r.err, "esperado ImportError, veio %s" % show(r.err[-200:] or r.out))

    @t.case("`import alchemy` não carrega o grimório sombrio (não explode)")
    def _():
        body = ("import sys\nwith contextlib.redirect_stdout(io.StringIO()):\n"
                "    import alchemy\n"
                "done([m for m in sys.modules if m.startswith('alchemy.grimoire.dark')])\n")
        got = value(t, body, files=(INIT,))
        t.expect(not got, "`import alchemy` já importa %s" % show(got))

    @t.case("ft_alembic_0 usa `import elements` (sem from)")
    def _():
        imps = need_import(t, a0, lambda i: has(i, "import", "elements"), "import elements")
        t.expect(not has(imps, "from", "elements"), "o subject pede `import ...`, não `from ... import`")

    t.case("ft_alembic_0 cria o fogo", lambda: run_shows(t, a0, FIRE))

    @t.case("ft_alembic_1 usa só `import alchemy`")
    def _():
        imps = need_import(t, A1, lambda i: has(i, "import", "alchemy"), "import alchemy")
        deeper = [i for i in imps if i[1].startswith("alchemy.") or (i[0] == "from" and i[1] == "alchemy")]
        t.expect(not deeper, "o subject pede só `import alchemy`, mas também tem %s" % code(deeper))

    @t.case("ft_alembic_1 tenta chamar alchemy.create_earth()")
    def _():
        t.need(A1)
        t.expect(uses_attr(t, A1, "alchemy", "create_earth"),
                 "esperado uma chamada a alchemy.create_earth() para mostrar que ela não é exposta")

    @t.case("ft_alembic_1 cria o ar e create_earth dá AttributeError")
    def _():
        r = run_shows(t, A1, AIR, allow_crash=True)
        both = r.out + r.err
        t.expect("create_earth" in both and ("AttributeError" in both or "has no attribute" in both),
                 "esperado o AttributeError de create_earth (no stderr ou capturado), veio %s / %s"
                 % (show(r.out), show(r.err[-300:])))
        t.expect(norm(EARTH) not in norm(r.out), "create_earth() não deveria funcionar via alchemy: %s"
                 % show(r.out))

    static(t, ["elements.py", ELEM, INIT, a0, A1])


# ---- ex3: distillation and transmutation -------------------------------


def ex3(t):
    t.group("ex3 distillation")
    d0, t0 = "ft_distillation_0.py", "ft_transmutation_0.py"

    t.case("potions.py define healing_potion e strength_potion",
           lambda: (t.need(POTIONS), t.defines(POTIONS, "healing_potion", "strength_potion")))
    t.case("healing_potion() usa terra e ar",
           lambda: same(t, call(t, "alchemy.potions", "healing_potion"), HEAL, "healing_potion()"))
    t.case("strength_potion() usa fogo e água",
           lambda: same(t, call(t, "alchemy.potions", "strength_potion"), STRENGTH, "strength_potion()"))
    t.case("potions.py chama as funções dos elementos (não copia as strings)",
           lambda: no_hardcode(t, POTIONS, "element created"))

    @t.case("potions.py busca fogo/água no elements.py da raiz")
    def _():
        body = ("import elements\n"
                "elements.create_fire = lambda: 'LT_FIRE'\n"
                "elements.create_water = lambda: 'LT_WATER'\n"
                "with contextlib.redirect_stdout(io.StringIO()):\n"
                "    import alchemy.potions as p\n"
                "    r = p.strength_potion()\n"
                "done(r)\n")
        got = value(t, body, files=("elements.py", POTIONS))
        t.expect("LT_FIRE" in got and "LT_WATER" in got,
                 "strength_potion() deveria chamar create_fire/create_water do elements.py, veio %s" % show(got))

    @t.case("potions.py busca terra/ar em alchemy/elements.py")
    def _():
        body = ("import alchemy.elements as e\n"
                "e.create_earth = lambda: 'LT_EARTH'\n"
                "e.create_air = lambda: 'LT_AIR'\n"
                "with contextlib.redirect_stdout(io.StringIO()):\n"
                "    import alchemy.potions as p\n"
                "    r = p.healing_potion()\n"
                "done(r)\n")
        got = value(t, body, files=(ELEM, POTIONS))
        t.expect("LT_EARTH" in got and ("LT_AIR" in got or norm(AIR) in norm(got)),
                 "healing_potion() deveria chamar create_earth/create_air de alchemy/elements.py, veio %s" % show(got))

    t.case("ft_distillation_0 usa `from alchemy.potions import ...`",
           lambda: need_import(t, d0, lambda i: has(i, "from", "alchemy.potions")
                               or has(i, "from", "alchemy", "potions"),
                               "from alchemy.potions import healing_potion, strength_potion"))
    t.case("ft_distillation_0 prepara as duas poções", lambda: run_shows(t, d0, STRENGTH, HEAL))
    static(t, [POTIONS, d0])

    t.group("ex3 transmutation")
    t.case("alchemy/transmutation é um pacote (tem __init__.py)", lambda: t.need(TINIT))
    t.case("recipes.py define lead_to_gold", lambda: (t.need(RECIPES), t.defines(RECIPES, "lead_to_gold")))
    t.case("lead_to_gold() retorna a receita completa",
           lambda: same(t, call(t, "alchemy.transmutation.recipes", "lead_to_gold"), GOLD, "lead_to_gold()"))
    t.case("recipes.py chama as funções (não copia as strings)",
           lambda: no_hardcode(t, RECIPES, "element created", "potion brewed"))

    @t.case("lead_to_gold() usa strength_potion de potions.py")
    def _():
        body = ("import alchemy.potions as p\n"
                "p.strength_potion = lambda: 'LT_POTION'\n"
                "with contextlib.redirect_stdout(io.StringIO()):\n"
                "    import alchemy.transmutation.recipes as m\n"
                "    r = m.lead_to_gold()\n"
                "done(r)\n")
        got = value(t, body, files=(POTIONS, RECIPES))
        t.expect("LT_POTION" in got, "lead_to_gold() deveria chamar strength_potion() de alchemy/potions.py, veio %s"
                 % show(got))

    t.case("recipes.py tem pelo menos um import relativo",
           lambda: need_import(t, RECIPES, lambda i: any(m.startswith(".") for k, m, ns in i),
                               "from ..potions import strength_potion (com ponto)"))
    t.case("recipes.py tem pelo menos um import absoluto",
           lambda: need_import(t, RECIPES, lambda i: any(not m.startswith(".") and m.split(".")[0] in
                                                         ("alchemy", "elements") for k, m, ns in i),
                               "from alchemy.elements import create_air (sem ponto)"))
    t.case("ft_transmutation_0 usa `import alchemy.transmutation.recipes`",
           lambda: need_import(t, t0, lambda i: has(i, "import", "alchemy.transmutation.recipes"),
                               "import alchemy.transmutation.recipes"))
    t.case("ft_transmutation_0 transforma chumbo em ouro", lambda: run_shows(t, t0, GOLD))
    static(t, [TINIT, RECIPES, t0])


# ---- ex4: kaboom -------------------------------------------------------


def ex4(t):
    t.group("ex4 kaboom")
    k0, k1 = "ft_kaboom_0.py", "ft_kaboom_1.py"

    t.case("alchemy/grimoire é um pacote (tem __init__.py)", lambda: t.need(GINIT))
    t.case("light_spellbook.py define light_spell_allowed_ingredients e light_spell_record",
           lambda: (t.need(LIGHT_BOOK), t.defines(LIGHT_BOOK, "light_spell_allowed_ingredients",
                                                  "light_spell_record")))
    t.case("light_validator.py define validate_ingredients",
           lambda: (t.need(LIGHT_VAL), t.defines(LIGHT_VAL, "validate_ingredients")))
    t.case("dark_spellbook.py define dark_spell_allowed_ingredients e dark_spell_record",
           lambda: (t.need(DARK_BOOK), t.defines(DARK_BOOK, "dark_spell_allowed_ingredients", "dark_spell_record")))

    @t.case("dark_validator.py define a função de validação")
    def _():
        t.need(DARK_VAL)
        top = [n.name for n in t.tree(DARK_VAL).body if isinstance(n, ast.FunctionDef)]
        t.expect(any("validat" in n for n in top), "esperado uma função dark_validate_..., achei %s" % show(top))

    @t.case("ingredientes da luz: earth, air, fire, water")
    def _():
        got = call(t, "alchemy.grimoire.light_spellbook", "light_spell_allowed_ingredients")
        t.expect(isinstance(got, list), "esperado uma list, veio %s" % show(got))
        lows = {str(x).lower() for x in got}
        t.expect(LIGHT <= lows, "esperado %s, veio %s" % (show(sorted(LIGHT)), show(got)))

    @t.case("ingredientes das trevas: bats, frogs, arsenic, eyeball")
    def _():
        t.need(DARK_BOOK)
        src = low(t.source(DARK_BOOK))
        missing = [w for w in DARK if w not in src]
        t.expect(not missing, "dark_spellbook.py deveria listar bats, frogs, arsenic, eyeball (faltou %s)"
                 % show(missing))

    def validate(ing):
        return call(t, "alchemy.grimoire.light_validator", "validate_ingredients", ing)

    for ing in ("Earth, wind and fire", "AIR", "Water", "molten fire"):
        @t.case("validate_ingredients(%s) é VALID (sem diferenciar maiúsculas)" % show(ing))
        def _(ing=ing):
            got = validate(ing)
            t.expect(isinstance(got, str) and low(ing) in low(got), "esperado os ingredientes %s no retorno, veio %s"
                     % (show(ing), show(got)))
            t.expect("VALID" in got and "INVALID" not in got, "esperado VALID, veio %s" % show(got))

    for ing in ("bats, frogs", "salt", ""):
        @t.case("validate_ingredients(%s) é INVALID" % show(ing))
        def _(ing=ing):
            got = validate(ing)
            t.expect(isinstance(got, str) and low(ing) in low(got), "esperado os ingredientes %s no retorno, veio %s"
                     % (show(ing), show(got)))
            t.expect("INVALID" in got, "esperado INVALID, veio %s" % show(got))

    def record(name, ing):
        return call(t, "alchemy.grimoire.light_spellbook", "light_spell_record", name, ing)

    @t.case("light_spell_record com ingrediente da luz: gravado (exemplo do subject)")
    def _():
        got = record("Fantasy", "Earth, wind and fire")
        t.expect(isinstance(got, str), "esperado uma str, veio %s" % show(got))
        t.expect("Fantasy" in got and "Earth, wind and fire" in got,
                 "esperado o nome e os ingredientes no retorno, veio %s" % show(got))
        t.expect("record" in low(got) and "reject" not in low(got) and "INVALID" not in got,
                 "esperado algo como %s, veio %s" % (show("Spell recorded: Fantasy (Earth, wind and fire - VALID)"),
                                                     show(got)))

    @t.case("light_spell_record sem ingrediente da luz: rejeitado")
    def _():
        got = record("Curse", "bats and frogs")
        t.expect(isinstance(got, str) and "Curse" in got, "esperado o nome do feitiço no retorno, veio %s"
                 % show(got))
        t.expect("reject" in low(got) or "INVALID" in got,
                 "esperado algo como %s, veio %s" % (show("Spell rejected: Curse (bats and frogs - INVALID)"),
                                                     show(got)))

    t.case("light_spellbook usa o validador para decidir",
           lambda: need_import(t, LIGHT_BOOK, lambda i: mentions(i, "light_validator") or mentions(i, "validate"),
                               "from .light_validator import validate_ingredients"))

    for first in (LIGHT_BOOK, LIGHT_VAL, GINIT):
        mod = first[:-3].replace("/", ".").replace(".__init__", "")

        @t.case("luz sem import circular: `import %s` funciona" % mod)
        def _(mod=mod, first=first):
            t.need(first)
            r = t.py("import " + mod, ex=".", allow_crash=True)
            t.expect(not r.crashed, "importar %s explode: %s" % (mod, show(r.last_error())))

    @t.case("trevas: dark_spellbook e dark_validator se importam no topo (ciclo de verdade)")
    def _():
        t.need(DARK_BOOK, DARK_VAL)
        t.expect(mentions(imports(t, DARK_BOOK, top_only=True), "dark_validator"),
                 "dark_spellbook.py deveria importar dark_validator no topo do arquivo")
        t.expect(mentions(imports(t, DARK_VAL, top_only=True), "dark_spellbook"),
                 "dark_validator.py deveria importar dark_spellbook no topo do arquivo")

    @t.case("trevas: `import alchemy.grimoire.dark_spellbook` explode com import circular")
    def _():
        t.need(DARK_BOOK)
        r = t.py("import alchemy.grimoire.dark_spellbook", ex=".", allow_crash=True)
        err = r.last_error()
        t.expect(r.crashed and ("ImportError" in err or "AttributeError" in err)
                 and ("circular" in err or "partially initialized" in err),
                 "esperado ImportError de import circular (partially initialized module), veio %s"
                 % show(err if r.crashed else "nenhum erro"))

    t.case("ft_kaboom_0 acessa o pacote grimoire",
           lambda: need_import(t, k0, lambda i: any(m.startswith("alchemy.grimoire") and "dark" not in m
                                                    for k, m, ns in i) or has(i, "from", "alchemy", "grimoire"),
                               "from alchemy.grimoire.light_spellbook import light_spell_record"))

    @t.case("ft_kaboom_0 grava um feitiço da luz sem explodir")
    def _():
        r = t.run(k0)
        t.expect("record" in low(r.out) and "VALID" in r.out and "INVALID" not in r.out,
                 "esperado algo como %s, veio %s"
                 % (show("Spell recorded: Fantasy (Earth, wind and fire - VALID)"), show(r.out)))

    t.case("ft_kaboom_1 acessa dark_spellbook direto",
           lambda: need_import(t, k1, lambda i: mentions(i, "dark_spellbook"),
                               "from alchemy.grimoire.dark_spellbook import dark_spell_record"))

    @t.case("ft_kaboom_1 explode com o import circular")
    def _():
        r = t.run(k1, allow_crash=True)
        both = low(r.out + r.err)
        t.expect("circular" in both or "partially initialized" in both,
                 "esperado o erro de import circular (no stderr ou capturado), veio %s / %s"
                 % (show(r.out), show(r.err[-300:])))
        t.expect("recorded" not in low(r.out), "o feitiço das trevas não deveria ser gravado: %s" % show(r.out))

    static(t, [GINIT, LIGHT_BOOK, LIGHT_VAL, DARK_BOOK, DARK_VAL, k0, k1])


def suite(t):
    # ft_alembic_1 has a mypy error on purpose: checked apart, without mypy.
    common(t, [f for f in PY if f != A1], typed=PY)
    t.group("entrega")
    for rel in [A1] + OTHER:
        t.case(rel, lambda rel=rel: t.need(rel))

    @t.case("flake8 em " + A1)
    def _():
        t.need(A1)
        r = t.tool("flake8", [A1])
        t.expect(r.code == 0, r.out.strip() or r.err.strip())

    t.case("sintaxe de " + A1, lambda: (t.need(A1), t.tree(A1)))
    rules(t)
    ex0_construct(t)
    ex0_loading(t)
    ex0_deps(t)
    ex1_static(t)
    ex1_oracle(t)
    ex2(t)
    ex3(t)
    ex4(t)
