"""Python Module 05 (Code Nexus): abstract classes, polymorphism, a data
processing pipeline."""

import ast
import builtins
import json
import re

from lt import common, show

EX0 = "ex0/data_processor.py"
EX1 = "ex1/data_stream.py"
EX2 = "ex2/data_pipeline.py"
FILES = [EX0, EX1, EX2]
PROCS = ["NumericProcessor", "TextProcessor", "LogProcessor"]
ALL_BUILTINS = set(dir(builtins))   # "All built-in functions are authorized"

LOG1 = {"log_level": "NOTICE", "log_message": "Connection to server"}
LOG2 = {"log_level": "ERROR", "log_message": "Unauthorized access!!"}
WARN = {"log_level": "WARNING", "log_message": "Telnet access! Use ssh instead"}
INFO = {"log_level": "INFO", "log_message": "User wil is connected"}
BATCH1 = ["Hello world", [3.14, -1, 2.71], [WARN, INFO], 42, ["Hi", "five"]]
BATCH2 = [21, ["I love AI", "LLMs are wonderful", "Stay healthy"],
          [{"log_level": "ERROR", "log_message": "500 server crash"},
           {"log_level": "NOTICE", "log_message": "Certificate expires in 10 days"}],
          [32, 42, 64, 84, 128, 168], "World hello"]

TEXTS = {"Hello world", "Hi", "five", "I love AI", "LLMs are wonderful", "Stay healthy", "World hello"}

# Runs in the student's process. The module is imported with stdout silenced
# (its demo may run on import when there is no __main__ guard).
PRE = r'''
import abc, ast, contextlib, inspect, io, typing
with contextlib.redirect_stdout(io.StringIO()):
    import %(mod)s as M

def safe(v):
    try:
        return ast.literal_eval(repr(v))
    except Exception:
        return repr(v)

def call(f, *a):
    b = io.StringIO()
    try:
        with contextlib.redirect_stdout(b):
            v = f(*a)
    except Exception as e:
        return {"ok": False, "type": type(e).__name__, "msg": str(e), "out": b.getvalue()}
    return {"ok": True, "value": safe(v), "out": b.getvalue()}

def outs(p, n):
    """n results of p.output(), stopping at the first exception."""
    r = []
    for _ in range(n):
        x = call(p.output)
        if not x["ok"]:
            r.append("exceção %%s(%%s)" %% (x["type"], x["msg"]))
            break
        r.append(x["value"])
    return r

def stats(ds):
    return call(ds.print_processors_stats)

def done(**kw):
    print("@@" + repr(kw))
'''


def probe(t, ex, mod, body):
    """Runs body (after PRE) and returns the dict it passed to done()."""
    r = t.py(PRE % {"mod": mod} + body, ex=ex, files=(mod + ".py",))
    for line in reversed(r.out.splitlines()):
        if line.startswith("@@"):
            return ast.literal_eval(line[2:])
    t.expect(False, "o teste não produziu resultado (saída: %s)" % show((r.out + r.err)[-300:]))


def raised(r):
    return "%s(%s)" % (r["type"], show(r["msg"])) if not r["ok"] else "retornou %s" % show(r["value"])


def ints(s):
    return [int(x) for x in re.findall(r"(?<![\d.])\d+(?![\d.])", s)]


def log_ok(v, d):
    """The stored log string has the level, then the message."""
    lv, msg = d["log_level"], d["log_message"]
    return isinstance(v, str) and lv in v and msg in v and v.index(lv) < v.index(msg)


def expect_outs(t, got, want, what):
    t.expect(got == want, "%s: esperado %s, recebido %s" % (what, show(want), show(got)))


def expect_logs(t, got, ranks, logs, what):
    ok = (len(got) == len(logs) and all(isinstance(g, tuple) and len(g) == 2 and g[0] == r and log_ok(g[1], d)
                                        for g, r, d in zip(got, ranks, logs)))
    want = [(r, "%s: %s" % (d["log_level"], d["log_message"])) for r, d in zip(ranks, logs)]
    t.expect(ok, "%s: esperado algo como %s, recebido %s" % (what, show(want), show(got)))


def stat_block(out, key):
    """The stats lines about one processor (key: numeric/text/log)."""
    lines = out.splitlines()
    for i, ln in enumerate(lines):
        if key in ln.lower():
            block = [ln]
            for nxt in lines[i + 1:]:
                if not nxt.strip() or any(k in nxt.lower() for k in ("numeric", "text", "log")):
                    break
                block.append(nxt)
            return " ".join(block)
    return None


def check_stats(t, out, key, total, remaining):
    b = stat_block(out, key)
    t.expect(b is not None, "as estatísticas não falam do processador %s: %s" % (key, show(out)))
    ns = ints(b)
    ok = any(ns[i] == total and remaining in ns[i + 1:] for i in range(len(ns)))
    t.expect(ok, "estatística de %s: esperado total %d e restante %d, recebido %s" % (key, total, remaining, show(b)))


def run_ok(t, rel):
    r = t.run(rel)
    t.expect(r.code == 0, "o programa terminou com código %d: %s" % (r.code, show(r.err[-300:])))
    t.expect(r.out.strip(), "o programa não imprimiu nada")
    return r.out


def annotation(t, rel, cls, meth, arg=1):
    """Text of the annotation of a method parameter (arg index, self = 0) or of
    the return (arg=None), lowercased, module-level type aliases expanded."""
    tree = t.tree(rel)
    aliases = {}
    for n in tree.body:
        if isinstance(n, ast.Assign) and len(n.targets) == 1 and isinstance(n.targets[0], ast.Name):
            aliases[n.targets[0].id] = ast.unparse(n.value)
        elif isinstance(n, ast.AnnAssign) and isinstance(n.target, ast.Name) and n.value is not None:
            aliases[n.target.id] = ast.unparse(n.value)
        elif type(n).__name__ == "TypeAlias":
            aliases[n.name.id] = ast.unparse(n.value)
    for c in tree.body:
        if isinstance(c, ast.ClassDef) and c.name == cls:
            for f in c.body:
                if isinstance(f, ast.FunctionDef) and f.name == meth:
                    if arg is None:
                        node = f.returns
                    else:
                        args = f.args.posonlyargs + f.args.args
                        t.expect(len(args) > arg, "%s.%s não recebe o parâmetro data" % (cls, meth))
                        node = args[arg].annotation
                    if node is None:
                        return ""
                    s = ast.unparse(node)
                    for _ in range(3):
                        s = re.sub(r"\b\w+\b", lambda m: "(%s)" % aliases[m.group(0)] if m.group(0) in aliases else m.group(0), s)
                    return s.lower().replace(" ", "")
            t.expect(False, "%s não define %s()" % (cls, meth))
    t.expect(False, "não define a classe %s" % cls)


# ---------------------------------------------------------------- shared

def processor_checks(t, ex, mod, rel, full):
    """Abstract base + the three processors (full on ex0, a summary after)."""

    def p(body):
        return probe(t, ex, mod, body)

    @t.case("DataProcessor herda de ABC")
    def _():
        d = p("done(ok=issubclass(M.DataProcessor, abc.ABC) or isinstance(M.DataProcessor, abc.ABCMeta))")
        t.expect(d["ok"], "DataProcessor deve herdar de ABC (class DataProcessor(ABC))")

    @t.case("DataProcessor() não pode ser instanciada (TypeError)")
    def _():
        d = p("done(r=call(M.DataProcessor))")
        t.expect(not d["r"]["ok"] and d["r"]["type"] == "TypeError",
                 "esperado TypeError ao instanciar a classe abstrata, recebido %s" % raised(d["r"]))

    @t.case("validate e ingest são @abstractmethod em DataProcessor")
    def _():
        d = p("done(a=sorted(getattr(M.DataProcessor, '__abstractmethods__', ())))")
        miss = [m for m in ("validate", "ingest") if m not in d["a"]]
        t.expect(not miss, "não são abstratos: %s (métodos abstratos: %s)" % (", ".join(miss), show(d["a"])))

    @t.case("output é um método normal (não abstrato) de DataProcessor")
    def _():
        d = p("done(has='output' in vars(M.DataProcessor) and callable(M.DataProcessor.output),"
              " a=sorted(getattr(M.DataProcessor, '__abstractmethods__', ())))")
        t.expect(d["has"], "DataProcessor não define output()")
        t.expect("output" not in d["a"], "output não deve ser abstrato (é implementado em DataProcessor)")

    for name in PROCS:
        @t.case("%s herda de DataProcessor e é instanciável sem argumentos" % name)
        def _(name=name):
            d = p("done(sub=issubclass(M.%s, M.DataProcessor), r=call(M.%s))" % (name, name))
            t.expect(d["sub"], "%s não herda de DataProcessor" % name)
            t.expect(d["r"]["ok"], "%s() falhou: %s" % (name, raised(d["r"])))

    if not full:
        @t.case("os processadores sobrescrevem validate e ingest")
        def _():
            d = p("done(miss=[c + '.' + m for c in %r for m in ('validate', 'ingest')"
                  " if getattr(getattr(M, c), m) is getattr(M.DataProcessor, m)])" % PROCS)
            t.expect(not d["miss"], "não sobrescrevem: %s" % ", ".join(d["miss"]))
        return

    for name in PROCS:
        @t.case("%s sobrescreve validate e ingest" % name)
        def _(name=name):
            d = p("done(miss=[m for m in ('validate', 'ingest')"
                  " if getattr(M.%s, m) is getattr(M.DataProcessor, m)])" % name)
            t.expect(not d["miss"], "%s não sobrescreve %s" % (name, ", ".join(d["miss"])))

    @t.case("uma subclasse sem ingest() continua abstrata")
    def _():
        d = p("class Half(M.DataProcessor):\n"
              "    def validate(self, data):\n"
              "        return True\n"
              "done(r=call(Half))")
        t.expect(not d["r"]["ok"] and d["r"]["type"] == "TypeError",
                 "esperado TypeError ao instanciar uma subclasse sem ingest(), recebido %s" % raised(d["r"]))

    @t.case("assinaturas de DataProcessor: validate(data: Any) -> bool, ingest(data: Any) -> None, output() -> tuple[int, str]")
    def _():
        got = {
            "validate(data)": (annotation(t, rel, "DataProcessor", "validate"), ("any",)),
            "validate() ->": (annotation(t, rel, "DataProcessor", "validate", None), ("bool",)),
            "ingest(data)": (annotation(t, rel, "DataProcessor", "ingest"), ("any",)),
            "ingest() ->": (annotation(t, rel, "DataProcessor", "ingest", None), ("none",)),
            "output() ->": (annotation(t, rel, "DataProcessor", "output", None), ("tuple[int,str]",)),
        }
        bad = ["%s %s" % (k, show(v)) for k, (v, ok) in got.items() if v.replace("typing.", "") not in ok]
        t.expect(not bad, "anotações diferentes do subject: %s" % ", ".join(bad))

    @t.case("validate() das subclasses mantém (data: Any) -> bool")
    def _():
        bad = []
        for c in PROCS:
            a = annotation(t, rel, c, "validate").replace("typing.", "")
            ret = annotation(t, rel, c, "validate", None)
            if a not in ("any", "object") or ret != "bool":
                bad.append("%s (data: %s) -> %s" % (c, a or "?", ret or "?"))
        t.expect(not bad, "assinatura diferente: %s" % ", ".join(bad))

    for c, groups in [
        ("NumericProcessor", [("int", "float"), ("list", "sequence")]),
        ("TextProcessor", [("str",), ("list", "sequence")]),
        ("LogProcessor", [("dict", "mapping"), ("list", "sequence")]),
    ]:
        @t.case("a assinatura de %s.ingest reflete os tipos aceitos (%s)" % (c, " e ".join(g[0] for g in groups)))
        def _(c=c, groups=groups):
            a = annotation(t, rel, c, "ingest")
            t.expect(a, "%s.ingest(data) sem anotação de tipo" % c)
            miss = [g[0] for g in groups if not any(re.search(r"\b%s\b" % w, a) for w in g)]
            t.expect(not miss, "%s.ingest(data: %s): faltam os tipos %s" % (c, a, ", ".join(miss)))

    # ---- validate -----------------------------------------------------
    table = {
        "NumericProcessor": ([42, 3.14, -1, 0, [1, 2, 3, 4, 5], [3.14, -1, 2.71], [1, 2.5]],
                             ["Hello", "42", ["Hi"], [1, "a"], LOG1, None]),
        "TextProcessor": (["Hello", "", ["Hello", "Nexus", "World"]],
                          [42, 3.14, ["a", 1], [1, 2], {"a": "b"}, None]),
        "LogProcessor": ([LOG1, [LOG1, LOG2]],
                         ["Hello", 42, ["Hello"], [LOG1, "x"], {"log_level": 1, "log_message": "x"}, None]),
    }
    for c, (good, bad) in table.items():
        for v, want in [(v, True) for v in good] + [(v, False) for v in bad]:
            @t.case("%s.validate(%s) é %s" % (c, show(v), want))
            def _(c=c, v=v, want=want):
                d = p("done(r=call(M.%s().validate, %r))" % (c, v))
                t.expect(d["r"]["ok"], "validate() não deve lançar exceção: %s" % raised(d["r"]))
                t.expect(bool(d["r"]["value"]) == want, "esperado %s, recebido %s" % (want, show(d["r"]["value"])))

    @t.case("validate() retorna um bool de verdade")
    def _():
        body = "R = []\n"
        for c, (good, bad) in table.items():
            body += "for v in %r:\n    x = M.%s().validate(v)\n    R.append((%r, v, type(x).__name__))\n" % (good + bad, c, c)
        d = p(body + "done(bad=[r for r in R if r[2] != 'bool'])")
        t.expect(not d["bad"], "validate() deve retornar True/False; ex.: %s.validate(%s) retornou %s"
                 % (d["bad"][0][0], show(d["bad"][0][1]), d["bad"][0][2]) if d["bad"] else "")

    # ---- ingest -------------------------------------------------------
    invalid = {
        "NumericProcessor": ["foo", ["a"], [1, "a"], None, LOG1],
        "TextProcessor": [42, ["a", 1], {"a": "b"}, None],
        "LogProcessor": ["Hello", 42, [LOG1, "x"], ["Hello"]],
    }
    for c, vals in invalid.items():
        for v in vals:
            @t.case("%s.ingest(%s) sem validar lança exceção" % (c, show(v)))
            def _(c=c, v=v):
                d = p("done(r=call(M.%s().ingest, %r))" % (c, v))
                t.expect(not d["r"]["ok"], "esperado uma exceção para dado inválido, mas %s" % raised(d["r"]))

    @t.case("ingest inválido não corrompe os dados já guardados")
    def _():
        d = p("n = M.NumericProcessor()\nn.ingest(7)\ncall(n.ingest, [8, 'x'])\nn.ingest(9)\n"
              "t_ = M.TextProcessor()\nt_.ingest('a')\ncall(t_.ingest, ['b', 2])\nt_.ingest('c')\n"
              "done(n=outs(n, 2), t=outs(t_, 2))")
        for k, want in (("n", ["7", "9"]), ("t", ["a", "c"])):
            got = [g[1] if isinstance(g, tuple) else g for g in d[k]]
            t.expect(got == want, "depois de um ingest inválido, esperado extrair %s, recebido %s" % (show(want), show(d[k])))

    # ---- output -------------------------------------------------------
    @t.case("exemplo: Numeric ingere [1, 2, 3, 4, 5], extrai 3 -> (0, '1'), (1, '2'), (2, '3')")
    def _():
        d = p("n = M.NumericProcessor()\nn.ingest([1, 2, 3, 4, 5])\ndone(o=outs(n, 3))")
        expect_outs(t, d["o"], [(0, "1"), (1, "2"), (2, "3")], "output()")

    @t.case("output() retorna uma tupla (int, str)")
    def _():
        d = p("n = M.NumericProcessor()\nn.ingest(42)\nv = n.output()\n"
              "done(t=type(v).__name__, n=len(v) if isinstance(v, tuple) else -1,"
              " k=[type(x).__name__ for x in v] if isinstance(v, tuple) else [])")
        t.expect(d["t"] == "tuple" and d["k"] == ["int", "str"], "esperado tuple[int, str], recebido %s %s" % (d["t"], d["k"]))

    @t.case("Numeric converte float, negativo e lista mista em str")
    def _():
        d = p("n = M.NumericProcessor()\nn.ingest([3.14, -1, 2.71])\nn.ingest(42)\nn.ingest([1, 2.5])\ndone(o=outs(n, 6))")
        expect_outs(t, d["o"], [(0, "3.14"), (1, "-1"), (2, "2.71"), (3, "42"), (4, "1"), (5, "2.5")], "output()")

    @t.case("exemplo: Text ingere ['Hello', 'Nexus', 'World'], extrai 1 -> (0, 'Hello')")
    def _():
        d = p("x = M.TextProcessor()\nx.ingest(['Hello', 'Nexus', 'World'])\ndone(o=outs(x, 1))")
        expect_outs(t, d["o"], [(0, "Hello")], "output()")

    @t.case("Text: str sozinha e lista, na ordem (FIFO)")
    def _():
        d = p("x = M.TextProcessor()\nx.ingest('Hello world')\nx.ingest(['Hi', 'five'])\ndone(o=outs(x, 3))")
        expect_outs(t, d["o"], [(0, "Hello world"), (1, "Hi"), (2, "five")], "output()")

    @t.case("exemplo: Log extrai 'NOTICE: Connection to server' e 'ERROR: Unauthorized access!!'")
    def _():
        d = p("x = M.LogProcessor()\nx.ingest(%r)\ndone(o=outs(x, 2))" % [LOG1, LOG2])
        expect_logs(t, d["o"], [0, 1], [LOG1, LOG2], "output()")

    @t.case("Log: um dict sozinho também é aceito")
    def _():
        d = p("x = M.LogProcessor()\nx.ingest(%r)\nx.ingest(%r)\ndone(o=outs(x, 2))" % (WARN, INFO))
        expect_logs(t, d["o"], [0, 1], [WARN, INFO], "output()")

    @t.case("o rank continua contando depois de extrair e ingerir de novo")
    def _():
        d = p("n = M.NumericProcessor()\nn.ingest([1, 2])\na = outs(n, 1)\nn.ingest(3)\ndone(o=a + outs(n, 2))")
        expect_outs(t, d["o"], [(0, "1"), (1, "2"), (2, "3")], "output()")

    @t.case("cada instância tem seus próprios dados")
    def _():
        d = p("a = M.NumericProcessor()\nb = M.NumericProcessor()\nx = M.TextProcessor()\n"
              "a.ingest(1)\nb.ingest(2)\nx.ingest('z')\ndone(a=outs(a, 1), b=outs(b, 1), x=outs(x, 1))")
        for k, want in (("a", (0, "1")), ("b", (0, "2")), ("x", (0, "z"))):
            t.expect(d[k] == [want], "dados compartilhados entre instâncias (atributo de classe?): esperado %s, recebido %s"
                     % (show(want), show(d[k])))


def stream_checks(t, ex, mod, rel):
    """DataStream: register_processor, process_stream, print_processors_stats."""

    def p(body):
        return probe(t, ex, mod, body)

    @t.case("DataStream define register_processor, process_stream e print_processors_stats")
    def _():
        d = p("done(miss=[m for m in ('register_processor', 'process_stream', 'print_processors_stats')"
              " if not callable(getattr(M.DataStream, m, None))])")
        t.expect(not d["miss"], "faltam os métodos: %s" % ", ".join(d["miss"]))

    @t.case("estatísticas sem nenhum processador: imprime e não quebra")
    def _():
        d = p("ds = M.DataStream()\ndone(r=stats(ds))")
        t.expect(d["r"]["ok"], "print_processors_stats() falhou: %s" % raised(d["r"]))
        t.expect(d["r"]["out"].strip(), "print_processors_stats() não imprimiu nada")

    SETUP = "ds = M.DataStream()\nn = M.NumericProcessor()\nx = M.TextProcessor()\nl = M.LogProcessor()\n"

    @t.case("só Numeric registrado: o lote do exemplo não quebra e mostra erro para cada elemento recusado")
    def _():
        d = p(SETUP + "ds.register_processor(n)\ndone(r=call(ds.process_stream, %r))" % BATCH1)
        t.expect(d["r"]["ok"], "process_stream() falhou: %s" % raised(d["r"]))
        out = d["r"]["out"]
        miss = [s for s in ("Hello world", "Telnet", "five") if s not in out]
        t.expect(not miss, "esperado uma mensagem de erro para cada elemento sem processador (%s), saída: %s"
                 % (", ".join(miss), show(out)))

    @t.case("só Numeric registrado: ele recebe 3.14, -1, 2.71, 42")
    def _():
        d = p(SETUP + "ds.register_processor(n)\ncall(ds.process_stream, %r)\ndone(o=outs(n, 4))" % BATCH1)
        expect_outs(t, d["o"], [(0, "3.14"), (1, "-1"), (2, "2.71"), (3, "42")], "Numeric output()")

    @t.case("só Numeric registrado: estatística total 4, restante 4")
    def _():
        d = p(SETUP + "ds.register_processor(n)\ncall(ds.process_stream, %r)\ndone(r=stats(ds))" % BATCH1)
        t.expect(d["r"]["ok"], "print_processors_stats() falhou: %s" % raised(d["r"]))
        check_stats(t, d["r"]["out"], "numeric", 4, 4)

    ALL = SETUP + "for q in (n, x, l):\n    ds.register_processor(q)\n"

    @t.case("com os 3 registrados, cada elemento vai para o processador certo")
    def _():
        d = p(ALL + "r = call(ds.process_stream, %r)\ndone(r=r, n=outs(n, 4), x=outs(x, 3), l=outs(l, 2))" % BATCH1)
        t.expect(d["r"]["ok"], "process_stream() falhou: %s" % raised(d["r"]))
        expect_outs(t, d["n"], [(0, "3.14"), (1, "-1"), (2, "2.71"), (3, "42")], "Numeric")
        expect_outs(t, d["x"], [(0, "Hello world"), (1, "Hi"), (2, "five")], "Text")
        expect_logs(t, d["l"], [0, 1], [WARN, INFO], "Log")

    @t.case("cenário do exemplo: estatísticas 8/8, 3/3, 2/2 e depois de consumir 3, 2, 1: 8/5, 3/1, 2/1")
    def _():
        d = p(SETUP + "ds.register_processor(n)\ncall(ds.process_stream, %r)\n"
              "ds.register_processor(x)\nds.register_processor(l)\ncall(ds.process_stream, %r)\n"
              "a = stats(ds)\nouts(n, 3)\nouts(x, 2)\nouts(l, 1)\ndone(a=a, b=stats(ds))" % (BATCH1, BATCH1))
        for k in ("a", "b"):
            t.expect(d[k]["ok"], "print_processors_stats() falhou: %s" % raised(d[k]))
        for key, tot, before, after in (("numeric", 8, 8, 5), ("text", 3, 3, 1), ("log", 2, 2, 1)):
            check_stats(t, d["a"]["out"], key, tot, before)
            check_stats(t, d["b"]["out"], key, tot, after)

    @t.case("o roteamento usa validate() do processador (polimorfismo)")
    def _():
        d = p("seen = []\n"
              "class Spy(M.TextProcessor):\n"
              "    def validate(self, data):\n"
              "        return data == 'spy'\n"
              "    def ingest(self, data):\n"
              "        seen.append(data)\n"
              "ds = M.DataStream()\nds.register_processor(Spy())\n"
              "done(r=call(ds.process_stream, ['spy', 'nope', 7]), seen=seen)")
        t.expect(d["r"]["ok"], "process_stream() falhou: %s" % raised(d["r"]))
        t.expect(d["seen"] == ["spy"], "um processador cujo validate() só aceita 'spy' recebeu %s (esperado ['spy'])"
                 % show(d["seen"]))

    @t.case("elementos que ninguém processa (None, tupla, dict qualquer) não quebram o stream")
    def _():
        d = p(ALL + "done(r=call(ds.process_stream, [None, (1, 2), {'x': 1}, 'ok']), x=outs(x, 1))")
        t.expect(d["r"]["ok"], "process_stream() falhou: %s" % raised(d["r"]))
        t.expect(d["r"]["out"].strip(), "nenhuma mensagem de erro para elementos sem processador")
        expect_outs(t, d["x"], [(0, "ok")], "Text depois dos elementos recusados")

    @t.case("process_stream([]) não quebra")
    def _():
        d = p(ALL + "done(r=call(ds.process_stream, []))")
        t.expect(d["r"]["ok"], "process_stream([]) falhou: %s" % raised(d["r"]))

    @t.case("cada DataStream tem seus próprios processadores")
    def _():
        d = p("a = M.DataStream()\nb = M.DataStream()\nn1 = M.NumericProcessor()\nn2 = M.NumericProcessor()\n"
              "a.register_processor(n1)\nb.register_processor(n2)\n"
              "a.process_stream([1])\nb.process_stream([2])\ndone(a=outs(n1, 1), b=outs(n2, 1))")
        t.expect(d["a"] == [(0, "1")] and d["b"] == [(0, "2")],
                 "registro compartilhado entre streams (atributo de classe?): n1 %s, n2 %s" % (show(d["a"]), show(d["b"])))


def static(t, rel, names):
    @t.case("define " + ", ".join(names))
    def _():
        t.need(rel)
        t.defines(rel, *names)

    @t.case("só importa abc e typing")
    def _():
        t.need(rel)
        t.authorized(rel, allowed=ALL_BUILTINS, imports={"abc", "typing"})

    @t.case("python3 %s roda sem exceção não tratada" % rel)
    def _():
        run_ok(t, rel)


# ---------------------------------------------------------------- ex0

def ex0(t):
    t.group("ex0 data_processor")
    static(t, EX0, ["DataProcessor"] + PROCS)

    @t.case("o programa testa validate (True e False) e mostra a exceção do ingest inválido")
    def _():
        out = run_ok(t, EX0)
        t.expect("True" in out and "False" in out, "esperado mostrar resultados True e False de validate: %s" % show(out))
        t.expect(re.search(r"exce|error|erro|improper|invalid", out, re.I),
                 "esperado mostrar a exceção do ingest sem validação: %s" % show(out))

    @t.case("o programa extrai dados dos 3 processadores")
    def _():
        out = run_ok(t, EX0)
        t.expect(re.search(r"numeric|num", out, re.I) and re.search(r"text", out, re.I) and re.search(r"log", out, re.I),
                 "esperado testar Numeric, Text e Log: %s" % show(out))

    processor_checks(t, "ex0", "data_processor", EX0, full=True)


# ---------------------------------------------------------------- ex1

def ex1(t):
    t.group("ex1 data_stream")
    static(t, EX1, ["DataProcessor"] + PROCS + ["DataStream"])

    @t.case("o programa mostra estatísticas antes e depois de consumir")
    def _():
        out = run_ok(t, EX1)
        t.expect(re.search(r"numeric", out, re.I), "esperado estatísticas do Numeric Processor: %s" % show(out))

    processor_checks(t, "ex1", "data_stream", EX1, full=False)
    stream_checks(t, "ex1", "data_stream", EX1)


# ---------------------------------------------------------------- ex2

def find_json(s):
    """Every JSON object/array printed in s."""
    dec, found = json.JSONDecoder(), []
    for i, ch in enumerate(s):
        if ch in "{[":
            try:
                v, _ = dec.raw_decode(s[i:])
            except ValueError:
                continue
            found.append(v)
    return found


def json_ok(v, data):
    """{"item_3": "42", ...} (keys with the rank), or a list of {rank, value}
    / [rank, value] entries."""
    if isinstance(v, dict):
        return (list(v.values()) == [x for _, x in data]
                and all(str(r) in str(k) for k, (r, _) in zip(v, data)))
    if isinstance(v, list) and len(v) == len(data):
        for e, (r, x) in zip(v, data):
            vals = list(e.values()) if isinstance(e, dict) else e if isinstance(e, list) else None
            if vals is None or x not in vals or not any(str(r) == str(y) for y in vals if y != x):
                return False
        return True
    return False


def ex2(t):
    t.group("ex2 data_pipeline")
    mod = "data_pipeline"
    static(t, EX2, ["DataProcessor"] + PROCS + ["DataStream", "ExportPlugin"])

    def p(body):
        return probe(t, "ex2", mod, body)

    @t.case("ExportPlugin herda de typing.Protocol e declara process_output")
    def _():
        d = p("done(proto=typing.Protocol in M.ExportPlugin.__mro__ and getattr(M.ExportPlugin, '_is_protocol', False),"
              " has=callable(getattr(M.ExportPlugin, 'process_output', None)))")
        t.expect(d["proto"], "ExportPlugin deve ser class ExportPlugin(Protocol)")
        t.expect(d["has"], "ExportPlugin não declara process_output")

    @t.case("process_output(self, data: list[tuple[int, str]]) -> None no ExportPlugin")
    def _():
        a = annotation(t, EX2, "ExportPlugin", "process_output").replace("typing.", "")
        r = annotation(t, EX2, "ExportPlugin", "process_output", None)
        t.expect(a == "list[tuple[int,str]]" and r == "none",
                 "esperado (data: list[tuple[int, str]]) -> None, recebido (data: %s) -> %s" % (a or "?", r or "?"))

    @t.case("DataStream define output_pipeline(nb, plugin)")
    def _():
        d = p("done(ok=callable(getattr(M.DataStream, 'output_pipeline', None)))")
        t.expect(d["ok"], "DataStream não define output_pipeline")

    PLUG = ("plugs = {}\n"
            "for k, v in vars(M).items():\n"
            "    if isinstance(v, type) and v is not M.ExportPlugin and callable(getattr(v, 'process_output', None)):\n"
            "        for w in ('csv', 'json'):\n"
            "            if w in k.lower():\n"
            "                plugs.setdefault(w, v)\n")

    def plug_out(kind, data):
        d = p(PLUG + "c = plugs.get(%r)\n"
              "done(name=c and c.__name__, r=c and call(c().process_output, %r))" % (kind, data))
        t.expect(d["name"], "nenhuma classe com 'CSV'/'JSON' no nome e um método process_output (plugin %s)" % kind.upper())
        t.expect(d["r"]["ok"], "%s().process_output falhou: %s" % (d["name"], raised(d["r"])))
        return d["r"]["out"]

    for kind in ("csv", "json"):
        @t.case("existe um plugin %s (classe com process_output)" % kind.upper())
        def _(kind=kind):
            d = p(PLUG + "done(name=plugs.get(%r) and plugs[%r].__name__)" % (kind, kind))
            t.expect(d["name"], "nenhuma classe com %s no nome e um método process_output" % show(kind.upper()))

    def csv_ok(out, data):
        joined = ",".join(x for _, x in data)
        rows = ["%d,%s" % (r, x) for r, x in data]
        lines = [ln.strip() for ln in out.splitlines()]
        return any(joined in ln for ln in lines) or all(r in lines for r in rows)

    for data in ([(0, "3.14"), (1, "-1"), (2, "2.71")],
                 [(0, "Hello world"), (1, "Hi"), (2, "five")],
                 [(3, "42")]):
        @t.case("CSV: process_output(%s) gera %s" % (show(data), show(",".join(x for _, x in data))))
        def _(data=data):
            out = plug_out("csv", data)
            t.expect(csv_ok(out, data), "esperado a linha CSV %s, saída: %s" % (show(",".join(x for _, x in data)), show(out)))

    for data in ([(3, "42"), (4, "21"), (5, "32")],
                 [(2, "ERROR: 500 server crash"), (3, "NOTICE: Certificate expires in 10 days")],
                 [(0, "Hello world")]):
        @t.case("JSON: process_output(%s) gera um JSON válido com esses valores" % show(data))
        def _(data=data):
            out = plug_out("json", data)
            want = "{%s}" % ", ".join('"item_%d": "%s"' % (r, x) for r, x in data)
            t.expect(any(json_ok(v, data) for v in find_json(out)), "esperado JSON como %s, saída: %s" % (want, show(out)))

    @t.case("JSON: aspas e barra invertida nos valores continuam gerando JSON válido")
    def _():
        data = [(0, 'say "hi"'), (1, "C:\\path")]
        out = plug_out("json", data)
        t.expect(any(json_ok(v, data) for v in find_json(out)),
                 "o JSON gerado é inválido ou perdeu os valores %s (escape \\\" e \\\\), saída: %s" % (show([x for _, x in data]), show(out)))

    SETUP = ("calls = []\n"
             "class Spy:\n"
             "    def process_output(self, data):\n"
             "        calls.append(list(data))\n"
             "ds = M.DataStream()\nn = M.NumericProcessor()\nx = M.TextProcessor()\nl = M.LogProcessor()\n"
             "for q in (n, x, l):\n    ds.register_processor(q)\n"
             "call(ds.process_stream, %r)\n" % BATCH1)

    def by_proc(t, calls):
        """Splits the plugin calls by processor (ignores empty ones)."""
        calls = [c for c in calls if c]
        got = {}
        for c in calls:
            vals = [e[1] if isinstance(e, tuple) and len(e) == 2 else e for e in c]
            k = ("numeric" if all(isinstance(v, str) and re.fullmatch(r"-?\d+(\.\d+)?", v) for v in vals)
                 else "text" if all(v in TEXTS for v in vals) else "log")
            t.expect(k not in got, "o plugin recebeu dois lotes do mesmo processador numa chamada de output_pipeline: %s" % show(calls))
            got[k] = c
        return got

    @t.case("output_pipeline(3, plugin) aceita qualquer objeto com process_output (duck typing)")
    def _():
        d = p(SETUP + "done(r=call(ds.output_pipeline, 3, Spy()), calls=calls)")
        t.expect(d["r"]["ok"], "output_pipeline falhou: %s" % raised(d["r"]))
        t.expect([c for c in d["calls"] if c], "o plugin nunca foi chamado com dados")

    @t.case("output_pipeline passa list[tuple[int, str]] ao plugin")
    def _():
        d = p(SETUP + "call(ds.output_pipeline, 3, Spy())\n"
              "done(bad=[repr(e) for c in calls for e in c if not (isinstance(e, tuple) and len(e) == 2"
              " and type(e[0]) is int and type(e[1]) is str)])")
        t.expect(not d["bad"], "esperado tuplas (int, str), recebido %s" % ", ".join(d["bad"][:3]))

    @t.case("exemplo: output_pipeline(3) exporta 3 de cada processador (Log só tem 2)")
    def _():
        d = p(SETUP + "call(ds.output_pipeline, 3, Spy())\ndone(calls=calls)")
        got = by_proc(t, d["calls"])
        expect_outs(t, got.get("numeric"), [(0, "3.14"), (1, "-1"), (2, "2.71")], "lote do Numeric")
        expect_outs(t, got.get("text"), [(0, "Hello world"), (1, "Hi"), (2, "five")], "lote do Text")
        expect_logs(t, got.get("log") or [], [0, 1], [WARN, INFO], "lote do Log")

    @t.case("exemplo: depois do CSV, estatísticas 4/1, 3/0, 2/0")
    def _():
        d = p(SETUP + "call(ds.output_pipeline, 3, Spy())\ndone(s=stats(ds))")
        t.expect(d["s"]["ok"], "print_processors_stats() falhou: %s" % raised(d["s"]))
        for key, tot, rem in (("numeric", 4, 1), ("text", 3, 0), ("log", 2, 0)):
            check_stats(t, d["s"]["out"], key, tot, rem)

    @t.case("exemplo: 2º lote e output_pipeline(5): ranks continuam (item_3..item_7, item_3..item_6, item_2..item_3)")
    def _():
        d = p(SETUP + "call(ds.output_pipeline, 3, Spy())\ndel calls[:]\n"
              "call(ds.process_stream, %r)\nr = call(ds.output_pipeline, 5, Spy())\n"
              "done(r=r, calls=calls, s=stats(ds))" % BATCH2)
        t.expect(d["r"]["ok"], "output_pipeline falhou: %s" % raised(d["r"]))
        got = by_proc(t, d["calls"])
        expect_outs(t, got.get("numeric"), [(3, "42"), (4, "21"), (5, "32"), (6, "42"), (7, "64")], "lote do Numeric")
        expect_outs(t, got.get("text"), [(3, "I love AI"), (4, "LLMs are wonderful"), (5, "Stay healthy"), (6, "World hello")],
                    "lote do Text")
        expect_logs(t, got.get("log") or [], [2, 3], BATCH2[2], "lote do Log")
        for key, tot, rem in (("numeric", 11, 3), ("text", 7, 0), ("log", 4, 0)):
            check_stats(t, d["s"]["out"], key, tot, rem)

    @t.case("output_pipeline com processadores vazios ou nb maior que o restante não quebra")
    def _():
        d = p(SETUP + "a = call(ds.output_pipeline, 100, Spy())\nb = call(ds.output_pipeline, 3, Spy())\n"
              "done(a=a, b=b, rest=[c for c in calls[3:] if c])")
        t.expect(d["a"]["ok"], "output_pipeline(100) falhou: %s" % raised(d["a"]))
        t.expect(d["b"]["ok"], "output_pipeline com tudo vazio falhou: %s" % raised(d["b"]))
        t.expect(not d["rest"], "com os processadores vazios o plugin ainda recebeu %s" % show(d["rest"]))

    @t.case("output_pipeline(0, plugin) não consome nada")
    def _():
        d = p(SETUP + "r = call(ds.output_pipeline, 0, Spy())\ndone(r=r, calls=[c for c in calls if c], o=outs(n, 1))")
        t.expect(d["r"]["ok"], "output_pipeline(0) falhou: %s" % raised(d["r"]))
        t.expect(not d["calls"], "nb=0 exportou %s" % show(d["calls"]))
        expect_outs(t, d["o"], [(0, "3.14")], "Numeric depois de output_pipeline(0)")

    @t.case("o programa exporta em CSV e em JSON válido")
    def _():
        out = run_ok(t, EX2)
        t.expect(any(isinstance(v, dict) and v or isinstance(v, list) and v for v in find_json(out)),
                 "nenhum JSON válido na saída: %s" % show(out))
        t.expect(re.search(r"^[^{}\[\]\n]*\S,\S[^{}\[\]\n]*$", out, re.M), "nenhuma linha CSV na saída: %s" % show(out))

    processor_checks(t, "ex2", mod, EX2, full=False)
    stream_checks(t, "ex2", mod, EX2)


def suite(t):
    # The subject asks ex0 to call ingest() with invalid data without
    # validating first, "leaving a mypy warning on purpose": ignore mypy
    # errors on lines that call ingest().
    tool = t.tool

    def lenient_mypy(name, args, cwd=None):
        r = tool(name, args, cwd)
        if name != "mypy" or r.code != 1:
            return r
        kept, drop = [], False
        for ln in r.out.splitlines():
            m = re.match(r"(.+?):(\d+): (error|note):", ln)
            if m and m.group(3) == "error":
                try:
                    with open(t.path(m.group(1)), encoding="utf-8") as f:
                        src = f.read().splitlines()[int(m.group(2)) - 1]
                except (OSError, IndexError):
                    src = ""
                drop = "ingest" in src
            if not drop:
                kept.append(ln)
        r.out = "\n".join(kept)
        if not any(": error:" in ln for ln in kept):
            r.code = 0
        return r

    t.tool = lenient_mypy
    common(t, FILES, typed=FILES)
    t.tool = tool
    ex0(t)
    ex1(t)
    ex2(t)
