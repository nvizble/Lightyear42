"""Python Module 01 (Code Cultivation): Object-Oriented Garden Systems."""

import ast
import re

from lt import common, show

EX = [
    ("ex0", "ft_garden_intro"),
    ("ex1", "ft_garden_data"),
    ("ex2", "ft_plant_growth"),
    ("ex3", "ft_plant_factory"),
    ("ex4", "ft_garden_security"),
    ("ex5", "ft_plant_types"),
    ("ex6", "ft_garden_analytics"),
]
FILES = ["%s/%s.py" % e for e in EX]

ALLOWED = {
    "ex0": {"print"},
    "ex1": {"print"},
    "ex2": {"print", "range", "round"},
    "ex3": {"print", "range", "round"},
    "ex4": {"print", "range", "round"},
    "ex5": {"super", "print", "range", "round"},
    "ex6": {"super", "print", "range", "round", "staticmethod", "classmethod"},
}

# Helpers available to every probe snippet. The student's module is imported
# with its stdout silenced; show() may print or return a string, both count.
PRE = r'''
import contextlib, inspect, io
with contextlib.redirect_stdout(io.StringIO()):
    import %(mod)s as M

def cap(f, *a):
    b = io.StringIO()
    with contextlib.redirect_stdout(b):
        r = f(*a)
    s = b.getvalue()
    if isinstance(r, str):
        s += r
    return s

def required(f):
    try:
        ps = inspect.signature(f).parameters.values()
    except (TypeError, ValueError):
        return 0
    return len([p for p in ps if p.default is p.empty
                and p.kind in (p.POSITIONAL_ONLY, p.POSITIONAL_OR_KEYWORD)])

def call0(m):
    """Calls a method with no argument, or with 1 for each required one."""
    return m(*([1] * required(m)))

def mk(cls, *a):
    """Instantiates cls, padding extra required parameters with 0."""
    a = list(a) + [0] * max(0, required(cls) - len(a))
    return cls(*a)

def done(**kw):
    print("@@" + repr(kw))
'''


def probe(t, ex, mod, body):
    """Runs body (after PRE) and returns the dict it passed to done()."""
    r = t.py(PRE % {"mod": mod} + body, ex=ex, files=(mod + ".py",))
    for line in reversed(r.out.splitlines()):
        if line.startswith("@@"):
            return ast.literal_eval(line[2:])
    t.expect(False, "o teste não produziu resultado (saída: %s)" % show(r.out[-200:]))


def need_class(t, rel, *names):
    t.need(rel)
    t.defines(rel, *names)


def nums(s):
    return [float(x) for x in re.findall(r"\d+(?:\.\d+)?", s)]


def ints(s):
    return {int(x) for x in re.findall(r"(?<![\d.])\d+(?![\d.])", s)}


def has_num(s, n):
    return any(abs(x - n) < 1e-6 for x in nums(s))


def has_main_guard(tree):
    for n in ast.walk(tree):
        if isinstance(n, ast.If) and isinstance(n.test, ast.Compare):
            parts = [n.test.left] + list(n.test.comparators)
            names = {p.id for p in parts if isinstance(p, ast.Name)}
            consts = {p.value for p in parts if isinstance(p, ast.Constant)}
            if "__name__" in names and "__main__" in consts:
                return True
    return False


def static_checks(t, ex, rel):
    t.case("usa só funções autorizadas",
           lambda: (t.need(rel), t.authorized(rel, allowed=ALLOWED[ex])))


def check_plant_show(t, s, name, height, age, what="show()"):
    t.expect(name in s, "%s deveria mostrar o nome %s, mostrou %s" % (what, show(name), show(s)))
    t.expect(has_num(s, height), "%s deveria mostrar a altura %s, mostrou %s" % (what, height, show(s)))
    t.expect(has_num(s, age), "%s deveria mostrar a idade %s, mostrou %s" % (what, age, show(s)))


# ---------------------------------------------------------------- ex0

def ex0(t):
    ex, mod = EX[0]
    rel = "%s/%s.py" % EX[0]
    t.group("ex0 " + mod)
    static_checks(t, ex, rel)

    @t.case("usa if __name__ == \"__main__\":")
    def _():
        t.need(rel)
        t.expect(has_main_guard(t.tree(rel)), "não achei o bloco if __name__ == \"__main__\":")

    @t.case("importar o arquivo não imprime nada")
    def _():
        r = t.py("import contextlib, io\nimport %s\n" % mod, ex=ex, files=(mod + ".py",))
        t.expect(r.out == "", "esperado nada ao importar, saiu %s" % show(r.out))

    @t.case("python3 %s.py mostra nome, altura e idade" % mod)
    def _():
        r = t.run(rel)
        t.expect(r.code == 0, "código de saída %d" % r.code)
        lines = [x for x in r.out.splitlines() if x.strip()]
        t.expect(len(lines) >= 3, "esperado pelo menos 3 linhas (nome, altura, idade), saiu %s" % show(r.out))
        t.expect(len(nums(r.out)) >= 2, "esperado altura e idade na saída, saiu %s" % show(r.out))


# ---------------------------------------------------------------- ex1

def ex1(t):
    ex, mod = EX[1]
    rel = "%s/%s.py" % EX[1]
    t.group("ex1 " + mod)
    static_checks(t, ex, rel)
    t.case("define a classe Plant", lambda: need_class(t, rel, "Plant"))

    @t.case("Plant tem o método show()")
    def _():
        d = probe(t, ex, mod, "done(ok=callable(getattr(M.Plant, 'show', None)))")
        t.expect(d["ok"], "Plant não tem um método show()")

    @t.case("python3 %s.py mostra pelo menos 3 plantas" % mod)
    def _():
        r = t.run(rel)
        t.expect(r.code == 0, "código de saída %d" % r.code)
        rows = [x for x in r.out.splitlines() if x.strip() and len(nums(x)) >= 2]
        t.expect(len(rows) >= 3, "esperado 3+ plantas (nome, altura, idade), saiu %s" % show(r.out))

    @t.case("show() mostra os dados da própria instância")
    def _():
        d = probe(t, ex, mod, """
P = M.Plant
if required(P) >= 3:
    a, b = P("Zinnia", 37, 41), P("Lotus", 12, 8)
else:
    done(skip=True)
    raise SystemExit
done(a=cap(a.show), b=cap(b.show))
""")
        if d.get("skip"):  # no constructor: attributes set afterwards, check the program's rows
            rows = [x for x in t.run(rel).out.splitlines() if len(nums(x)) >= 2]
            t.expect(len(set(rows)) >= 3, "as 3 plantas deveriam ser diferentes, saiu %s" % show(rows))
            return
        check_plant_show(t, d["a"], "Zinnia", 37, 41)
        check_plant_show(t, d["b"], "Lotus", 12, 8)


# ---------------------------------------------------------------- ex2

GROWTH_ROW = re.compile(r"(\w+):\s*(\d+(?:\.\d+)?)\s*cm,\s*(\d+)\s*days?")


def ex2(t):
    ex, mod = EX[2]
    rel = "%s/%s.py" % EX[2]
    t.group("ex2 " + mod)
    static_checks(t, ex, rel)
    t.case("define a classe Plant", lambda: need_class(t, rel, "Plant"))

    @t.case("Plant tem show(), grow() e age()")
    def _():
        d = probe(t, ex, mod, "done(miss=[m for m in ('show', 'grow', 'age') "
                              "if not callable(getattr(M.Plant, m, None))])")
        t.expect(not d["miss"], "faltam os métodos: %s" % ", ".join(d["miss"]))

    @t.case("python3 %s.py simula a semana e mostra o crescimento" % mod)
    def _():
        r = t.run(rel)
        t.expect(r.code == 0, "código de saída %d" % r.code)
        t.expect(r.out.strip() != "", "não imprimiu nada")
        rows = GROWTH_ROW.findall(r.out)
        t.expect(len(rows) >= 2, "esperado a planta mostrada a cada dia, saiu %s" % show(r.out))
        growth = [x for x in r.out.splitlines() if "growth" in x.lower() or "week" in x.lower()]
        t.expect(growth and nums(growth[-1]), "faltou o total de crescimento da semana")
        want = float(rows[-1][1]) - float(rows[0][1])
        got = nums(growth[-1])[-1]
        t.expect(abs(got - want) < 0.051,
                 "crescimento da semana: esperado %.1f (altura final - inicial), mostrou %s"
                 % (want, show(growth[-1])))

    @t.case("python3 %s.py: a idade avança ao longo da semana" % mod)
    def _():
        rows = GROWTH_ROW.findall(t.run(rel).out)
        if len(rows) < 2:
            t.skip("formato da saída diferente do exemplo")
        t.expect(int(rows[-1][2]) > int(rows[0][2]),
                 "idade não mudou: %s -> %s" % (rows[0][2], rows[-1][2]))

    @t.case("grow() e age() mudam a planta")
    def _():
        d = probe(t, ex, mod, """
P = M.Plant
if required(P) < 3:
    done(skip=True)
    raise SystemExit
p = P("Zinnia", 37.0, 41)
s0 = cap(p.show)
cap(call0, p.grow)
s1 = cap(p.show)
cap(call0, p.age)
s2 = cap(p.show)
done(s0=s0, s1=s1, s2=s2)
""")
        if d.get("skip"):
            t.skip("Plant sem construtor (name, height, age)")
        check_plant_show(t, d["s0"], "Zinnia", 37, 41)
        t.expect(d["s1"] != d["s0"], "grow() não mudou nada: %s" % show(d["s1"]))
        t.expect(d["s2"] != d["s1"], "age() não mudou nada: %s" % show(d["s2"]))


# ---------------------------------------------------------------- ex3

def ex3(t):
    ex, mod = EX[3]
    rel = "%s/%s.py" % EX[3]
    t.group("ex3 " + mod)
    static_checks(t, ex, rel)
    t.case("define a classe Plant", lambda: need_class(t, rel, "Plant"))

    @t.case("Plant(name, height, age) já cria a planta pronta")
    def _():
        d = probe(t, ex, mod, "p = M.Plant('Zinnia', 37.0, 41)\ndone(s=cap(p.show))")
        check_plant_show(t, d["s"], "Zinnia", 37, 41)

    @t.case("cada planta guarda seus próprios dados")
    def _():
        d = probe(t, ex, mod, """
a = M.Plant('Zinnia', 37.0, 41)
b = M.Plant('Lotus', 12.0, 8)
done(a=cap(a.show), b=cap(b.show))
""")
        check_plant_show(t, d["a"], "Zinnia", 37, 41, "show() da 1ª planta")
        check_plant_show(t, d["b"], "Lotus", 12, 8, "show() da 2ª planta")

    @t.case("grow() funciona logo após a criação")
    def _():
        d = probe(t, ex, mod, """
p = M.Plant('Zinnia', 37.0, 41)
if not callable(getattr(p, 'grow', None)):
    done(skip=True)
    raise SystemExit
s0 = cap(p.show)
cap(call0, p.grow)
done(s0=s0, s1=cap(p.show))
""")
        if d.get("skip"):
            t.skip("Plant sem grow()")
        t.expect(d["s1"] != d["s0"], "grow() não mudou a planta: %s" % show(d["s1"]))

    @t.case("python3 %s.py mostra pelo menos 5 plantas" % mod)
    def _():
        r = t.run(rel)
        t.expect(r.code == 0, "código de saída %d" % r.code)
        rows = [x for x in r.out.splitlines() if len(nums(x)) >= 2]
        t.expect(len(rows) >= 5, "esperado 5+ plantas, saiu %s" % show(r.out))
        t.expect(len(set(rows)) >= 5, "as plantas deveriam ser diferentes, saiu %s" % show(r.out))


# ---------------------------------------------------------------- ex4

def ex4(t):
    ex, mod = EX[4]
    rel = "%s/%s.py" % EX[4]
    t.group("ex4 " + mod)
    static_checks(t, ex, rel)
    t.case("define a classe Plant", lambda: need_class(t, rel, "Plant"))

    @t.case("Plant tem set_height, set_age, get_height, get_age")
    def _():
        d = probe(t, ex, mod, "done(miss=[m for m in ('set_height', 'set_age', 'get_height', 'get_age') "
                              "if not callable(getattr(M.Plant, m, None))])")
        t.expect(not d["miss"], "faltam os métodos: %s" % ", ".join(d["miss"]))

    @t.case("get_height()/get_age() devolvem os valores do construtor")
    def _():
        d = probe(t, ex, mod, "p = M.Plant('Zinnia', 37.5, 41)\ndone(h=p.get_height(), a=p.get_age())")
        t.expect(d["h"] == 37.5, "get_height(): esperado 37.5, veio %s" % show(d["h"]))
        t.expect(d["a"] == 41, "get_age(): esperado 41, veio %s" % show(d["a"]))

    @t.case("set_height()/set_age() aceitam valores válidos (inclusive 0)")
    def _():
        d = probe(t, ex, mod, """
p = M.Plant('Zinnia', 37.5, 41)
cap(p.set_height, 25.0); cap(p.set_age, 30)
h, a = p.get_height(), p.get_age()
cap(p.set_height, 0); cap(p.set_age, 0)
done(h=h, a=a, h0=p.get_height(), a0=p.get_age())
""")
        t.expect(d["h"] == 25.0, "set_height(25.0) -> get_height(): esperado 25.0, veio %s" % show(d["h"]))
        t.expect(d["a"] == 30, "set_age(30) -> get_age(): esperado 30, veio %s" % show(d["a"]))
        t.expect(d["h0"] == 0, "set_height(0) deveria ser aceito, get_height() veio %s" % show(d["h0"]))
        t.expect(d["a0"] == 0, "set_age(0) deveria ser aceito, get_age() veio %s" % show(d["a0"]))

    @t.case("set_height(negativo) é recusado com mensagem de erro")
    def _():
        d = probe(t, ex, mod, """
p = M.Plant('Zinnia', 37.5, 41)
done(msg=cap(p.set_height, -5.0), h=p.get_height())
""")
        t.expect(d["h"] == 37.5, "altura deveria continuar 37.5, ficou %s" % show(d["h"]))
        t.expect(d["msg"].strip() != "", "set_height(-5.0) deveria imprimir um erro, não imprimiu nada")

    @t.case("set_age(negativo) é recusado com mensagem de erro")
    def _():
        d = probe(t, ex, mod, """
p = M.Plant('Zinnia', 37.5, 41)
done(msg=cap(p.set_age, -3), a=p.get_age())
""")
        t.expect(d["a"] == 41, "idade deveria continuar 41, ficou %s" % show(d["a"]))
        t.expect(d["msg"].strip() != "", "set_age(-3) deveria imprimir um erro, não imprimiu nada")

    @t.case("criar com valores negativos não guarda negativos")
    def _():
        d = probe(t, ex, mod, """
b = io.StringIO()
with contextlib.redirect_stdout(b):
    p = M.Plant('Bad', -5.0, -2)
done(msg=b.getvalue(), h=p.get_height(), a=p.get_age())
""")
        t.expect(d["h"] >= 0, "Plant('Bad', -5.0, -2): altura ficou %s" % show(d["h"]))
        t.expect(d["a"] >= 0, "Plant('Bad', -5.0, -2): idade ficou %s" % show(d["a"]))
        t.expect(d["msg"].strip() != "", "deveria imprimir um erro ao criar com valores negativos")

    @t.case("altura e idade são protegidas (_atributo, sem mangling)")
    def _():
        d = probe(t, ex, mod, """
p = M.Plant('Zinnia', 37.5, 41)
v = vars(p)
done(h=[k for k in v if v[k] == 37.5], a=[k for k in v if v[k] == 41 and v[k] is not True])
""")
        keys = d["h"] + d["a"]
        if not keys:
            t.skip("não achei os atributos de altura/idade na instância")
        public = [k for k in keys if not k.startswith("_")]
        mangled = [k for k in keys if k.startswith("_Plant__")]
        t.expect(not public, "atributos públicos: %s (use _nome)" % ", ".join(public))
        t.expect(not mangled, "use o 'protected' (_nome), não o mangling: %s" % ", ".join(mangled))

    @t.case("python3 %s.py demonstra a validação" % mod)
    def _():
        r = t.run(rel)
        t.expect(r.code == 0, "código de saída %d" % r.code)
        low = r.out.lower()
        t.expect(any(w in low for w in ("error", "erro", "invalid", "reject", "negative")),
                 "a saída deveria mostrar uma atualização recusada, saiu %s" % show(r.out))


# ---------------------------------------------------------------- ex5

def calls_super(tree, cls):
    for n in tree.body:
        if isinstance(n, ast.ClassDef) and n.name == cls:
            return any(isinstance(c, ast.Call) and isinstance(c.func, ast.Name) and c.func.id == "super"
                       for c in ast.walk(n))
    return False


def ex5(t):
    ex, mod = EX[5]
    rel = "%s/%s.py" % EX[5]
    t.group("ex5 " + mod)
    static_checks(t, ex, rel)
    t.case("define Plant, Flower, Tree, Vegetable",
           lambda: need_class(t, rel, "Plant", "Flower", "Tree", "Vegetable"))

    @t.case("Flower, Tree e Vegetable herdam de Plant")
    def _():
        d = probe(t, ex, mod, "done(bad=[c for c in ('Flower', 'Tree', 'Vegetable') "
                              "if not issubclass(getattr(M, c), M.Plant)])")
        t.expect(not d["bad"], "não herdam de Plant: %s" % ", ".join(d["bad"]))

    @t.case("as subclasses chamam super()")
    def _():
        t.need(rel)
        tree = t.tree(rel)
        bad = [c for c in ("Flower", "Tree", "Vegetable") if not calls_super(tree, c)]
        t.expect(not bad, "não usam super(): %s" % ", ".join(bad))

    @t.case("Flower: show() mostra a planta e a cor")
    def _():
        d = probe(t, ex, mod, "f = mk(M.Flower, 'Zinnia', 37.0, 41, 'purple')\ndone(s=cap(f.show))")
        check_plant_show(t, d["s"], "Zinnia", 37, 41, "Flower.show()")
        t.expect("purple" in d["s"], "Flower.show() deveria mostrar a cor 'purple', mostrou %s" % show(d["s"]))

    @t.case("Flower: bloom() muda o que show() mostra")
    def _():
        d = probe(t, ex, mod, """
f = mk(M.Flower, 'Zinnia', 37.0, 41, 'purple')
s0 = cap(f.show)
cap(f.bloom)
done(s0=s0, s1=cap(f.show))
""")
        t.expect(d["s1"] != d["s0"], "show() igual antes e depois do bloom(): %s" % show(d["s1"]))

    @t.case("Tree: show() mostra a planta e o diâmetro do tronco")
    def _():
        d = probe(t, ex, mod, "x = mk(M.Tree, 'Maple', 210.0, 400, 7.5)\ndone(s=cap(x.show))")
        check_plant_show(t, d["s"], "Maple", 210, 400, "Tree.show()")
        t.expect(has_num(d["s"], 7.5), "Tree.show() deveria mostrar o diâmetro 7.5, mostrou %s" % show(d["s"]))

    @t.case("Tree: produce_shade() imprime a sombra")
    def _():
        d = probe(t, ex, mod, "x = mk(M.Tree, 'Maple', 210.0, 400, 7.5)\ndone(s=cap(x.produce_shade))")
        t.expect(d["s"].strip() != "", "produce_shade() não imprimiu nada")

    @t.case("Vegetable: show() mostra a planta, a estação e o valor nutricional")
    def _():
        d = probe(t, ex, mod, "v = mk(M.Vegetable, 'Carrot', 6.0, 12, 'October')\ndone(s=cap(v.show))")
        check_plant_show(t, d["s"], "Carrot", 6, 12, "Vegetable.show()")
        t.expect("October" in d["s"], "Vegetable.show() deveria mostrar 'October', mostrou %s" % show(d["s"]))
        t.expect("nutri" in d["s"].lower(), "Vegetable.show() deveria mostrar o valor nutricional, mostrou %s"
                 % show(d["s"]))

    @t.case("Vegetable: valor nutricional começa em 0 e sobe com grow()/age()")
    def _():
        d = probe(t, ex, mod, """
import re
v = mk(M.Vegetable, 'Carrot', 6.0, 12, 'October')
def nv():
    for k in ('nutritional_value', '_nutritional_value'):
        if k in vars(v):
            return vars(v)[k]
    g = getattr(v, 'get_nutritional_value', None)
    if callable(g):
        return g()
    m = re.search(r'nutri[^\\d\\n]*(\\d+(?:\\.\\d+)?)', cap(v.show), re.I)
    return float(m.group(1)) if m else None
n0 = nv()
s0 = cap(v.show)
for _ in range(5):
    cap(call0, v.grow)
    cap(call0, v.age)
done(n0=n0, n1=nv(), s0=s0, s1=cap(v.show))
""")
        t.expect(d["n0"] is not None, "não achei o nutritional_value")
        t.expect(d["n0"] == 0, "valor nutricional inicial: esperado 0, veio %s" % show(d["n0"]))
        t.expect(d["n1"] > 0, "depois de grow()/age() o valor nutricional deveria subir, ficou %s" % show(d["n1"]))
        t.expect(d["s1"] != d["s0"], "grow()/age() não mudaram a planta: %s" % show(d["s1"]))

    @t.case("as subclasses herdam grow() e age() de Plant")
    def _():
        d = probe(t, ex, mod, "done(bad=['%s.%s' % (c, m) for c in ('Flower', 'Tree', 'Vegetable') "
                              "for m in ('grow', 'age', 'show') if not callable(getattr(getattr(M, c), m, None))])")
        t.expect(not d["bad"], "faltam: %s" % ", ".join(d["bad"]))

    @t.case("python3 %s.py roda sem erro" % mod)
    def _():
        r = t.run(rel)
        t.expect(r.code == 0, "código de saída %d" % r.code)
        t.expect(r.out.strip() != "", "não imprimiu nada")


# ---------------------------------------------------------------- ex6

STATS = """
plants = [v for v in vars(M).values() if inspect.isclass(v) and issubclass(v, M.Plant)]
funcs = [v for k, v in vars(M).items() if inspect.isfunction(v)
         and v.__module__ == M.__name__ and required(v) == 1]
def stats_of(p):
    outs = []
    for f in funcs:
        try:
            outs.append(cap(f, p))
        except Exception as e:
            outs.append('')
    return outs
"""


def ex6(t):
    ex, mod = EX[6]
    rel = "%s/%s.py" % EX[6]
    t.group("ex6 " + mod)
    static_checks(t, ex, rel)
    t.case("define Plant, Flower, Tree, Seed",
           lambda: need_class(t, rel, "Plant", "Flower", "Tree", "Seed"))

    @t.case("Seed herda de Flower; Flower e Tree herdam de Plant")
    def _():
        d = probe(t, ex, mod, "done(seed=issubclass(M.Seed, M.Flower), flower=issubclass(M.Flower, M.Plant), "
                              "tree=issubclass(M.Tree, M.Plant))")
        t.expect(d["seed"], "Seed não herda de Flower")
        t.expect(d["flower"], "Flower não herda de Plant")
        t.expect(d["tree"], "Tree não herda de Plant")

    @t.case("Plant tem um staticmethod que diz se a idade passa de um ano")
    def _():
        d = probe(t, ex, mod, """
res = []
for k, v in vars(M.Plant).items():
    if isinstance(v, staticmethod):
        f = getattr(M.Plant, k)
        try:
            with contextlib.redirect_stdout(io.StringIO()):
                res.append((k, [f(n) for n in (30, 364, 366, 400)]))
        except Exception as e:
            res.append((k, repr(e)))
done(res=res)
""")
        t.expect(d["res"], "Plant não tem nenhum staticmethod")
        want = [False, False, True, True]
        ok = [k for k, v in d["res"] if isinstance(v, list) and [bool(x) for x in v] == want
              and all(isinstance(x, bool) for x in v)]
        t.expect(ok, "esperado f(30)=False f(364)=False f(366)=True f(400)=True, veio %s"
                 % show(d["res"]))

    @t.case("Plant tem um classmethod que cria uma planta anônima")
    def _():
        d = probe(t, ex, mod, """
res = []
for k, v in vars(M.Plant).items():
    if isinstance(v, classmethod):
        f = getattr(M.Plant, k)
        if required(f) == 0:
            b = io.StringIO()
            with contextlib.redirect_stdout(b):
                p = f()
            res.append((k, isinstance(p, M.Plant), cap(p.show) if isinstance(p, M.Plant) else ''))
done(res=res)
""")
        t.expect(d["res"], "Plant não tem classmethod chamável sem argumentos")
        ok = [r for r in d["res"] if r[1]]
        t.expect(ok, "o classmethod deveria devolver uma Plant, veio %s" % show(d["res"]))
        t.expect(ok[0][2].strip() != "", "show() da planta anônima não mostrou nada")

    @t.case("Plant tem uma classe aninhada de estatísticas")
    def _():
        d = probe(t, ex, mod, "done(n=[k for k, v in vars(M.Plant).items() if inspect.isclass(v)])")
        t.expect(d["n"], "não achei classe aninhada dentro de Plant")

    @t.case("estatísticas são encapsuladas (_atributo)")
    def _():
        d = probe(t, ex, mod, """
nested = tuple(v for v in vars(M.Plant).values() if inspect.isclass(v))
p = mk(M.Plant, 'Zinnia', 37.0, 41)
objs = [o for o in vars(p).values() if nested and isinstance(o, nested)]
done(found=bool(objs), keys=[k for o in objs for k in vars(o)])
""")
        t.expect(d["found"], "a planta não guarda um objeto da classe aninhada")
        public = [k for k in d["keys"] if not k.startswith("_")]
        t.expect(not public, "contadores públicos: %s (use _nome)" % ", ".join(public))

    @t.case("função fora das classes mostra grow/age/show contados")
    def _():
        d = probe(t, ex, mod, STATS + """
p = mk(M.Plant, 'Zinnia', 37.0, 41)
for _ in range(3):
    cap(call0, p.grow)
for _ in range(5):
    cap(call0, p.age)
for _ in range(2):
    cap(p.show)
done(nf=len(funcs), outs=stats_of(p))
""")
        t.expect(d["nf"], "não achei função de nível de módulo que recebe uma planta")
        ok = [o for o in d["outs"] if {3, 5, 2} <= ints(o)]
        t.expect(ok, "depois de 3 grow, 5 age, 2 show esperado 3, 5 e 2 nas estatísticas, saiu %s"
                 % show(d["outs"]))

    @t.case("Flower: contadores também funcionam (herdados)")
    def _():
        d = probe(t, ex, mod, STATS + """
p = mk(M.Flower, 'Zinnia', 37.0, 41, 'red')
for _ in range(4):
    cap(call0, p.grow)
for _ in range(6):
    cap(p.show)
done(outs=stats_of(p))
""")
        ok = [o for o in d["outs"] if {4, 0, 6} <= ints(o)]
        t.expect(ok, "depois de 4 grow, 0 age, 6 show esperado 4, 0 e 6, saiu %s" % show(d["outs"]))

    @t.case("Tree: estatística conta produce_shade()")
    def _():
        d = probe(t, ex, mod, STATS + """
x = mk(M.Tree, 'Maple', 210.0, 400, 7.5)
before = stats_of(x)
for _ in range(7):
    cap(x.produce_shade)
done(before=before, outs=stats_of(x))
""")
        ok = [o for o in d["outs"] if 7 in ints(o)]
        t.expect(ok, "depois de 7 produce_shade() esperado 7 nas estatísticas, saiu %s" % show(d["outs"]))
        t.expect(any(7 not in ints(o) for o in d["before"]),
                 "o 7 já aparecia antes do produce_shade(): %s" % show(d["before"]))

    @t.case("Seed: show() mostra a flor e as sementes")
    def _():
        d = probe(t, ex, mod, "s = mk(M.Seed, 'Sunflower', 80.0, 45, 'yellow')\ndone(s=cap(s.show))")
        check_plant_show(t, d["s"], "Sunflower", 80, 45, "Seed.show()")
        t.expect("yellow" in d["s"], "Seed.show() deveria mostrar a cor 'yellow', mostrou %s" % show(d["s"]))
        t.expect("seed" in d["s"].lower(), "Seed.show() deveria mostrar as sementes, mostrou %s" % show(d["s"]))

    @t.case("Seed: sementes aparecem depois do bloom()")
    def _():
        d = probe(t, ex, mod, """
s = mk(M.Seed, 'Sunflower', 80.0, 45, 'yellow')
s0 = cap(s.show)
cap(s.bloom)
done(s0=s0, s1=cap(s.show))
""")
        seeds = [x for x in d["s0"].splitlines() if "seed" in x.lower()]
        if seeds and nums(seeds[-1]):
            t.expect(nums(seeds[-1])[-1] == 0, "antes do bloom() as sementes deveriam ser 0: %s" % show(seeds[-1]))
        t.expect(d["s1"] != d["s0"], "show() igual antes e depois do bloom(): %s" % show(d["s1"]))

    @t.case("python3 %s.py roda sem erro" % mod)
    def _():
        r = t.run(rel)
        t.expect(r.code == 0, "código de saída %d" % r.code)
        t.expect(r.out.strip() != "", "não imprimiu nada")


def suite(t):
    common(t, FILES, typed=FILES)
    for f in (ex0, ex1, ex2, ex3, ex4, ex5, ex6):
        f(t)
