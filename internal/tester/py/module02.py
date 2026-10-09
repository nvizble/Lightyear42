"""Python Module 02 (Garden Guardian): ex0..ex4, exceptions."""

import ast
import builtins
import re

from lt import common, show

EX = [
    ("ex0", "ft_first_exception"),
    ("ex1", "ft_raise_exception"),
    ("ex2", "ft_different_errors"),
    ("ex3", "ft_custom_errors"),
    ("ex4", "ft_finally_block"),
]
FILES = ["%s/%s.py" % (ex, fn) for ex, fn in EX]
EX2 = "ex2/ft_different_errors.py"

# Built-in exception classes are always allowed ("raise ValueError(...)").
EXC = {n for n in dir(builtins) if isinstance(getattr(builtins, n), type) and issubclass(getattr(builtins, n), BaseException)}
ALLOWED = {
    "ex0": {"int", "print"},
    "ex1": {"int", "print"},
    "ex2": {"print", "open", "int"},
    "ex3": {"print", "super"},
    "ex4": {"print", "super"},
}

# Runs in the student's process: imports the module silently, then call(f, *a)
# reports what f did (value or exception, plus what it printed).
PRELUDE = '''
import contextlib, io
def call(f, *a):
    buf = io.StringIO()
    try:
        with contextlib.redirect_stdout(buf):
            v = f(*a)
    except Exception as e:
        return {"ok": False, "type": type(e).__name__, "msg": str(e),
                "mro": [c.__name__ for c in type(e).__mro__], "out": buf.getvalue()}
    return {"ok": True, "value": v if isinstance(v, (int, str, type(None))) else repr(v),
            "vtype": type(v).__name__, "out": buf.getvalue()}
with contextlib.redirect_stdout(io.StringIO()):
    import %s as m
'''


def suite(t):
    # ex2 is checked by mypy separately: its TypeError line is wrong on purpose.
    common(t, [f for f in FILES if f != EX2], typed=FILES)

    def probe(ex, fn, body):
        """Runs body after importing the student's module as m; body sets R.
        Returns R."""
        code = PRELUDE % fn + body + '\nprint("@@" + repr(R))\n'
        r = t.py(code, ex=ex, files=[fn + ".py"])
        lines = [ln for ln in r.out.splitlines() if ln.startswith("@@")]
        t.expect(lines, "o teste não terminou: %s" % show(r.out + r.err))
        return ast.literal_eval(lines[-1][2:])

    def call(ex, fn, func, *args):
        return probe(ex, fn, "R = call(m.%s%s)" % (func, "".join(", %r" % (a,) for a in args)))

    def raised(r):
        return "%s(%s)" % (r["type"], show(r["msg"])) if not r["ok"] else "retornou %s" % show(r["value"])

    def run(rel):
        r = t.run(rel)
        t.expect(r.code == 0, "o programa terminou com código %d: %s" % (r.code, show(r.err[-300:])))
        t.expect(r.out.strip(), "o programa não imprimiu nada")
        return r.out

    def basics(ex, fn, names):
        rel = "%s/%s.py" % (ex, fn)

        @t.case("define %s" % ", ".join(names))
        def _():
            t.need(rel)
            t.defines(rel, *names)

        @t.case("só usa funções autorizadas (%s)" % ", ".join(sorted(ALLOWED[ex])))
        def _():
            t.need(rel)
            t.authorized(rel, allowed=ALLOWED[ex] | EXC)

        @t.case("python3 %s roda sem exceção não tratada" % rel)
        def _():
            run(rel)

    def func_node(rel, name):
        for n in ast.walk(t.tree(rel)):
            if isinstance(n, ast.FunctionDef) and n.name == name:
                return n
        t.expect(False, "não define %s()" % name)

    # ---- ex0 -------------------------------------------------------------

    def temp_common(ex, fn, rel):
        @t.case('input_temperature("25") retorna o int 25')
        def _():
            r = call(ex, fn, "input_temperature", "25")
            t.expect(r["ok"] and r["value"] == 25 and r["vtype"] == "int", "esperado 25 (int), recebido %s" % raised(r))

        for bad in ["abc", "", "12.5", "2five"]:
            @t.case("input_temperature(%s) lança uma exceção" % show(bad))
            def _(bad=bad):
                r = call(ex, fn, "input_temperature", bad)
                t.expect(not r["ok"], "esperado uma exceção (ValueError), mas %s" % raised(r))

        @t.case('a exceção de input_temperature("abc") é ValueError')
        def _():
            r = call(ex, fn, "input_temperature", "abc")
            t.expect(not r["ok"] and "ValueError" in r["mro"], "esperado ValueError, recebido %s" % raised(r))

        @t.case("test_temperature() trata o erro (não propaga exceção)")
        def _():
            r = call(ex, fn, "test_temperature")
            t.expect(r["ok"], "test_temperature() deixou escapar %s" % raised(r))
            t.expect("25" in r["out"] and "abc" in r["out"], "esperado testar '25' e 'abc', saída: %s" % show(r["out"]))

        @t.case("o programa mostra o erro de 'abc' e continua depois dele")
        def _():
            out = run(rel)
            lines = [ln for ln in out.splitlines() if ln.strip()]
            t.expect(re.search(r"invalid literal|error|erro|invalid", out, re.I), "esperado uma mensagem de erro para 'abc', saída: %s" % show(out))
            last_abc = max((i for i, ln in enumerate(lines) if "abc" in ln), default=-1)
            t.expect(last_abc >= 0, "esperado testar 'abc', saída: %s" % show(out))
            t.expect(last_abc < len(lines) - 1, "nada foi impresso depois do erro de 'abc' (o programa deve continuar), saída: %s" % show(out))

    ex, fn = EX[0]
    rel = FILES[0]
    t.group("ex0 " + fn)
    basics(ex, fn, ["input_temperature", "test_temperature"])
    temp_common(ex, fn, rel)
    for s, v in [("0", 0), ("-5", -5), ("1000", 1000)]:
        @t.case("input_temperature(%s) retorna %d" % (show(s), v))
        def _(s=s, v=v):
            r = call(ex, fn, "input_temperature", s)
            t.expect(r["ok"] and r["value"] == v, "esperado %d, recebido %s" % (v, raised(r)))

    # ---- ex1 -------------------------------------------------------------

    ex, fn = EX[1]
    rel = FILES[1]
    t.group("ex1 " + fn)
    basics(ex, fn, ["input_temperature", "test_temperature"])
    temp_common(ex, fn, rel)
    for s, v in [("0", 0), ("40", 40), ("39", 39), ("1", 1)]:
        @t.case("input_temperature(%s) retorna %d (limites 0..40 incluídos)" % (show(s), v))
        def _(s=s, v=v):
            r = call(ex, fn, "input_temperature", s)
            t.expect(r["ok"] and r["value"] == v, "esperado %d, recebido %s" % (v, raised(r)))
    for s in ["41", "100", "-1", "-50", "99999"]:
        @t.case("input_temperature(%s) lança uma exceção (fora de 0..40)" % show(s))
        def _(s=s):
            r = call(ex, fn, "input_temperature", s)
            t.expect(not r["ok"], "esperado uma exceção, mas %s" % raised(r))
            t.expect(r["msg"].strip(), "a exceção não tem mensagem (%s)" % r["type"])

    @t.case("mensagens diferentes para quente demais e frio demais")
    def _():
        hot = call(ex, fn, "input_temperature", "100")
        cold = call(ex, fn, "input_temperature", "-50")
        t.expect(not hot["ok"] and not cold["ok"], "esperado exceção para '100' e '-50', recebido %s e %s" % (raised(hot), raised(cold)))
        t.expect(hot["msg"] != cold["msg"], "a mesma mensagem para 100 e -50: %s" % show(hot["msg"]))

    @t.case("test_temperature() testa '25', 'abc', '100' e '-50' sem propagar exceção")
    def _():
        r = call(ex, fn, "test_temperature")
        t.expect(r["ok"], "test_temperature() deixou escapar %s" % raised(r))
        missing = [s for s in ["25", "abc", "100", "-50"] if s not in r["out"]]
        t.expect(not missing, "não testa %s, saída: %s" % (", ".join(missing), show(r["out"])))

    @t.case("o programa testa os 4 valores na ordem e continua depois do último erro")
    def _():
        out = run(rel)
        pos = [out.find(s) for s in ["25", "abc", "100", "-50"]]
        t.expect(-1 not in pos and pos == sorted(pos), "esperado '25', 'abc', '100', '-50' nessa ordem, saída: %s" % show(out))
        lines = [ln for ln in out.splitlines() if ln.strip()]
        last = max(i for i, ln in enumerate(lines) if "-50" in ln)
        t.expect(last < len(lines) - 1, "nada foi impresso depois do erro de '-50', saída: %s" % show(out))

    # ---- ex2 -------------------------------------------------------------

    ex, fn = EX[2]
    rel = EX2
    t.group("ex2 " + fn)
    basics(ex, fn, ["garden_operations", "test_error_types"])
    KINDS = [
        (0, "ValueError", r"ValueError|invalid literal"),
        (1, "ZeroDivisionError", r"ZeroDivisionError|division by zero|by zero"),
        (2, "FileNotFoundError", r"FileNotFoundError|No such file"),
        (3, "TypeError", r"TypeError|concatenate|unsupported operand"),
    ]
    for n, name, _ in KINDS:
        @t.case("garden_operations(%d) lança %s" % (n, name))
        def _(n=n, name=name):
            r = call(ex, fn, "garden_operations", n)
            t.expect(not r["ok"] and r["type"] == name, "esperado %s, recebido %s" % (name, raised(r)))
    for n in [4, 5, 42, -1]:
        @t.case("garden_operations(%d) não lança nada" % n)
        def _(n=n):
            r = call(ex, fn, "garden_operations", n)
            t.expect(r["ok"], "esperado retornar normalmente, recebido %s" % raised(r))

    @t.case("garden_operations usa código com defeito, não raise")
    def _():
        node = func_node(rel, "garden_operations")
        raises = [n.lineno for n in ast.walk(node) if isinstance(n, ast.Raise)]
        t.expect(not raises, "garden_operations tem raise na linha %s (o erro deve vir do código com defeito)" % ", ".join(map(str, raises)))

    @t.case("pega vários tipos de erro num mesmo try")
    def _():
        t.need(rel)
        ok = any(isinstance(n, ast.Try) and (len(n.handlers) >= 2 or any(isinstance(h.type, ast.Tuple) for h in n.handlers))
                 for n in ast.walk(t.tree(rel)))
        t.expect(ok, "nenhum try com except (A, B) ou com vários except")

    def all_kinds(out):
        missing = [name for _, name, pat in KINDS if not re.search(pat, out)]
        t.expect(not missing, "não mostra %s, saída: %s" % (", ".join(missing), show(out)))

    @t.case("test_error_types() pega os 4 erros sem propagar exceção")
    def _():
        r = call(ex, fn, "test_error_types")
        t.expect(r["ok"], "test_error_types() deixou escapar %s" % raised(r))
        all_kinds(r["out"])

    @t.case("o programa mostra os 4 tipos de erro e continua até o fim")
    def _():
        out = run(rel)
        all_kinds(out)
        lines = [ln for ln in out.splitlines() if ln.strip()]
        last = max(i for i, ln in enumerate(lines) if re.search(KINDS[3][2], ln))
        t.expect(last < len(lines) - 1, "nada foi impresso depois do TypeError, saída: %s" % show(out))

    t.group("flake8")

    @t.case("o código segue o flake8: " + EX2)
    def _():
        t.need(EX2)
        r = t.tool("flake8", [EX2])
        t.expect(r.code == 0, (r.out.strip() or r.err.strip())[:1500])

    t.group("mypy")

    @t.case("o mypy não acusa erros fora de garden_operations: " + EX2)
    def _():
        t.need(EX2)
        node = func_node(EX2, "garden_operations")
        lo, hi = node.lineno, node.end_lineno
        r = t.tool("mypy", ["--ignore-missing-imports", "--no-error-summary", "--cache-dir", "/dev/null", EX2])
        bad = []
        for ln in r.out.splitlines():
            m = re.match(r"[^:]+:(\d+): error:", ln)
            if m and not lo <= int(m.group(1)) <= hi:
                bad.append(ln)
        t.expect(not bad, "\n".join(bad)[:1500])

    t.group("sintaxe")
    if t.exists(EX2):
        t.case(EX2, lambda: t.tree(EX2))
    t.group("entrega")
    t.case(EX2, lambda: t.need(EX2))

    # ---- ex3 -------------------------------------------------------------

    ex, fn = EX[3]
    rel = FILES[3]
    t.group("ex3 " + fn)
    basics(ex, fn, ["GardenError", "PlantError", "WaterError"])
    for sub, base, want in [
        ("GardenError", "Exception", True),
        ("PlantError", "GardenError", True),
        ("WaterError", "GardenError", True),
        ("PlantError", "WaterError", False),
        ("WaterError", "PlantError", False),
    ]:
        @t.case("%s %sherda de %s" % (sub, "" if want else "não ", base))
        def _(sub=sub, base=base, want=want):
            b = "Exception" if base == "Exception" else "m." + base
            got = probe(ex, fn, "R = issubclass(m.%s, %s)" % (sub, b))
            t.expect(got == want, "issubclass(%s, %s): esperado %s, recebido %s" % (sub, base, want, got))

    for cls in ["GardenError", "PlantError", "WaterError"]:
        @t.case("%s() sem argumento tem uma mensagem padrão" % cls)
        def _(cls=cls):
            r = probe(ex, fn, "R = call(lambda: str(m.%s()))" % cls)
            t.expect(r["ok"], "%s() falhou: %s" % (cls, raised(r)))
            t.expect(r["value"].strip(), "str(%s()) é vazio, esperado uma mensagem padrão" % cls)

        @t.case("%s(\"msg\") guarda a mensagem dada" % cls)
        def _(cls=cls):
            r = probe(ex, fn, "R = call(lambda: str(m.%s('The tomato plant is wilting!')))" % cls)
            t.expect(r["ok"] and r["value"] == "The tomato plant is wilting!",
                     "esperado %s, recebido %s" % (show("The tomato plant is wilting!"), raised(r)))

    @t.case("PlantError e WaterError têm mensagens padrão próprias")
    def _():
        r = probe(ex, fn, "R = call(lambda: (str(m.PlantError()), str(m.WaterError())))")
        t.expect(r["ok"], "falhou: %s" % raised(r))
        p, w = ast.literal_eval(r["value"])
        t.expect(p != w, "as duas têm a mesma mensagem padrão: %s" % show(p))

    @t.case("except GardenError pega PlantError e WaterError")
    def _():
        body = (
            "R = []\n"
            "for c in (m.PlantError, m.WaterError):\n"
            "    try:\n"
            "        raise c('x')\n"
            "    except m.GardenError:\n"
            "        R.append(c.__name__)\n"
        )
        got = probe(ex, fn, body)
        t.expect(got == ["PlantError", "WaterError"], "esperado pegar as duas, pegou %s" % got)

    @t.case("o arquivo lança PlantError e WaterError e usa except GardenError")
    def _():
        tree = t.tree(rel)

        def name(n):
            n = n.func if isinstance(n, ast.Call) else n
            return n.id if isinstance(n, ast.Name) else n.attr if isinstance(n, ast.Attribute) else None

        raised_ = {name(n.exc) for n in ast.walk(tree) if isinstance(n, ast.Raise) and n.exc is not None}
        caught = set()
        for h in (n for n in ast.walk(tree) if isinstance(n, ast.ExceptHandler) and n.type is not None):
            caught |= {name(e) for e in (h.type.elts if isinstance(h.type, ast.Tuple) else [h.type])}
        missing = [c for c in ["PlantError", "WaterError"] if c not in raised_]
        t.expect(not missing, "nenhum raise de %s" % ", ".join(missing))
        t.expect("GardenError" in caught, "nenhum except GardenError (mostrar que ele pega todos os erros do jardim)")

    @t.case("o programa mostra PlantError, WaterError e GardenError")
    def _():
        out = run(rel)
        missing = [c for c in ["PlantError", "WaterError", "GardenError"] if c not in out]
        t.expect(not missing, "não mostra %s, saída: %s" % (", ".join(missing), show(out)))

    # ---- ex4 -------------------------------------------------------------

    ex, fn = EX[4]
    rel = FILES[4]
    t.group("ex4 " + fn)
    basics(ex, fn, ["water_plant", "test_watering_system"])

    @t.case("PlantError existe e herda de Exception")
    def _():
        got = probe(ex, fn, "R = isinstance(getattr(m, 'PlantError', None), type) and issubclass(m.PlantError, Exception)")
        t.expect(got, "o arquivo não tem uma classe PlantError que herda de Exception")

    for name in ["Tomato", "Lettuce", "Carrots", "Basil"]:
        @t.case("water_plant(%s) rega e imprime uma mensagem" % show(name))
        def _(name=name):
            r = call(ex, fn, "water_plant", name)
            t.expect(r["ok"], "esperado sucesso, recebido %s" % raised(r))
            t.expect(name in r["out"], "esperado uma mensagem com %s, saída: %s" % (show(name), show(r["out"])))
    for name in ["lettuce", "tomato", "carrots"]:
        @t.case("water_plant(%s) lança PlantError com o nome" % show(name))
        def _(name=name):
            r = call(ex, fn, "water_plant", name)
            t.expect(not r["ok"] and "PlantError" in r["mro"], "esperado PlantError, recebido %s" % raised(r))
            t.expect(name in r["msg"], "a mensagem não diz qual planta: %s" % show(r["msg"]))

    @t.case("test_watering_system usa try/except/finally")
    def _():
        node = func_node(rel, "test_watering_system")
        tries = [n for n in ast.walk(node) if isinstance(n, ast.Try)]
        t.expect(any(n.finalbody for n in tries), "test_watering_system não tem bloco finally")
        t.expect(any(n.handlers for n in tries), "test_watering_system não tem except")

    # Swaps water_plant for a fake (fails on non-capitalized names) and calls test_watering_system with the
    # plant list if it takes one.
    FAKE = '''
import inspect
calls = []
def fake(name):
    calls.append(name)
    if FAIL and name != name.capitalize():
        raise m.PlantError("Invalid plant name to water: 'lettuce'")
m.water_plant = fake
ps = [p for p in inspect.signature(m.test_watering_system).parameters.values()
      if p.default is p.empty and p.kind in (p.POSITIONAL_ONLY, p.POSITIONAL_OR_KEYWORD)]
R = call(m.test_watering_system, *([PLANTS] * len(ps)))
R["calls"] = calls
R["nparams"] = len(ps)
'''

    def fake_run(fail):
        plants = ["Tomato", "lettuce", "Carrots"] if fail else ["Tomato", "Lettuce", "Carrots"]
        r = probe(ex, fn, "FAIL = %s\nPLANTS = %r\n" % (fail, plants) + FAKE)
        if r["nparams"] > 1:
            t.skip("test_watering_system recebe %d parâmetros; não sei chamá-la" % r["nparams"])
        if not r["ok"] and not r["calls"] and r["type"] in ("TypeError", "AttributeError"):
            t.skip("não consegui chamar test_watering_system: %s" % raised(r))
        return r

    @t.case("test_watering_system trata o PlantError e para no primeiro erro")
    def _():
        r = fake_run(True)
        t.expect(r["ok"], "test_watering_system deixou escapar %s" % raised(r))
        bad = [i for i, c in enumerate(r["calls"]) if c != c.capitalize()]
        if not bad:
            t.skip("test_watering_system não rega nenhum nome inválido")
        t.expect(bad[0] == len(r["calls"]) - 1, "depois do erro em %s continuou regando: %s" % (show(r["calls"][bad[0]]), show(r["calls"][bad[0] + 1:])))

    @t.case("o finally fecha o sistema mesmo quando há erro")
    def _():
        good = fake_run(False)
        bad = fake_run(True)
        good_lines = [ln for ln in good["out"].splitlines() if ln.strip()]
        bad_lines = [ln for ln in bad["out"].splitlines() if ln.strip()]
        t.expect(good_lines, "test_watering_system não imprime nada (deveria abrir e fechar o sistema)")
        closing = good_lines[-1]
        ok = closing in bad_lines or any(re.search(r"clos|fech|shut", ln, re.I) for ln in bad_lines)
        t.expect(ok, "sem mensagem de fechamento quando há erro; sem erro termina com %s, com erro imprimiu %s"
                 % (show(closing), show(bad["out"])))

    @t.case("o programa rega as plantas válidas, mostra o PlantError e fecha o sistema")
    def _():
        out = run(rel)
        t.expect("Tomato" in out, "esperado regar 'Tomato', saída: %s" % show(out))
        t.expect("PlantError" in out or re.search(r"invalid|error|erro", out, re.I),
                 "esperado mostrar o PlantError da planta inválida, saída: %s" % show(out))
        lines = [ln for ln in out.splitlines() if ln.strip()]
        opens = sum(1 for ln in lines if re.search(r"open|abr", ln, re.I))
        closes = sum(1 for ln in lines if re.search(r"clos|fech|shut", ln, re.I))
        t.expect(closes >= opens,
                 "esperado fechar o sistema toda vez que abre (abriu %d, fechou %d), saída: %s" % (opens, closes, show(out)))
