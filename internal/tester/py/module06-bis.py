"""Python Module 06 bis (Code Nexus): abstract classes, polymorphism,
Protocols, abstract factories and strategies."""

import ast
import builtins
import csv
import glob
import json
import os
import re

from lt import common, show

DP, DS, DPL = "data_processor.py", "data_stream.py", "data_pipeline.py"
BATTLE, CAPACITOR, TOURNAMENT = "battle.py", "capacitor.py", "tournament.py"
ROOT = [DP, DS, DPL, BATTLE, CAPACITOR, TOURNAMENT]
IMPORTS = {"abc", "typing", "data_processor", "data_stream", "data_pipeline", "ex3", "ex4"}
BUILTINS = set(dir(builtins)) - {"eval", "exec"}

LOG1 = {"log_level": "NOTICE", "log_message": "Connection to server"}
LOG2 = {"log_level": "ERROR", "log_message": "Unauthorized access!!"}
BATCH1 = ["Hello world", [3.14, -1, 2.71],
          [{"log_level": "WARNING", "log_message": "Telnet access! Use ssh instead"},
           {"log_level": "INFO", "log_message": "User wil is connected"}],
          42, ["Hi", "five"]]
BATCH2 = [21, ["I love AI", "LLMs are wonderful", "Stay healthy"],
          [{"log_level": "ERROR", "log_message": "500 server crash"},
           {"log_level": "NOTICE", "log_message": "Certificate expires in 10 days"}],
          [32, 42, 64, 84, 128, 168], "World hello"]

# The subject's entities: checked to the letter when the student kept the names.
KNOWN = {
    "Flameling": ("Flameling is a Fire type Creature", "Flameling uses Ember!"),
    "Pyrodon": ("Pyrodon is a Fire/Flying type Creature", "Pyrodon uses Flamethrower!"),
    "Shiftling": ("Shiftling is a Normal type Creature", "Shiftling attacks normally.",
                  "Shiftling shifts into a sharper form!", "Shiftling performs a boosted strike!",
                  "Shiftling returns to normal."),
    "Morphagon": ("Morphagon is a Normal/Dragon type Creature", "Morphagon attacks normally.",
                  "Morphagon morphs into a dragonic battle form!",
                  "Morphagon unleashes a devastating morph strike!", "Morphagon stabilizes its form."),
}


def norm(s):
    """Lenient on quotes, case and spacing; strict on the words."""
    s = re.sub(r"[\"'`‘’“”]", "", str(s)).lower()
    return " ".join(s.split())


# ---- running -----------------------------------------------------------

# Helpers for every probe (run from the project root). Student modules are
# imported with stdout silenced; text() takes a returned str or, if the
# method prints instead, what it printed.
PRE = r'''
import abc, contextlib, glob, importlib, inspect, io, os

def done(x):
    print('@@' + repr(x))

def quiet(f, *a):
    with contextlib.redirect_stdout(io.StringIO()):
        return f(*a)

def cap(f, *a):
    b = io.StringIO()
    with contextlib.redirect_stdout(b):
        r = f(*a)
    return r, b.getvalue()

def text(f, *a):
    r, s = cap(f, *a)
    return (r if isinstance(r, str) else s).strip()

def load(mod):
    return quiet(importlib.import_module, mod)

def get(mod, name):
    v = getattr(load(mod), name, None)
    for o in ('data_stream', 'data_processor'):
        if v is None and o != mod and os.path.exists(o + '.py'):
            v = getattr(load(o), name, None)
    if v is None:
        raise LookupError('%s.py não tem %s' % (mod, name))
    return v

def classes(pkg):
    out = []
    for f in sorted(glob.glob(pkg + '/**/*.py', recursive=True)):
        mod = f[:-3].replace(os.sep, '.')
        if mod.endswith('.__init__'):
            mod = mod[:-9]
        for v in vars(load(mod)).values():
            if inspect.isclass(v) and v.__module__.startswith(pkg) and v not in out:
                out.append(v)
    return out

def root(cs, *names):
    found = [c for c in cs if set(names) <= set(getattr(c, '__abstractmethods__', ()))]
    return min(found, key=lambda c: len(c.__mro__)) if found else None

def abstract(cls):
    """Instantiating cls fails because it is abstract."""
    try:
        quiet(cls)
    except TypeError as e:
        return 'abstract' in str(e)
    return False

def init(obj, cls):
    """Calls cls.__init__ padding its required parameters with 'LT'."""
    ps = [p for p in inspect.signature(cls.__init__).parameters.values()
          if p.default is p.empty and p.kind in (p.POSITIONAL_ONLY, p.POSITIONAL_OR_KEYWORD)]
    cls.__init__(obj, *(['LT'] * (len(ps) - 1)))
'''

_memo = {}


def value(t, body, files=()):
    """Runs PRE + body from the project root; body calls done(x), x comes
    back. Memoized: many cases read one probe."""
    if body not in _memo:
        try:
            r = t.py(PRE + body, ex=".", files=files)
            got = [ast.literal_eval(line[2:]) for line in r.out.splitlines() if line.startswith("@@")]
            t.expect(got, "o teste não produziu resultado (saída: %s)" % show(r.out[-300:]))
            _memo[body] = (True, got[-1])
        except Exception as e:
            _memo[body] = (False, e)
    ok, v = _memo[body]
    if not ok:
        raise v
    return v


def has_lines(t, out, wants, what):
    """wants appear in out in this order, each inside one line."""
    lines = [norm(x) for x in out.splitlines()]
    i = 0
    for w in wants:
        while i < len(lines) and norm(w) not in lines[i]:
            i += 1
        t.expect(i < len(lines), "%s: esperado %s (nessa ordem), faltou %s. Saída: %s"
                 % (what, show(list(wants)), show(w), show(out)))
        i += 1


def stat_line(t, out, word, total, remaining):
    """The stats line of a processor (the one naming word) shows total then
    remaining."""
    lines = [x for x in out.splitlines() if word in x.lower()]
    t.expect(lines, "nenhuma linha de estatística do %s processor em %s" % (word, show(out)))
    nums = [int(n) for n in re.findall(r"\d+", lines[-1])]
    t.expect(total in nums and remaining in nums[nums.index(total) + 1:] if total in nums else False,
             "%s processor: esperado total %d e restante %d, veio %s" % (word, total, remaining, show(lines[-1])))


# ---- static helpers ------------------------------------------------------


def method(t, rel, cls, name):
    for n in t.tree(rel).body:
        if isinstance(n, ast.ClassDef) and n.name == cls:
            for f in n.body:
                if isinstance(f, ast.FunctionDef) and f.name == name:
                    return f
    t.expect(False, "%s não define %s.%s" % (rel, cls, name))


def ann(node):
    if node is None:
        return ""
    return ast.unparse(node).replace("typing.", "").replace("Tuple[", "tuple[").replace(" ", "")


def names_used(t, rel):
    tree = t.tree(rel)
    out = {n.id for n in ast.walk(tree) if isinstance(n, ast.Name)}
    out |= {n.attr for n in ast.walk(tree) if isinstance(n, ast.Attribute)}
    out |= {a.name for n in ast.walk(tree) if isinstance(n, ast.ImportFrom) for a in n.names}
    return out


def imported(t, rel):
    try:
        tree = t.tree(rel)
    except Exception:
        return set()
    mods = {a.name for n in ast.walk(tree) if isinstance(n, ast.Import) for a in n.names}
    mods |= {n.module for n in ast.walk(tree) if isinstance(n, ast.ImportFrom) and n.module}
    return {m.split(".")[0] for m in mods}


# ---- shared checks: files, flake8, mypy ----------------------------------


def package_files(t, pkg):
    base = t.path(pkg)
    found = sorted(os.path.relpath(p, t.root) for p in glob.glob(os.path.join(base, "**", "*.py"), recursive=True))
    return sorted(set(found) | {pkg + "/__init__.py"})


def checks(t, files):
    # data_processor.py has a mypy error on purpose (ingest of invalid data
    # without validation), and mypy reports it in every file that imports it:
    # those files get a mypy run that forgives the [arg-type] of that file.
    special = {DP}
    grew = True
    while grew:
        grew = False
        for f in (DS, DPL):
            if f not in special and t.exists(f) and {s[:-3] for s in special} & imported(t, f):
                special.add(f)
                grew = True
    special = [f for f in ROOT if f in special]
    rest = [f for f in files if f not in special]
    common(t, rest, typed=files)

    t.group("entrega")
    for rel in special:
        t.case(rel, lambda rel=rel: t.need(rel))

    t.group("flake8")

    @t.case("flake8 em " + ", ".join(special))
    def _():
        present = [f for f in special if t.exists(f)]
        if not present:
            t.skip("nenhum arquivo entregue")
        r = t.tool("flake8", present)
        t.expect(r.code == 0, r.out.strip() or r.err.strip())

    t.group("mypy")

    @t.case("mypy em %s (só o erro de propósito do ingest inválido)" % ", ".join(special))
    def _():
        present = [f for f in special if t.exists(f)]
        if not present:
            t.skip("nenhum arquivo entregue")
        r = t.tool("mypy", ["--ignore-missing-imports", "--no-error-summary", "--cache-dir", "/dev/null"] + present)
        bad = [x for x in r.out.splitlines() if ": error:" in x
               and not (x.startswith(DP + ":") and "[arg-type]" in x)]
        t.expect(not bad and (r.code in (0, 1)), "\n".join(bad) or r.err.strip())

    t.group("sintaxe")
    for rel in special:
        t.case(rel, lambda rel=rel: (t.need(rel), t.tree(rel)))

    t.group("imports permitidos")
    for rel in files:
        t.case("só abc, typing e o próprio projeto, sem eval/exec: " + rel,
               lambda rel=rel: (t.need(rel), t.authorized(rel, allowed=BUILTINS, imports=IMPORTS)))


# ---- ex0: data_processor.py ------------------------------------------------

PROCS = ("NumericProcessor", "TextProcessor", "LogProcessor")

VALID = {
    "NumericProcessor": [(42, True), (3.14, True), (-1, True), ([1, 2, 3], True), ([3.14, -1, 2], True),
                         ("Hello", False), ("42", False), (["a"], False), ([1, "a"], False),
                         (LOG1, False), (None, False)],
    "TextProcessor": [("Hello", True), (["Hello", "Nexus"], True), (42, False), ([1, 2], False),
                      (["a", 1], False), (LOG1, False), (None, False)],
    "LogProcessor": [(LOG1, True), ([LOG1, LOG2], True), ("Hello", False), (42, False), (["a"], False),
                     ({"log_level": 1, "log_message": 2}, False), ([LOG1, "x"], False)],
}

INVALID = [("NumericProcessor", "foo"), ("NumericProcessor", ["a", 1]), ("TextProcessor", 42),
           ("TextProcessor", [1, 2]), ("LogProcessor", "Hello"), ("LogProcessor", [42])]

# (processor, data ingested in turn, how many outputs, expected outputs)
OUTPUTS = [
    ("NumericProcessor", [[1, 2, 3, 4, 5]], 3, [(0, "1"), (1, "2"), (2, "3")]),
    ("NumericProcessor", [3.14, [-1, 2.71], 42], 4, [(0, "3.14"), (1, "-1"), (2, "2.71"), (3, "42")]),
    ("TextProcessor", [["Hello", "Nexus", "World"], "Hi"], 4,
     [(0, "Hello"), (1, "Nexus"), (2, "World"), (3, "Hi")]),
    ("LogProcessor", [[LOG1, LOG2]], 2, [(0, "NOTICE: Connection to server"), (1, "ERROR: Unauthorized access!!")]),
    ("LogProcessor", [LOG2], 1, [(0, "ERROR: Unauthorized access!!")]),
]


def ex0(t):
    t.group("ex0 data_processor")
    m = "data_processor"

    t.case("define DataProcessor, NumericProcessor, TextProcessor e LogProcessor",
           lambda: (t.need(DP), t.defines(DP, "DataProcessor", *PROCS)))

    def info():
        return value(t, "D = get(%r, 'DataProcessor')\n"
                        "done({'abc': abc.ABC in D.__mro__, 'abstract': abstract(D),\n"
                        "      'methods': sorted(getattr(D, '__abstractmethods__', ())), 'output': callable(getattr(D, 'output', None)),\n"
                        "      'subs': {n: [issubclass(get(%r, n), D), sorted(vars(get(%r, n))),\n"
                        "                   sorted(getattr(get(%r, n), '__abstractmethods__', ()))] for n in %r}})\n"
                        % (m, m, m, m, PROCS), files=(DP,))

    t.case("DataProcessor herda de ABC", lambda: t.expect(info()["abc"], "DataProcessor deveria herdar de ABC"))
    t.case("DataProcessor() dá TypeError (classe abstrata)",
           lambda: t.expect(info()["abstract"], "instanciar DataProcessor deveria dar TypeError de classe abstrata"))

    @t.case("validate e ingest são abstratos, output é concreto")
    def _():
        got = info()["methods"]
        t.expect({"validate", "ingest"} <= set(got), "esperado validate e ingest abstratos, veio %s" % show(got))
        t.expect("output" not in got and info()["output"], "output deveria ser um método normal de DataProcessor")

    for name in PROCS:
        @t.case("%s herda de DataProcessor e sobrescreve validate e ingest" % name)
        def _(name=name):
            sub, own, left = info()["subs"][name]
            t.expect(sub, "%s deveria herdar de DataProcessor" % name)
            miss = [x for x in ("validate", "ingest") if x not in own]
            t.expect(not miss, "%s não sobrescreve %s" % (name, ", ".join(miss)))
            t.expect(not left, "%s ainda tem métodos abstratos: %s" % (name, show(left)))

    @t.case("assinaturas de DataProcessor: validate(data: Any) -> bool, ingest(data: Any) -> None, output -> tuple[int, str]")
    def _():
        t.need(DP)
        v, i, o = (method(t, DP, "DataProcessor", x) for x in ("validate", "ingest", "output"))
        got = (ann(v.args.args[-1].annotation), ann(v.returns), ann(i.args.args[-1].annotation), ann(i.returns),
               ann(o.returns))
        t.expect(got == ("Any", "bool", "Any", "None", "tuple[int,str]"), "assinaturas diferentes do subject: %s"
                 % show(got))

    for name in PROCS:
        @t.case("%s: validate(data: Any) -> bool e ingest com o tipo aceito (não Any)" % name)
        def _(name=name):
            t.need(DP)
            v, i = method(t, DP, name, "validate"), method(t, DP, name, "ingest")
            t.expect(ann(v.args.args[-1].annotation) == "Any" and ann(v.returns) == "bool",
                     "%s.validate deveria ser (self, data: Any) -> bool" % name)
            a = ann(i.args.args[-1].annotation)
            t.expect(a and "Any" not in a, "%s.ingest deveria anotar os tipos aceitos, veio %s" % (name, show(a or "nada")))

    def validates(name):
        return value(t, "p = get(%r, %r)()\n"
                        "done([(lambda r: r if isinstance(r, bool) else repr(r))(p.validate(d)) for d in %r])\n"
                     % (m, name, [d for d, _ in VALID[name]]), files=(DP,))

    for name, cases in VALID.items():
        for i, (data, want) in enumerate(cases):
            @t.case("%s.validate(%s) é %s" % (name, show(data), want))
            def _(name=name, i=i, data=data, want=want):
                got = validates(name)[i]
                t.expect(got is want, "esperado %s, veio %s" % (want, show(got)))

    for name, data in INVALID:
        @t.case("%s.ingest(%s) sem validar levanta exceção" % (name, show(data)))
        def _(name=name, data=data):
            got = value(t, "p = get(%r, %r)()\n"
                           "try:\n    quiet(p.ingest, %r)\n    done('nada')\n"
                           "except Exception as e:\n    done('ok')\n" % (m, name, data), files=(DP,))
            t.expect(got == "ok", "esperado uma exceção para dado inválido, não levantou nada")

    for name, feeds, n, want in OUTPUTS:
        @t.case("%s: ingest %s e %d output() -> %s" % (name, " + ".join(show(f) for f in feeds), n, show(want)))
        def _(name=name, feeds=feeds, n=n, want=want):
            got = value(t, "p = get(%r, %r)()\n"
                           "for d in %r:\n    quiet(p.ingest, d)\n"
                           "done([quiet(p.output) for _ in range(%d)])\n" % (m, name, feeds, n), files=(DP,))
            t.expect(all(isinstance(x, tuple) for x in got), "output() deveria retornar uma tuple, veio %s" % show(got))
            t.expect(got == want, "esperado %s (o mais antigo primeiro, com o rank), veio %s" % (show(want), show(got)))

    @t.case("o rank continua depois de consumir: ingest [1, 2], output, ingest 3")
    def _():
        got = value(t, "p = get(%r, 'NumericProcessor')()\nquiet(p.ingest, [1, 2])\na = quiet(p.output)\n"
                       "quiet(p.ingest, 3)\ndone([a, quiet(p.output), quiet(p.output)])\n" % m, files=(DP,))
        want = [(0, "1"), (1, "2"), (2, "3")]
        t.expect(got == want, "esperado %s, veio %s" % (show(want), show(got)))

    @t.case("cada instância guarda os próprios dados (nada compartilhado na classe)")
    def _():
        got = value(t, "a, b = get(%r, 'NumericProcessor')(), get(%r, 'NumericProcessor')()\n"
                       "quiet(a.ingest, 1)\nquiet(b.ingest, 2)\ndone(quiet(b.output))\n" % (m, m), files=(DP,))
        t.expect(got == (0, "2"), "o segundo processor deveria dar (0, '2'), veio %s" % show(got))

    @t.case("python3 data_processor.py roda e mostra as validações")
    def _():
        r = t.run(DP)
        t.expect("True" in r.out and "False" in r.out, "esperado testes de validate (True e False) na saída: %s"
                 % show(r.out))


# ---- ex1: data_stream.py ---------------------------------------------------

STREAM = r'''
ds = get(%(m)r, 'DataStream')()
n, tx, lg = (get(%(m)r, c)() for c in ('NumericProcessor', 'TextProcessor', 'LogProcessor'))
r = {'empty': cap(ds.print_processors_stats)[1]}
quiet(ds.register_processor, n)
r['errors'] = cap(ds.process_stream, %(b1)r)[1]
r['stats1'] = cap(ds.print_processors_stats)[1]
quiet(ds.register_processor, tx)
quiet(ds.register_processor, lg)
quiet(ds.process_stream, %(b1)r)
r['stats2'] = cap(ds.print_processors_stats)[1]
r['out'] = [[quiet(p.output) for _ in range(k)] for p, k in ((n, 3), (tx, 2), (lg, 1))]
r['stats3'] = cap(ds.print_processors_stats)[1]
done(r)
'''


def ex1(t):
    t.group("ex1 data_stream")
    m = "data_stream"

    @t.case("DataStream tem register_processor, process_stream e print_processors_stats")
    def _():
        got = value(t, "D = get(%r, 'DataStream')\ndone([x for x in ('register_processor', 'process_stream',"
                       " 'print_processors_stats') if not callable(getattr(D, x, None))])\n" % m, files=(DS,))
        t.expect(not got, "DataStream não tem %s" % show(got))

    def run():
        return value(t, STREAM % {"m": m, "b1": BATCH1}, files=(DS,))

    t.case("print_processors_stats sem processor mostra algo",
           lambda: t.expect(run()["empty"].strip(), "nada impresso sem processors (ex.: 'No processor found, no data')"))

    @t.case("só Numeric registrado: um erro impresso por elemento que ninguém processa (3)")
    def _():
        lines = [x for x in run()["errors"].splitlines() if x.strip()]
        t.expect(len(lines) >= 3, "esperado 3 mensagens de erro, veio %s" % show(run()["errors"]))

    t.case("só Numeric: estatística total 4, restante 4",
           lambda: stat_line(t, run()["stats1"], "numeric", 4, 4))

    for word, total, rem in (("numeric", 8, 8), ("text", 3, 3), ("log", 2, 2)):
        t.case("mesmo lote com os 3 registrados: %s total %d, restante %d" % (word, total, rem),
               lambda word=word, total=total, rem=rem: stat_line(t, run()["stats2"], word, total, rem))

    @t.case("cada elemento foi para o processor certo (output de cada um)")
    def _():
        want = [[(0, "3.14"), (1, "-1"), (2, "2.71")], [(0, "Hello world"), (1, "Hi")],
                [(0, "WARNING: Telnet access! Use ssh instead")]]
        t.expect(run()["out"] == want, "esperado %s, veio %s" % (show(want), show(run()["out"])))

    for word, total, rem in (("numeric", 8, 5), ("text", 3, 1), ("log", 2, 1)):
        t.case("depois de consumir 3/2/1: %s total %d, restante %d" % (word, total, rem),
               lambda word=word, total=total, rem=rem: stat_line(t, run()["stats3"], word, total, rem))

    @t.case("process_stream não explode com tipos desconhecidos (None, set, tuple)")
    def _():
        got = value(t, "ds = get(%r, 'DataStream')()\nn = get(%r, 'NumericProcessor')()\n"
                       "quiet(ds.register_processor, n)\nquiet(ds.process_stream, [None, {1, 2}, (1, 2), 7])\n"
                       "done(quiet(n.output))\n" % (m, m), files=(DS,))
        t.expect(got == (0, "7"), "esperado (0, '7') no Numeric, veio %s" % show(got))

    @t.case("polimorfismo: aceita um DataProcessor novo, criado pelo teste")
    def _():
        got = value(t, "D = get(%r, 'DataProcessor')\nseen = []\n"
                       "class LTProc(D):\n"
                       "    def __init__(self):\n        init(self, D)\n"
                       "    def validate(self, data):\n        return isinstance(data, complex)\n"
                       "    def ingest(self, data):\n        seen.append(data)\n"
                       "ds = get(%r, 'DataStream')()\ntx = get(%r, 'TextProcessor')()\n"
                       "quiet(ds.register_processor, LTProc())\nquiet(ds.register_processor, tx)\n"
                       "quiet(ds.process_stream, [1j, 'x'])\ndone([repr(seen), quiet(tx.output)])\n" % (m, m, m),
                    files=(DS,))
        t.expect(got == ["[1j]", (0, "x")], "o DataStream deveria mandar 1j para o processor novo e 'x' ao Text, "
                 "veio %s" % show(got))

    t.case("python3 data_stream.py roda", lambda: t.expect(t.run(DS).out.strip(), "nada impresso"))


# ---- ex2: data_pipeline.py -------------------------------------------------

PIPE = r'''
calls = []
class Rec:
    def process_output(self, data):
        calls.append([tuple(x) for x in data] if isinstance(data, list) else repr(data))
ds = get(%(m)r, 'DataStream')()
for c in ('NumericProcessor', 'TextProcessor', 'LogProcessor'):
    quiet(ds.register_processor, get(%(m)r, c)())
quiet(ds.process_stream, %(b1)r)
r = {}
quiet(ds.output_pipeline, 3, Rec())
r['calls1'], calls[:] = list(calls), []
r['stats1'] = cap(ds.print_processors_stats)[1]
quiet(ds.process_stream, %(b2)r)
r['stats2'] = cap(ds.print_processors_stats)[1]
quiet(ds.output_pipeline, 5, Rec())
r['calls2'] = list(calls)
r['stats3'] = cap(ds.print_processors_stats)[1]
done(r)
'''

CALLS1 = [[(0, "3.14"), (1, "-1"), (2, "2.71")], [(0, "Hello world"), (1, "Hi"), (2, "five")],
          [(0, "WARNING: Telnet access! Use ssh instead"), (1, "INFO: User wil is connected")]]
CALLS2 = [[(3, "42"), (4, "21"), (5, "32"), (6, "42"), (7, "64")],
          [(3, "I love AI"), (4, "LLMs are wonderful"), (5, "Stay healthy"), (6, "World hello")],
          [(2, "ERROR: 500 server crash"), (3, "NOTICE: Certificate expires in 10 days")]]


def ex2(t):
    t.group("ex2 data_pipeline")
    m = "data_pipeline"

    @t.case("ExportPlugin é um Protocol com process_output")
    def _():
        got = value(t, "import typing\nE = get(%r, 'ExportPlugin')\n"
                       "done([typing.Protocol in E.__mro__ and getattr(E, '_is_protocol', False),"
                       " 'process_output' in vars(E)])\n" % m, files=(DPL,))
        t.expect(got[0], "ExportPlugin deveria herdar de typing.Protocol")
        t.expect(got[1], "ExportPlugin deveria declarar process_output")

    @t.case("process_output(self, data: list[tuple[int, str]]) -> None no ExportPlugin")
    def _():
        t.need(DPL)
        f = method(t, DPL, "ExportPlugin", "process_output")
        got = (ann(f.args.args[-1].annotation), ann(f.returns))
        t.expect(got == ("list[tuple[int,str]]", "None"), "assinatura diferente do subject: %s" % show(got))

    @t.case("DataStream.output_pipeline(self, nb: int, plugin: ExportPlugin) -> None")
    def _():
        t.need(DPL)
        f = method(t, DPL, "DataStream", "output_pipeline")
        got = [ann(a.annotation) for a in f.args.args[1:]] + [ann(f.returns)]
        t.expect(got == ["int", "ExportPlugin", "None"], "assinatura diferente do subject: %s" % show(got))

    def plugins():
        return value(t, "E = get(%r, 'ExportPlugin')\nM = load(%r)\n"
                        "done(sorted(n for n, c in vars(M).items() if inspect.isclass(c) and c is not E\n"
                        "            and callable(vars(c).get('process_output'))))\n" % (m, m), files=(DPL,))

    def plugin(kind):
        got = [p for p in plugins() if kind in p.lower()]
        t.expect(got, "nenhum plugin %s (classe com process_output e 'csv'/'json' no nome), achei %s"
                 % (kind.upper(), show(plugins())))
        return got[0]

    def out_of(kind, data):
        name = plugin(kind)
        return value(t, "done(cap(load(%r).%s().process_output, %r)[1])\n" % (m, name, data), files=(DPL,))

    t.case("tem um plugin CSV", lambda: plugin("csv"))
    t.case("tem um plugin JSON", lambda: plugin("json"))

    @t.case("plugin CSV imprime 3.14,-1,2.71")
    def _():
        out = out_of("csv", [(0, "3.14"), (1, "-1"), (2, "2.71")])
        t.expect("3.14,-1,2.71" in [x.strip() for x in out.splitlines()], "esperado a linha %s, veio %s"
                 % (show("3.14,-1,2.71"), show(out)))

    @t.case("plugin CSV gera CSV válido com vírgula no valor ('a,b')")
    def _():
        out = out_of("csv", [(0, "a,b"), (1, "c")])
        rows = list(csv.reader(out.splitlines()))
        t.expect(["a,b", "c"] in rows, "esperado uma linha CSV que leia ['a,b', 'c'] (com aspas), veio %s"
                 % show(out))

    def json_of(out):
        try:
            return json.loads(out[out.index("{"):out.rindex("}") + 1])
        except ValueError:
            t.expect(False, "a saída não tem um objeto JSON válido: %s" % show(out))

    @t.case("plugin JSON imprime um objeto JSON com os valores na ordem")
    def _():
        got = json_of(out_of("json", [(3, "42"), (4, "21"), (5, "32")]))
        t.expect(isinstance(got, dict) and list(got.values()) == ["42", "21", "32"],
                 "esperado {\"item_3\": \"42\", \"item_4\": \"21\", \"item_5\": \"32\"}, veio %s" % show(got))

    @t.case("plugin JSON escapa aspas e barras (JSON válido)")
    def _():
        want = ['say "hi"', "back\\slash"]
        got = json_of(out_of("json", [(0, want[0]), (1, want[1])]))
        t.expect(isinstance(got, dict) and list(got.values()) == want, "esperado os valores %s, veio %s"
                 % (show(want), show(got)))

    def run():
        return value(t, PIPE % {"m": m, "b1": BATCH1, "b2": BATCH2}, files=(DPL,))

    @t.case("output_pipeline(3) com um plugin qualquer (duck typing): consome 3 de cada processor")
    def _():
        got = [x for c in run()["calls1"] for x in c]
        want = [x for c in CALLS1 for x in c]
        t.expect(got == want, "esperado %s, veio %s" % (show(want), show(got)))

    @t.case("output_pipeline chama o plugin uma vez por processor")
    def _():
        got = [c for c in run()["calls1"] if c]
        t.expect(got == CALLS1, "esperado as chamadas %s, veio %s" % (show(CALLS1), show(got)))

    for key, rows in (("stats1", (("numeric", 4, 1), ("text", 3, 0), ("log", 2, 0))),
                      ("stats2", (("numeric", 11, 8), ("text", 7, 4), ("log", 4, 2))),
                      ("stats3", (("numeric", 11, 3), ("text", 7, 0), ("log", 4, 0)))):
        when = {"stats1": "depois do output_pipeline(3)", "stats2": "depois do segundo lote",
                "stats3": "depois do output_pipeline(5)"}[key]
        for word, total, rem in rows:
            t.case("%s: %s total %d, restante %d" % (when, word, total, rem),
                   lambda key=key, word=word, total=total, rem=rem: stat_line(t, run()[key], word, total, rem))

    @t.case("output_pipeline(5) no segundo lote: ranks continuam (item_3...)")
    def _():
        got = [c for c in run()["calls2"] if c]
        t.expect(got == CALLS2, "esperado %s, veio %s" % (show(CALLS2), show(got)))

    @t.case("output_pipeline com nb maior que o que sobrou não explode")
    def _():
        got = value(t, "calls = []\nclass Rec:\n    def process_output(self, data):\n        calls.extend(data)\n"
                       "ds = get(%r, 'DataStream')()\nn = get(%r, 'NumericProcessor')()\n"
                       "quiet(ds.register_processor, n)\nquiet(ds.process_stream, [1, 2])\n"
                       "quiet(ds.output_pipeline, 10, Rec())\ndone([tuple(x) for x in calls])\n" % (m, m),
                    files=(DPL,))
        t.expect(got == [(0, "1"), (1, "2")], "esperado [(0, '1'), (1, '2')], veio %s" % show(got))

    t.case("python3 data_pipeline.py roda", lambda: t.expect(t.run(DPL).out.strip(), "nada impresso"))


# ---- ex3: factories and capabilities ---------------------------------------

EX3 = r'''
ex3 = load('ex3')
cs = classes('ex3')
C = root(cs, 'attack')
Cap = root(cs, 'transform', 'revert')
F = root(cs, 'create_base', 'create_evolved')
r = {}
for k, c in (('creature', C), ('cap', Cap), ('factory', F)):
    r[k] = c and {'name': c.__name__, 'abc': abc.ABC in c.__mro__, 'abstract': abstract(c),
                  'methods': sorted(getattr(c, '__abstractmethods__', ()))}
if C:
    r['creature']['describe'] = callable(getattr(C, 'describe', None)) and 'describe' not in C.__abstractmethods__
r['cap_is_creature'] = bool(C and Cap and (issubclass(Cap, C) or issubclass(C, Cap)))
r['entities'] = sorted(c.__name__ for c in cs if C and issubclass(c, C) and not inspect.isabstract(c))
exposed = {n: v for n, v in vars(ex3).items() if inspect.isclass(v)}
r['exposed'] = sorted(n for n, v in exposed.items() if C and issubclass(v, C) and not inspect.isabstract(v))
r['factories'] = []
for n, fc in sorted(exposed.items()):
    if not (F and issubclass(fc, F) and not inspect.isabstract(fc)):
        continue
    f = quiet(fc)
    fam = {'name': n}
    for kind in ('base', 'evolved'):
        make = getattr(f, 'create_' + kind)
        e = quiet(make)
        d = {'cls': type(e).__name__, 'creature': bool(C) and isinstance(e, C),
             'cap': bool(Cap) and isinstance(e, Cap), 'fresh': quiet(make) is not e,
             'describe': text(e.describe), 'attack': text(e.attack)}
        if d['cap']:
            before = dict(vars(e))
            d['transform'] = text(e.transform)
            d['state'] = sorted(k for k in vars(e) if vars(e)[k] != before.get(k))
            d['attack2'] = text(e.attack)
            d['revert'] = text(e.revert)
            d['attack3'] = text(e.attack)
        fam[kind] = d
    r['factories'].append(fam)
done(r)
'''


def ex3(t):
    t.group("ex3 fábricas e capacidades")

    def info():
        return value(t, EX3, files=("ex3/__init__.py",))

    def part(key, what):
        got = info()[key]
        t.expect(got, "não achei %s no ex3" % what)
        return got

    t.case("`import ex3` funciona", info)

    @t.case("Creature: classe abstrata (ABC) com attack abstrato e describe concreto")
    def _():
        c = part("creature", "a classe abstrata com attack abstrato (Creature)")
        t.expect(c["abc"], "%s deveria herdar de ABC" % c["name"])
        t.expect(c["describe"], "%s deveria ter describe concreto" % c["name"])

    t.case("Creature() dá TypeError (classe abstrata)",
           lambda: t.expect(part("creature", "Creature")["abstract"], "instanciar Creature deveria dar TypeError"))

    @t.case("TransformCapability: ABC com transform e revert abstratos")
    def _():
        c = part("cap", "a classe abstrata com transform e revert abstratos (TransformCapability)")
        t.expect(c["abc"], "%s deveria herdar de ABC" % c["name"])
        t.expect(c["abstract"], "instanciar %s deveria dar TypeError" % c["name"])

    @t.case("a capacidade NÃO herda de Creature")
    def _():
        part("cap", "TransformCapability")
        t.expect(not info()["cap_is_creature"], "TransformCapability e Creature não deveriam herdar uma da outra")

    @t.case("CreatureFactory: ABC com create_base e create_evolved abstratos")
    def _():
        c = part("factory", "a fábrica abstrata com create_base e create_evolved abstratos")
        t.expect(c["abc"], "%s deveria herdar de ABC" % c["name"])
        t.expect(c["abstract"], "instanciar %s deveria dar TypeError" % c["name"])

    def facts():
        got = info()["factories"]
        t.expect(len(got) >= 2, "ex3 deveria expor pelo menos 2 fábricas concretas, expõe %s"
                 % show([f["name"] for f in got]))
        return got

    t.case("ex3 expõe as duas fábricas concretas", facts)
    t.case("ex3 NÃO expõe as entidades concretas",
           lambda: t.expect(not info()["exposed"], "ex3 expõe %s: só as fábricas deveriam aparecer"
                            % show(info()["exposed"])))

    def ents():
        return [(f["name"], k, f[k]) for f in facts() for k in ("base", "evolved")]

    @t.case("toda entidade criada é uma Creature")
    def _():
        bad = ["%s.create_%s" % (n, k) for n, k, e in ents() if not e["creature"]]
        t.expect(not bad, "%s não retorna uma Creature" % show(bad))

    @t.case("base e evoluída são classes diferentes")
    def _():
        bad = [f["name"] for f in facts() if f["base"]["cls"] == f["evolved"]["cls"]]
        t.expect(not bad, "%s cria a mesma classe nos dois métodos" % show(bad))

    t.case("cada create_* cria uma instância nova",
           lambda: t.expect(all(e["fresh"] for _, _, e in ents()), "create_* devolve sempre o mesmo objeto"))

    @t.case("uma família simples e uma família com transform (sem misturar)")
    def _():
        caps = [(f["base"]["cap"], f["evolved"]["cap"]) for f in facts()]
        t.expect((False, False) in caps, "nenhuma fábrica só de entidades simples (sem a capacidade)")
        t.expect((True, True) in caps, "nenhuma fábrica de entidades com transform (base e evoluída)")
        t.expect(all(a == b for a, b in caps), "uma fábrica mistura entidade com e sem capacidade: %s" % show(caps))

    t.case("describe e attack retornam texto",
           lambda: t.expect(all(e["describe"] and e["attack"] for _, _, e in ents()),
                            "describe/attack vazio em %s" % show([n for n, _, e in ents()
                                                                  if not (e["describe"] and e["attack"])])))

    def transformers():
        got = [e for _, _, e in ents() if e["cap"]]
        t.expect(got, "nenhuma entidade com a capacidade de transformar")
        return got

    @t.case("transform muda o attack, revert volta ao normal")
    def _():
        for e in transformers():
            t.expect(e["attack2"] != e["attack"], "%s: attack depois do transform deveria mudar, veio %s"
                     % (e["cls"], show(e["attack2"])))
            t.expect(e["attack3"] == e["attack"], "%s: depois do revert o attack deveria voltar a %s, veio %s"
                     % (e["cls"], show(e["attack"]), show(e["attack3"])))

    t.case("um atributo guarda o estado transformado",
           lambda: t.expect(all(e["state"] for e in transformers()), "transform não muda nenhum atributo"))

    for cls, want in KNOWN.items():
        @t.case("%s: mensagens do exemplo" % cls)
        def _(cls=cls, want=want):
            found = [e for _, _, e in ents() if e["cls"] == cls]
            if not found:
                t.skip("sem %s (o subject permite outros nomes)" % cls)
            e = found[0]
            got = (e["describe"], e["attack"]) + ((e["transform"], e["attack2"], e["revert"]) if len(want) > 2 else ())
            t.expect([norm(x) for x in got] == [norm(x) for x in want], "esperado %s, veio %s"
                     % (show(list(want)), show(list(got))))

    for rel in (BATTLE, CAPACITOR, TOURNAMENT):
        @t.case("%s usa as fábricas, não as classes concretas" % rel)
        def _(rel=rel):
            t.need(rel)
            bad = sorted(set(info()["entities"]) & names_used(t, rel))
            t.expect(not bad, "%s cita %s direto" % (rel, show(bad)))

    @t.case("python3 battle.py: cada fábrica cria, descreve e ataca base e evoluída")
    def _():
        out = t.run(BATTLE).out
        for f in facts():
            b, e = f["base"], f["evolved"]
            has_lines(t, out, [b["describe"], b["attack"], e["describe"], e["attack"]], f["name"])

    @t.case("python3 battle.py: as bases de duas fábricas lutam")
    def _():
        out = t.run(BATTLE).out
        bases = [f["base"] for f in facts()]
        for a in bases:
            for b in bases:
                if a is not b:
                    try:
                        has_lines(t, out, [a["describe"], b["describe"], a["attack"], b["attack"]], "luta")
                        return
                    except Exception:
                        pass
        t.expect(False, "esperado uma luta: describe das duas bases, depois os ataques. Saída: %s" % show(out))

    @t.case("python3 capacitor.py: describe, attack, transform, attack, revert de cada uma")
    def _():
        out = t.run(CAPACITOR).out
        fam = [f for f in facts() if f["base"]["cap"] and f["evolved"]["cap"]]
        t.expect(fam, "nenhuma fábrica com transform na base e na evoluída")
        seq = [x for k in ("base", "evolved")
               for x in (fam[0][k][y] for y in ("describe", "attack", "transform", "attack2", "revert"))]
        has_lines(t, out, seq, CAPACITOR)


# ---- ex4: strategies ---------------------------------------------------------

EX4 = r'''
cs = classes('ex3')
C = root(cs, 'attack')
Cap = root(cs, 'transform', 'revert')
F = root(cs, 'create_base', 'create_evolved')
ex3 = load('ex3')
facts = [quiet(v) for n, v in sorted(vars(ex3).items())
         if inspect.isclass(v) and F and issubclass(v, F) and not inspect.isabstract(v)]
plain = [f for f in facts if not isinstance(quiet(f.create_base), Cap)][0]
morph = [f for f in facts if isinstance(quiet(f.create_base), Cap)][0]
cs4 = classes('ex4')
B = root(cs4, 'act', 'is_valid')
by = {c.__name__: c for c in cs4}
N, A = by.get('NormalStrategy'), by.get('AggressiveStrategy')
'''

EX4_INFO = EX4 + r'''
r = {'base': B and {'name': B.__name__, 'abc': abc.ABC in B.__mro__, 'abstract': abstract(B)},
     'N': bool(N and B and issubclass(N, B) and not inspect.isabstract(N)),
     'A': bool(A and B and issubclass(A, B) and not inspect.isabstract(A))}
p, m, me = quiet(plain.create_base), quiet(morph.create_base), quiet(morph.create_evolved)
r['plain'] = {'cls': type(p).__name__, 'attack': text(p.attack)}
r['morph'] = {'cls': type(m).__name__, 'attack': text(m.attack)}
r['valid'] = [quiet(s().is_valid, e) for s, e in ((N, p), (N, m), (A, m), (A, me), (A, p))]
r['n_plain'] = text(N().act, p)
r['n_morph'] = text(N().act, m)
r['a_morph'] = text(A().act, m)
r['after'] = text(m.attack)
mt = quiet(morph.create_base)
r['morph_msgs'] = [text(mt.transform), text(mt.attack), text(mt.revert)]
try:
    quiet(A().act, p)
    r['err'] = None
except Exception as e:
    r['err'] = [type(e).__name__, type(e).__module__, str(e)]
log = []
class LTPlain(C):
    def attack(self):
        log.append('attack')
        return 'LT attack'
class LTMorph(C, Cap):
    def attack(self):
        log.append('attack')
        return 'LT attack'
    def transform(self):
        log.append('transform')
        return 'LT transform'
    def revert(self):
        log.append('revert')
        return 'LT revert'
class LTDuck:
    name = 'LTDuck'
    def attack(self):
        return 'LT attack'
    def transform(self):
        return 'LT transform'
    def revert(self):
        return 'LT revert'
def clone(cls, src):
    o = object.__new__(cls)
    o.__dict__.update(vars(src))
    return o
lp, lm = clone(LTPlain, p), clone(LTMorph, m)
r['lt_valid'] = [quiet(A().is_valid, lm), quiet(A().is_valid, lp), quiet(N().is_valid, lp)]
quiet(A().act, lm)
r['lt_a'], log[:] = list(log), []
quiet(N().act, lm)
r['lt_n'], log[:] = list(log), []
try:
    quiet(A().act, lp)
except Exception:
    pass
r['lt_bad'] = list(log)
r['duck'] = quiet(A().is_valid, LTDuck())
done(r)
'''

TOUR = EX4 + r'''
import ast
tree = ast.parse(open('tournament.py').read())
fs = [n.name for n in tree.body if isinstance(n, ast.FunctionDef) and len(n.args.args) == 1]
pick = [n for n in fs if n in ('battle', 'tournament')] or fs
T = load('tournament')
b = getattr(T, pick[0]) if len(pick) >= 1 else None
r = {'fn': pick, 'raised': None}
if b:
    try:
        r['out'] = cap(b, %s)[1]
    except Exception as e:
        r['raised'] = str(e)
        r['out'] = ''
done(r)
'''


def ex4(t):
    t.group("ex4 estratégias")

    def info():
        return value(t, EX4_INFO, files=("ex4/__init__.py", "ex3/__init__.py"))

    t.case("`import ex4` e as estratégias funcionam", info)

    @t.case("BattleStrategy: ABC com act e is_valid abstratos")
    def _():
        c = info()["base"]
        t.expect(c, "não achei a classe abstrata com act e is_valid abstratos (BattleStrategy)")
        t.expect(c["abc"], "%s deveria herdar de ABC" % c["name"])
        t.expect(c["abstract"], "instanciar %s deveria dar TypeError" % c["name"])

    t.case("NormalStrategy herda de BattleStrategy e é concreta",
           lambda: t.expect(info()["N"], "NormalStrategy deveria herdar de BattleStrategy e implementar tudo"))
    t.case("AggressiveStrategy herda de BattleStrategy e é concreta",
           lambda: t.expect(info()["A"], "AggressiveStrategy deveria herdar de BattleStrategy e implementar tudo"))

    labels = ["Normal.is_valid(simples) é True", "Normal.is_valid(com transform) é True",
              "Aggressive.is_valid(base com transform) é True", "Aggressive.is_valid(evoluída com transform) é True",
              "Aggressive.is_valid(simples) é False"]
    wants = [True, True, True, True, False]
    for i, (label, want) in enumerate(zip(labels, wants)):
        t.case(label, lambda i=i, want=want: t.expect(info()["valid"][i] is want, "esperado %s, veio %s"
                                                      % (want, show(info()["valid"][i]))))

    t.case("Normal.act numa entidade simples só ataca",
           lambda: t.expect(norm(info()["plain"]["attack"]) in norm(info()["n_plain"]),
                            "esperado %s, veio %s" % (show(info()["plain"]["attack"]), show(info()["n_plain"]))))

    @t.case("Normal.act numa entidade com transform só ataca (sem transformar)")
    def _():
        got = info()["n_morph"]
        t.expect(norm(info()["morph"]["attack"]) in norm(got) and norm(info()["morph_msgs"][0]) not in norm(got),
                 "esperado só %s, veio %s" % (show(info()["morph"]["attack"]), show(got)))

    @t.case("Aggressive.act: transform, attack, revert")
    def _():
        has_lines(t, info()["a_morph"], info()["morph_msgs"], "Aggressive.act")

    t.case("depois do Aggressive.act a entidade voltou ao normal",
           lambda: t.expect(info()["after"] == info()["morph"]["attack"], "attack depois do act: %s"
                            % show(info()["after"])))

    @t.case("Aggressive.act numa entidade simples levanta uma exceção própria com mensagem clara")
    def _():
        err = info()["err"]
        t.expect(err, "esperado uma exceção, não levantou nada")
        name, mod, msg = err
        t.expect(mod.split(".")[0] in ("ex3", "ex4"), "esperado uma exceção dedicada (definida no projeto), veio %s"
                 % name)
        t.expect(msg.strip(), "a exceção %s veio sem mensagem" % name)
        cls = info()["plain"]["cls"]
        t.expect(cls not in KNOWN or cls in msg, "a mensagem deveria citar %s, veio %s" % (cls, show(msg)))

    t.case("Aggressive.act inválido não chega a atacar",
           lambda: t.expect(not info()["lt_bad"], "chamou %s antes de recusar" % show(info()["lt_bad"])))

    @t.case("usa isinstance na capacidade: aceita uma entidade nova com transform")
    def _():
        t.expect(info()["lt_valid"][:2] == [True, False], "Aggressive.is_valid numa Creature nova com/sem a "
                 "capacidade: esperado [True, False], veio %s" % show(info()["lt_valid"][:2]))
        t.expect(info()["lt_a"] == ["transform", "attack", "revert"], "Aggressive.act na entidade nova: esperado "
                 "transform, attack, revert, veio %s" % show(info()["lt_a"]))

    t.case("Normal aceita qualquer Creature nova e só chama attack",
           lambda: t.expect(info()["lt_valid"][2] is True and info()["lt_n"] == ["attack"],
                            "esperado is_valid True e só attack, veio %s / %s"
                            % (show(info()["lt_valid"][2]), show(info()["lt_n"]))))

    t.case("um objeto com transform mas que não herda a capacidade não é válido",
           lambda: t.expect(info()["duck"] is False, "Aggressive.is_valid deveria usar isinstance na capacidade, "
                            "veio %s" % show(info()["duck"])))

    @t.case("ex4 não cita as entidades concretas (só a capacidade)")
    def _():
        ents = value(t, EX3, files=("ex3/__init__.py",))["entities"]
        for rel in package_files(t, "ex4"):
            if t.exists(rel):
                bad = sorted(set(ents) & names_used(t, rel))
                t.expect(not bad, "%s cita %s: deveria checar a capacidade com isinstance" % (rel, show(bad)))

    def tour(opps):
        r = value(t, TOUR % opps, files=(TOURNAMENT, "ex4/__init__.py"))
        t.expect(r["fn"], "não achei a função de batalha (uma função com um parâmetro: a lista de oponentes)")
        return r

    @t.case("tournament: cada oponente enfrenta cada outro uma vez (3 oponentes, 3 lutas)")
    def _():
        r = tour("[(plain, N()), (morph, A()), (morph, N())]")
        t.expect(r["raised"] is None, "a batalha levantou %s" % show(r["raised"]))
        out = norm(r["out"])
        i = info()
        tr, boost, rev = (norm(x) for x in i["morph_msgs"])
        for what, s in (("ataque da entidade simples", i["plain"]["attack"]), ("transform", tr), ("ataque turbinado", boost),
                        ("revert", rev), ("ataque normal da entidade com transform", i["morph"]["attack"])):
            n = out.count(norm(s))
            t.expect(n == 2, "%s (%s): esperado 2 vezes, veio %d. Saída: %s" % (what, show(s), n, show(r["out"])))

    @t.case("tournament: par entidade/estratégia inválido aborta o torneio")
    def _():
        r = tour("[(morph, A()), (plain, A()), (plain, N())]")
        i = info()
        msg = i["err"][2] if i["err"] else "LT_NO_ERROR"
        t.expect(r["raised"] == msg or norm(msg) in norm(r["out"]), "esperado a mensagem %s na saída, veio %s"
                 % (show(msg), show(r["out"] or r["raised"])))
        t.expect(norm(i["plain"]["attack"]) not in norm(r["out"]), "o torneio continuou depois do erro: %s"
                 % show(r["out"]))
        t.expect(norm(r["out"]).count(norm(i["morph_msgs"][0])) <= 1, "o torneio continuou depois do erro: %s"
                 % show(r["out"]))

    @t.case("python3 tournament.py roda e as duas famílias lutam")
    def _():
        out = t.run(TOURNAMENT).out
        f = value(t, EX3, files=("ex3/__init__.py",))["factories"]
        miss = [x["base"]["describe"] for x in f[:2] if norm(x["base"]["describe"]) not in norm(out)]
        t.expect(not miss, "esperado %s na saída, veio %s" % (show(miss), show(out)))


def suite(t):
    files = ROOT + package_files(t, "ex3") + package_files(t, "ex4")
    checks(t, files)
    ex0(t)
    ex1(t)
    ex2(t)
    ex3(t)
    ex4(t)
