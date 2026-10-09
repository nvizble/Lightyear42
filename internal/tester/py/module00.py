"""Python Module 00 (Growing Code): ex0..ex7, garden functions."""

import ast

from lt import common, show

EX = [
    ("ex0", "ft_hello_garden"),
    ("ex1", "ft_garden_name"),
    ("ex2", "ft_plot_area"),
    ("ex3", "ft_harvest_total"),
    ("ex4", "ft_plant_age"),
    ("ex5", "ft_water_reminder"),
    ("ex6", "ft_count_harvest_iterative"),
    ("ex6", "ft_count_harvest_recursive"),
    ("ex7", "ft_seed_inventory"),
]
FILES = ["%s/%s.py" % (ex, fn) for ex, fn in EX]

ALLOWED = {
    "ex0": {"print"},
    "ex1": {"input", "print"},
    "ex2": {"input", "int", "print"},
    "ex3": {"input", "int", "print"},
    "ex4": {"input", "int", "print"},
    "ex5": {"input", "int", "print"},
    "ex6": {"input", "int", "print", "range"},
    "ex7": {"print"},
}


def suite(t):
    common(t, FILES, typed=["ex7/ft_seed_inventory.py"])

    def call(ex, fn, stdin="", args=""):
        """Imports fn from the student's file and calls it; returns stdout."""
        code = "from %s import %s\n%s(%s)\n" % (fn, fn, fn, args)
        return t.py(code, ex=ex, stdin=stdin, files=[fn + ".py"]).out

    def exact(out, want):
        t.expect(out == want, "esperado %s, recebido %s" % (show(want), show(out)))

    def result(out, lines):
        """Lenient on the prompts: the last len(lines) lines must be lines
        (the first of them may carry the input() prompts before it)."""
        got = out.splitlines()
        n = len(lines)
        ok = len(got) >= n and got[-n].endswith(lines[0]) and got[len(got) - n + 1:] == lines[1:]
        t.expect(ok, "esperado terminar com %s, recebido %s" % (show("\n".join(lines) + "\n"), show(out)))

    def basics(ex, fn, names=None):
        rel = "%s/%s.py" % (ex, fn)

        @t.case("%s define %s()" % (fn + ".py", ", ".join(names or [fn])))
        def _():
            t.need(rel)
            t.defines(rel, *(names or [fn]))

        @t.case("%s só usa funções autorizadas (%s)" % (fn + ".py", ", ".join(sorted(ALLOWED[ex]))))
        def _():
            t.need(rel)
            t.authorized(rel, allowed=ALLOWED[ex])

        @t.case("%s não tem código fora de funções" % (fn + ".py"))
        def _():
            t.need(rel)
            bad = []
            for i, n in enumerate(t.tree(rel).body):
                if isinstance(n, (ast.FunctionDef, ast.Import, ast.ImportFrom)):
                    continue
                if i == 0 and isinstance(n, ast.Expr) and isinstance(n.value, ast.Constant) and isinstance(n.value.value, str):
                    continue  # module docstring
                bad.append("linha %d" % n.lineno)
            t.expect(not bad, "código no nível do arquivo (só a função é permitida): " + ", ".join(bad))

        @t.case("importar %s não executa nada" % (fn + ".py"))
        def _():
            r = t.py("import %s\n" % fn, ex=ex, files=[fn + ".py"], allow_crash=True)
            t.expect(not r.crashed, "importar o arquivo deu erro: %s" % r.last_error())
            t.expect(r.out == "", "importar o arquivo já imprime %s (não chame a função no arquivo)" % show(r.out))

    # ---- ex0 -------------------------------------------------------------
    t.group("ex0 ft_hello_garden")
    basics("ex0", "ft_hello_garden")
    t.case("exemplo do subject", lambda: exact(call("ex0", "ft_hello_garden"), "Hello, Garden Community!\n"))

    @t.case("não lê input")
    def _():
        exact(call("ex0", "ft_hello_garden", stdin="lixo\n"), "Hello, Garden Community!\n")

    @t.case("chamar duas vezes imprime duas vezes")
    def _():
        out = t.py("from ft_hello_garden import ft_hello_garden\nft_hello_garden()\nft_hello_garden()\n",
                   ex="ex0", files=["ft_hello_garden.py"]).out
        exact(out, "Hello, Garden Community!\n" * 2)

    # ---- ex1 -------------------------------------------------------------
    t.group("ex1 ft_garden_name")
    basics("ex1", "ft_garden_name")
    P1 = "Enter garden name: "

    def garden(name):
        return "Garden: %s\nStatus: Growing well!\n" % name

    t.case("exemplo do subject (saída exata com prompt)",
           lambda: exact(call("ex1", "ft_garden_name", "Community Garden\n"), P1 + garden("Community Garden")))
    for name in ["Rose", "Jardim do Bairro 42", "a", "  espaços  ", "Café & Flores!", "x" * 300]:
        t.case("nome %s" % show(name)[:40],
               lambda name=name: result(call("ex1", "ft_garden_name", name + "\n"), garden(name).splitlines()))
    t.case("nome vazio", lambda: result(call("ex1", "ft_garden_name", "\n"), ["Garden: ", "Status: Growing well!"]))

    # ---- ex2 -------------------------------------------------------------
    t.group("ex2 ft_plot_area")
    basics("ex2", "ft_plot_area")
    P2 = "Enter length: Enter width: "
    t.case("exemplo do subject 5 x 3 (saída exata com prompts)",
           lambda: exact(call("ex2", "ft_plot_area", "5\n3\n"), P2 + "Plot area: 15\n"))
    for a, b in [(3, 5), (1, 1), (0, 7), (7, 0), (0, 0), (12, 12), (100, 250), (123456789, 987654321)]:
        t.case("%d x %d" % (a, b),
               lambda a=a, b=b: result(call("ex2", "ft_plot_area", "%d\n%d\n" % (a, b)), ["Plot area: %d" % (a * b)]))

    # ---- ex3 -------------------------------------------------------------
    t.group("ex3 ft_harvest_total")
    basics("ex3", "ft_harvest_total")
    P3 = "Day 1 harvest: Day 2 harvest: Day 3 harvest: "
    t.case("exemplo do subject 5 + 8 + 3 (saída exata com prompts)",
           lambda: exact(call("ex3", "ft_harvest_total", "5\n8\n3\n"), P3 + "Total harvest: 16\n"))
    for v in [(0, 0, 0), (1, 2, 3), (10, 0, 5), (100, 200, 300), (999999999, 999999999, 999999999)]:
        t.case("%d + %d + %d" % v,
               lambda v=v: result(call("ex3", "ft_harvest_total", "%d\n%d\n%d\n" % v), ["Total harvest: %d" % sum(v)]))

    @t.case("soma números, não concatena strings")
    def _():
        result(call("ex3", "ft_harvest_total", "1\n2\n3\n"), ["Total harvest: 6"])

    # ---- ex4 -------------------------------------------------------------
    t.group("ex4 ft_plant_age")
    basics("ex4", "ft_plant_age")
    P4 = "Enter plant age in days: "
    READY, WAIT = "Plant is ready to harvest!", "Plant needs more time to grow."
    t.case("exemplo do subject 75 (saída exata com prompt)",
           lambda: exact(call("ex4", "ft_plant_age", "75\n"), P4 + READY + "\n"))
    t.case("exemplo do subject 45 (saída exata com prompt)",
           lambda: exact(call("ex4", "ft_plant_age", "45\n"), P4 + WAIT + "\n"))
    for age in [60, 61, 59, 0, 1, 100, 100000]:
        want = READY if age > 60 else WAIT
        label = " (limite: estritamente > 60)" if age in (60, 61) else ""
        t.case("idade %d%s" % (age, label),
               lambda age=age, want=want: result(call("ex4", "ft_plant_age", "%d\n" % age), [want]))

    # ---- ex5 -------------------------------------------------------------
    t.group("ex5 ft_water_reminder")
    basics("ex5", "ft_water_reminder")
    P5 = "Days since last watering: "
    WATER, FINE = "Water the plants!", "Plants are fine"
    t.case("exemplo do subject 4 (saída exata com prompt)",
           lambda: exact(call("ex5", "ft_water_reminder", "4\n"), P5 + WATER + "\n"))
    t.case("exemplo do subject 1 (saída exata com prompt)",
           lambda: exact(call("ex5", "ft_water_reminder", "1\n"), P5 + FINE + "\n"))
    for d in [2, 3, 0, 30, 100000]:
        want = WATER if d > 2 else FINE
        label = " (limite: estritamente > 2)" if d in (2, 3) else ""
        t.case("%d dias%s" % (d, label),
               lambda d=d, want=want: result(call("ex5", "ft_water_reminder", "%d\n" % d), [want]))

    # ---- ex6 -------------------------------------------------------------
    t.group("ex6 ft_count_harvest")
    P6 = "Days until harvest: "

    def days(n):
        return ["Day %d" % i for i in range(1, n + 1)] + ["Harvest time!"]

    for fn in ("ft_count_harvest_iterative", "ft_count_harvest_recursive"):
        basics("ex6", fn)
        t.case("%s exemplo do subject 5 (saída exata com prompt)" % fn,
               lambda fn=fn: exact(call("ex6", fn, "5\n"), P6 + "\n".join(days(5)) + "\n"))
        for n in [1, 2, 10, 42, 200]:
            t.case("%s %d dias" % (fn, n), lambda fn=fn, n=n: result(call("ex6", fn, "%d\n" % n), days(n)))

    def tree6(fn):
        t.need("ex6/%s.py" % fn)
        return t.tree("ex6/%s.py" % fn)

    def self_calls(tree):
        """Names of functions that call themselves."""
        out = set()
        for f in ast.walk(tree):
            if isinstance(f, ast.FunctionDef):
                for c in ast.walk(f):
                    if isinstance(c, ast.Call) and isinstance(c.func, ast.Name) and c.func.id == f.name:
                        out.add(f.name)
        return out

    def loops(tree):
        kinds = (ast.For, ast.While, ast.ListComp, ast.SetComp, ast.GeneratorExp, ast.DictComp)
        return [n.lineno for n in ast.walk(tree) if isinstance(n, kinds)]

    @t.case("ft_count_harvest_recursive é recursiva (uma função chama a si mesma)")
    def _():
        t.expect(self_calls(tree6("ft_count_harvest_recursive")),
                 "nenhuma função do arquivo chama a si mesma: não é recursiva")

    @t.case("ft_count_harvest_recursive não usa for/while")
    def _():
        ls = loops(tree6("ft_count_harvest_recursive"))
        t.expect(not ls, "a versão recursiva usa laço na linha %s" % ", ".join(map(str, ls)))

    @t.case("ft_count_harvest_iterative usa laço (for/while)")
    def _():
        t.expect(loops(tree6("ft_count_harvest_iterative")), "a versão iterativa não tem for/while")

    @t.case("ft_count_harvest_iterative não é recursiva")
    def _():
        rec = self_calls(tree6("ft_count_harvest_iterative"))
        t.expect(not rec, "a versão iterativa é recursiva: %s" % ", ".join(sorted(rec)))

    @t.case("as duas versões imprimem exatamente o mesmo (7 dias)")
    def _():
        a = call("ex6", "ft_count_harvest_iterative", "7\n")
        b = call("ex6", "ft_count_harvest_recursive", "7\n")
        t.expect(a == b, "iterativa %s, recursiva %s" % (show(a), show(b)))

    @t.case("recursiva funciona chamada duas vezes seguidas")
    def _():
        out = t.py("from ft_count_harvest_recursive import ft_count_harvest_recursive as f\nf()\nf()\n",
                   ex="ex6", stdin="2\n3\n", files=["ft_count_harvest_recursive.py"]).out
        result(out, days(3))
        t.expect(out.count("Harvest time!") == 2 and out.count("Day 1") == 2,
                 "esperado duas contagens completas (2 e 3 dias), recebido %s" % show(out))

    # ---- ex7 -------------------------------------------------------------
    t.group("ex7 ft_seed_inventory")
    basics("ex7", "ft_seed_inventory")

    @t.case("assinatura (seed_type: str, quantity: int, unit: str) -> None")
    def _():
        t.need("ex7/ft_seed_inventory.py")
        f = [n for n in t.tree("ex7/ft_seed_inventory.py").body
             if isinstance(n, ast.FunctionDef) and n.name == "ft_seed_inventory"]
        t.expect(f, "não define: ft_seed_inventory")
        f = f[0]
        got = "(%s) -> %s" % (", ".join("%s: %s" % (a.arg, ast.unparse(a.annotation) if a.annotation else "?")
                                        for a in f.args.args),
                              ast.unparse(f.returns) if f.returns else "?")
        want = "(seed_type: str, quantity: int, unit: str) -> None"
        t.expect(got == want, "esperado %s, recebido %s" % (show(want), show(got)))

    def seeds(args, want):
        exact(call("ex7", "ft_seed_inventory", args=args), want + "\n")

    t.case("exemplo do subject tomato 15 packets",
           lambda: seeds('"tomato", 15, "packets"', "Tomato seeds: 15 packets available"))
    t.case("exemplo do subject carrot 8 grams",
           lambda: seeds('"carrot", 8, "grams"', "Carrot seeds: 8 grams total"))
    t.case("exemplo do subject lettuce 12 area",
           lambda: seeds('"lettuce", 12, "area"', "Lettuce seeds: covers 12 square meters"))
    for args, want in [
        ('"basil", 1, "packets"', "Basil seeds: 1 packets available"),
        ('"pumpkin", 0, "grams"', "Pumpkin seeds: 0 grams total"),
        ('"sunflower", 1000000, "area"', "Sunflower seeds: covers 1000000 square meters"),
        ('"Tomato", 3, "packets"', "Tomato seeds: 3 packets available"),
    ]:
        t.case("%s" % args, lambda args=args, want=want: seeds(args, want))
    for args in ['"tomato", 15, "kg"', '"carrot", 8, ""', '"lettuce", 12, "acres"']:
        t.case("unidade desconhecida: %s" % args,
               lambda args=args: seeds(args, "Unknown unit type"))

    @t.case("retorna None")
    def _():
        out = t.py("from ft_seed_inventory import ft_seed_inventory as f\nr = f('pea', 2, 'grams')\nprint(repr(r))\n",
                   ex="ex7", files=["ft_seed_inventory.py"]).out
        exact(out, "Pea seeds: 2 grams total\nNone\n")
