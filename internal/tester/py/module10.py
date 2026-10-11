"""Python Module 10 (FuncMage Chronicles): lambdas, higher-order functions,
closures, functools, decorators."""

import ast
import builtins
import math
import sys

from lt import Fail, common, show

EX0 = "ex0/lambda_spells.py"
EX1 = "ex1/higher_magic.py"
EX2 = "ex2/scope_mysteries.py"
EX3 = "ex3/functools_artifacts.py"
EX4 = "ex4/decorator_mastery.py"
FILES = [EX0, EX1, EX2, EX3, EX4]

# "All built-in functions are authorized, but eval() and exec()".
ALL_BUILTINS = set(dir(builtins)) - {"eval", "exec"}
# Standard library only; functools/operator only where the exercise allows.
STDLIB = set(sys.stdlib_module_names) - {"functools", "operator"}
TYPES = {"list", "dict", "tuple", "str", "int", "float", "bool", "set"} | {
    n for n in dir(builtins) if isinstance(getattr(builtins, n), type) and issubclass(getattr(builtins, n), BaseException)}
EX0_OK = {"map", "filter", "sorted", "min", "max", "round", "sum", "len", "print", "isinstance"} | TYPES

# Loads only the definitions of a student file (imports, functions, classes,
# constant assignments) so its main code doesn't run.
LOADER = r'''
import ast, types, io, contextlib
def _load(path):
    tree = ast.parse(open(path, encoding="utf-8").read())
    funcs = {n.name for n in tree.body
             if isinstance(n, (ast.FunctionDef, ast.ClassDef))}
    def risky(n):
        for x in ast.walk(n):
            if isinstance(x, ast.Call) and isinstance(x.func, ast.Name) \
                    and (x.func.id in funcs or x.func.id in ("input", "print")):
                return True
        return False
    mod = types.ModuleType("student")
    mod.__file__ = path
    for n in tree.body:
        if (isinstance(n, (ast.Import, ast.ImportFrom, ast.FunctionDef, ast.ClassDef))
                or (isinstance(n, (ast.Assign, ast.AnnAssign)) and not risky(n))):
            try:
                exec(compile(ast.Module(body=[n], type_ignores=[]), path, "exec"), mod.__dict__)
            except Exception:
                pass
    return mod
def _out(fn, *a, **k):
    buf = io.StringIO()
    with contextlib.redirect_stdout(buf):
        r = fn(*a, **k)
    return r, buf.getvalue()
'''


def snippet(t, rel, body, timeout=5):
    """Runs body with the student's definitions loaded as m; body prints OK
    or KO <message> as its last line."""
    ex, name = rel.split("/")
    r = t.py(LOADER + "m = _load(%r)\n" % name + body, ex, files=(name,), timeout=timeout)
    out = [x for x in r.out.strip().splitlines() if x.strip()]
    last = out[-1].strip() if out else ""
    if last.startswith("KO "):
        raise Fail(last[3:])
    t.expect(last == "OK", "saída inesperada do teste: " + show(r.out[-300:]))


def val(t, rel, expr):
    """The value of expr (a literal) evaluated with the student's module as m."""
    ex, name = rel.split("/")
    code = LOADER + "m = _load(%r)\n" % name + (
        "try:\n    _r = (%s)\nexcept Exception as e:\n"
        "    print('\\nE %%s: %%s' %% (type(e).__name__, e))\n"
        "else:\n    print('\\nR ' + repr(_r))\n") % expr
    r = t.py(code, ex, files=(name,))
    lines = [x for x in r.out.splitlines() if x.strip()]
    last = lines[-1] if lines else ""
    if last.startswith("E "):
        raise Fail("%s levantou %s" % (expr.replace("m.", ""), last[2:]))
    t.expect(last.startswith("R "), "saída inesperada do teste: " + show(r.out[-300:]))
    try:
        return ast.literal_eval(last[2:])
    except (ValueError, SyntaxError):
        raise Fail("%s devolveu %s" % (expr.replace("m.", ""), last[2:]))


def eq(t, rel, expr, want):
    got = val(t, rel, expr)
    t.expect(got == want and type(got) is type(want),
             "%s: esperava %s, veio %s" % (expr.replace("m.", ""), show(want), show(got)))


def func(t, rel, name):
    for n in t.tree(rel).body:
        if isinstance(n, ast.FunctionDef) and n.name == name:
            return n
    raise Fail("não define %s()" % name)


def method(t, rel, cls, name):
    for n in t.tree(rel).body:
        if isinstance(n, ast.ClassDef) and n.name == cls:
            for f in n.body:
                if isinstance(f, ast.FunctionDef) and f.name == name:
                    return f
            raise Fail("a classe %s não define %s()" % (cls, name))
    raise Fail("não define a classe " + cls)


def sig(t, node, params):
    got = [a.arg for a in node.args.posonlyargs + node.args.args]
    t.expect(len(got) == len(params), "%s(%s): esperava os parâmetros (%s)"
             % (node.name, ", ".join(got), ", ".join(params)))


def calls(node, name):
    """Calls of name(...) or x.name(...) inside node."""
    return [n for n in ast.walk(node) if isinstance(n, ast.Call)
            and ((isinstance(n.func, ast.Name) and n.func.id == name)
                 or (isinstance(n.func, ast.Attribute) and n.func.attr == name))]


def mentions(node, name):
    return any((isinstance(n, ast.Name) and n.id == name) or (isinstance(n, ast.Attribute) and n.attr == name)
               or (isinstance(n, ast.alias) and n.name == name)
               or (isinstance(n, ast.ImportFrom) and n.module == name) for n in ast.walk(node))


def signatures(t, rel, spec):
    for name, params in spec.items():
        t.case("%s(%s)" % (name, ", ".join(params)), lambda n=name, p=params: sig(t, func(t, rel, n), p))


def no_typing_callable(t, rel):
    for n in ast.walk(t.tree(rel)):
        bad = (isinstance(n, ast.ImportFrom) and n.module == "typing" and any(a.name == "Callable" for a in n.names)) \
            or (isinstance(n, ast.Attribute) and n.attr == "Callable" and isinstance(n.value, ast.Name)
                and n.value.id == "typing")
        if bad:
            raise Fail("Callable deve vir de collections.abc, não de typing (linha %d)" % n.lineno)


def suite(t):
    common(t, FILES, typed=FILES)
    t.group("regras gerais")
    for rel in FILES:
        if t.exists(rel):
            t.case("sem global: " + rel, lambda rel=rel: t.expect(
                not any(isinstance(n, ast.Global) for n in ast.walk(t.tree(rel))),
                "usa 'global' (variáveis globais são proibidas)"))
            t.case("Callable de collections.abc: " + rel, lambda rel=rel: no_typing_callable(t, rel))
    ex0(t)
    ex1(t)
    ex2(t)
    ex3(t)
    ex4(t)


# ---- ex0 ---------------------------------------------------------------

ARTS = [{"name": "Crystal Orb", "power": 85, "type": "orb"},
        {"name": "Fire Staff", "power": 92, "type": "staff"},
        {"name": "Iron Ring", "power": 40, "type": "ring"},
        {"name": "Moon Blade", "power": 77, "type": "blade"},
        {"name": "Void Tome", "power": 100, "type": "tome"}]
MAGES = [{"name": "Alex", "power": 70, "element": "fire"},
         {"name": "Riley", "power": 45, "element": "ice"},
         {"name": "Jordan", "power": 50, "element": "earth"},
         {"name": "Sam", "power": 92, "element": "wind"},
         {"name": "Kai", "power": 49, "element": "water"}]


def ex0(t):
    t.group("ex0 lambda_spells")
    t.case("só usa o permitido (sem eval/exec, só stdlib)", lambda: t.authorized(EX0, ALL_BUILTINS, STDLIB))
    signatures(t, EX0, {"artifact_sorter": ["artifacts"], "power_filter": ["mages", "min_power"],
                        "spell_transformer": ["spells"], "mage_stats": ["mages"]})
    names = ["artifact_sorter", "power_filter", "spell_transformer", "mage_stats"]

    @t.case("as funções só usam map, filter, sorted, min, max, round, sum, len")
    def _():
        bad = set()
        for name in names:
            node = func(t, EX0, name)
            local = {a.arg for n in ast.walk(node) if isinstance(n, ast.arguments) for a in n.args}
            local |= {n.id for n in ast.walk(node) if isinstance(n, ast.Name) and isinstance(n.ctx, ast.Store)}
            for c in ast.walk(node):
                if isinstance(c, ast.Call) and isinstance(c.func, ast.Name) and c.func.id in ALL_BUILTINS | {"eval", "exec"} \
                        and c.func.id not in EX0_OK and c.func.id not in local:
                    bad.add("%s() em %s" % (c.func.id, name))
        t.expect(not bad, "usa o que não é permitido: " + ", ".join(sorted(bad)))

    @t.case("nenhum def dentro das funções (lambda no lugar)")
    def _():
        for name in names:
            inner = [n.name for n in ast.walk(func(t, EX0, name)) if isinstance(n, ast.FunctionDef) and n.name != name]
            t.expect(not inner, "%s define a função %s com def: use lambda" % (name, inner[0] if inner else ""))

    @t.case("artifact_sorter usa sorted() com key=lambda")
    def _():
        ok = [c for c in calls(func(t, EX0, "artifact_sorter"), "sorted")
              if any(k.arg == "key" and isinstance(k.value, ast.Lambda) for k in c.keywords)]
        t.expect(ok, "não encontrei sorted(..., key=lambda ...) em artifact_sorter")

    @t.case("power_filter usa filter() com lambda")
    def _():
        ok = [c for c in calls(func(t, EX0, "power_filter"), "filter") if c.args and isinstance(c.args[0], ast.Lambda)]
        t.expect(ok, "não encontrei filter(lambda ..., mages) em power_filter")

    @t.case("spell_transformer usa map() com lambda")
    def _():
        ok = [c for c in calls(func(t, EX0, "spell_transformer"), "map") if c.args and isinstance(c.args[0], ast.Lambda)]
        t.expect(ok, "não encontrei map(lambda ..., spells) em spell_transformer")

    @t.case("mage_stats usa max() e min() com lambdas")
    def _():
        node = func(t, EX0, "mage_stats")
        t.expect(calls(node, "max") and calls(node, "min"), "mage_stats devia usar max() e min()")
        t.expect(any(isinstance(n, ast.Lambda) for n in ast.walk(node)), "mage_stats não usa nenhuma lambda")

    def sorter(arts):
        got = val(t, EX0, "m.artifact_sorter(%r)" % (arts,))
        t.expect(isinstance(got, list), "esperava uma list, veio " + show(got))
        want = sorted(arts, key=lambda a: a["power"], reverse=True)
        ok = len(got) == len(arts) and all(g in arts for g in got) and all(a in got for a in arts) \
            and [g["power"] for g in got] == [w["power"] for w in want]
        t.expect(ok, "esperava em ordem decrescente de power %s, veio %s" % (show(want), show(got)))

    t.case("artifact_sorter: exemplo (Fire Staff 92 antes de Crystal Orb 85)", lambda: sorter(ARTS[:2]))
    t.case("artifact_sorter: 5 artefatos", lambda: sorter(ARTS))
    t.case("artifact_sorter: empates", lambda: sorter([{"name": c, "power": p, "type": "x"}
                                                       for c, p in zip("abcde", [5, 9, 5, 9, 1])]))
    t.case("artifact_sorter: negativos", lambda: sorter([{"name": c, "power": p, "type": "x"}
                                                         for c, p in zip("abc", [-3, 0, -10])]))
    t.case("artifact_sorter: um artefato", lambda: sorter(ARTS[:1]))
    t.case("artifact_sorter: lista vazia", lambda: eq(t, EX0, "m.artifact_sorter([])", []))

    def pfilter(mages, mn):
        eq(t, EX0, "m.power_filter(%r, %r)" % (mages, mn), [x for x in mages if x["power"] >= mn])

    t.case("power_filter: power >= 50 (50 entra, 49 não)", lambda: pfilter(MAGES, 50))
    t.case("power_filter: mantém a ordem original", lambda: pfilter(MAGES, 45))
    t.case("power_filter: ninguém passa", lambda: pfilter(MAGES, 1000))
    t.case("power_filter: todos passam", lambda: pfilter(MAGES, 0))
    t.case("power_filter: min_power negativo", lambda: pfilter(
        [{"name": "a", "power": -5, "element": "x"}, {"name": "b", "power": -20, "element": "y"}], -10))
    t.case("power_filter: lista vazia", lambda: pfilter([], 10))

    def transf(spells):
        eq(t, EX0, "m.spell_transformer(%r)" % (spells,), ["* " + s + " *" for s in spells])

    t.case("spell_transformer: exemplo (fireball heal shield)", lambda: transf(["fireball", "heal", "shield"]))
    t.case("spell_transformer: um feitiço com espaço", lambda: transf(["ice storm"]))
    t.case("spell_transformer: string vazia", lambda: transf([""]))
    t.case("spell_transformer: lista vazia", lambda: transf([]))

    def stats(powers):
        mages = [{"name": "m%d" % i, "power": p, "element": "x"} for i, p in enumerate(powers)]
        got = val(t, EX0, "m.mage_stats(%r)" % (mages,))
        want = {"max_power": max(powers), "min_power": min(powers), "avg_power": round(sum(powers) / len(powers), 2)}
        t.expect(isinstance(got, dict) and set(got) == set(want),
                 "esperava um dict com as chaves max_power, min_power, avg_power, veio " + show(got))
        t.expect(got["max_power"] == want["max_power"] and got["min_power"] == want["min_power"]
                 and isinstance(got["avg_power"], (int, float)) and abs(got["avg_power"] - want["avg_power"]) < 1e-6,
                 "esperava %s, veio %s" % (show(want), show(got)))
        t.expect(type(got["avg_power"]) is float, "avg_power devia ser float, veio " + show(got["avg_power"]))

    t.case("mage_stats: 5 magos", lambda: stats([m["power"] for m in MAGES]))
    t.case("mage_stats: média arredondada a 2 casas (10, 20, 20)", lambda: stats([10, 20, 20]))
    t.case("mage_stats: média inteira (10, 20) é float", lambda: stats([10, 20]))
    t.case("mage_stats: um mago", lambda: stats([42]))
    t.case("mage_stats: negativos", lambda: stats([-7, -2, -30]))


# ---- ex1 ---------------------------------------------------------------

SPELLS = r'''
log = []
def fireball(target, power):
    log.append(("fireball", target, power))
    return "Fireball hits %s for %d" % (target, power)
def heal(target, power):
    log.append(("heal", target, power))
    return "Heal restores %s for %d HP" % (target, power)
def shield(target, power):
    log.append(("shield", target, power))
    return "Shield on %s (%d)" % (target, power)
def ko(msg):
    print("KO " + msg)
    raise SystemExit
'''


def ex1(t):
    t.group("ex1 higher_magic")
    t.case("só usa o permitido (sem eval/exec, só stdlib, sem functools/operator)",
           lambda: t.authorized(EX1, ALL_BUILTINS, STDLIB))
    signatures(t, EX1, {"spell_combiner": ["spell1", "spell2"], "power_amplifier": ["base_spell", "multiplier"],
                        "conditional_caster": ["condition", "spell"], "spell_sequence": ["spells"]})

    def sn(body):
        snippet(t, EX1, SPELLS + body)

    t.case("spell_combiner: devolve uma função sem lançar os feitiços", lambda: sn(r'''
c = m.spell_combiner(fireball, heal)
if not callable(c): ko("spell_combiner devia devolver uma função, veio %r" % (c,))
if log: ko("os feitiços foram lançados antes de chamar a função combinada: %r" % log)
print("OK")
'''))
    t.case("spell_combiner: tupla com os dois resultados", lambda: sn(r'''
r = m.spell_combiner(fireball, heal)("Dragon", 10)
w = (fireball("Dragon", 10), heal("Dragon", 10))
if r != w or type(r) is not tuple: ko("esperava %r, veio %r" % (w, r))
print("OK")
'''))
    t.case("spell_combiner: os dois recebem os mesmos argumentos, uma vez cada", lambda: sn(r'''
m.spell_combiner(heal, shield)("Orc", 7)
if sorted(log) != [("heal", "Orc", 7), ("shield", "Orc", 7)]: ko("chamadas esperadas heal/shield('Orc', 7), veio %r" % log)
print("OK")
'''))
    t.case("spell_combiner: funciona com lambdas", lambda: sn(r'''
r = m.spell_combiner(lambda t, p: p + 1, lambda t, p: t * 2)("ab", 1)
if r != (2, "abab"): ko("esperava (2, 'abab'), veio %r" % (r,))
print("OK")
'''))
    t.case("power_amplifier: exemplo (10 x 3 = 30)", lambda: sn(r'''
r = m.power_amplifier(fireball, 3)("Dragon", 10)
if r != "Fireball hits Dragon for 30": ko("esperava o feitiço com power 30, veio %r" % (r,))
print("OK")
'''))
    t.case("power_amplifier: multiplicador 1, 0 e power negativo", lambda: sn(r'''
for mult, p, w in ((1, 8, 8), (0, 8, 0), (4, -2, -8)):
    log.clear()
    m.power_amplifier(heal, mult)("Elf", p)
    if log != [("heal", "Elf", w)]: ko("power %d x %d: esperava heal('Elf', %d), veio %r" % (p, mult, w, log))
print("OK")
'''))
    t.case("power_amplifier: dois amplificadores independentes", lambda: sn(r'''
a3, a5 = m.power_amplifier(fireball, 3), m.power_amplifier(fireball, 5)
r = (a3("X", 2), a5("X", 2), a3("X", 2))
w = ("Fireball hits X for 6", "Fireball hits X for 10", "Fireball hits X for 6")
if r != w: ko("esperava %r, veio %r" % (w, r))
print("OK")
'''))
    t.case("conditional_caster: condição verdadeira lança o feitiço", lambda: sn(r'''
r = m.conditional_caster(lambda t, p: p >= 20, fireball)("Goblin", 30)
if r != "Fireball hits Goblin for 30": ko("esperava o resultado do feitiço, veio %r" % (r,))
print("OK")
'''))
    t.case("conditional_caster: condição falsa devolve 'Spell fizzled' sem lançar", lambda: sn(r'''
r = m.conditional_caster(lambda t, p: p >= 20, fireball)("Goblin", 5)
if r != "Spell fizzled": ko("esperava 'Spell fizzled', veio %r" % (r,))
if log: ko("o feitiço foi lançado mesmo com a condição falsa")
print("OK")
'''))
    t.case("conditional_caster: condição recebe os mesmos argumentos", lambda: sn(r'''
seen = []
def cond(target, power):
    seen.append((target, power))
    return target == "Troll"
c = m.conditional_caster(cond, heal)
r = (c("Troll", 3), c("Elf", 3))
if seen != [("Troll", 3), ("Elf", 3)]: ko("a condição devia receber (target, power), recebeu %r" % seen)
if r != ("Heal restores Troll for 3 HP", "Spell fizzled"): ko("esperava (cura, 'Spell fizzled'), veio %r" % (r,))
print("OK")
'''))
    t.case("spell_sequence: lista com os resultados, em ordem", lambda: sn(r'''
r = m.spell_sequence([shield, fireball, heal])("Knight", 7)
w = [shield("Knight", 7), fireball("Knight", 7), heal("Knight", 7)]
if r != w or type(r) is not list: ko("esperava %r, veio %r" % (w, r))
print("OK")
'''))
    t.case("spell_sequence: cada feitiço uma vez, com os mesmos argumentos", lambda: sn(r'''
s = m.spell_sequence([heal, heal, fireball])
if log: ko("os feitiços foram lançados antes de chamar a sequência")
s("Ent", 2)
if log != [("heal", "Ent", 2), ("heal", "Ent", 2), ("fireball", "Ent", 2)]: ko("chamadas: %r" % log)
print("OK")
'''))
    t.case("spell_sequence: lista vazia dá []", lambda: sn(r'''
r = m.spell_sequence([])("X", 1)
if r != []: ko("esperava [], veio %r" % (r,))
print("OK")
'''))


# ---- ex2 ---------------------------------------------------------------

def ex2(t):
    t.group("ex2 scope_mysteries")
    t.case("só usa o permitido (sem eval/exec, só stdlib, sem functools/operator)",
           lambda: t.authorized(EX2, ALL_BUILTINS, STDLIB))
    signatures(t, EX2, {"mage_counter": [], "spell_accumulator": ["initial_power"],
                        "enchantment_factory": ["enchantment_type"], "memory_vault": []})
    t.case("usa nonlocal", lambda: t.expect(any(isinstance(n, ast.Nonlocal) for n in ast.walk(t.tree(EX2))),
                                            "não encontrei nenhum nonlocal"))

    def sn(body):
        snippet(t, EX2, "def ko(msg):\n    print('KO ' + msg)\n    raise SystemExit\n" + body)

    def closure(expr):
        sn(r'''
f = %s
if not callable(f): ko("esperava uma função, veio %%r" %% (f,))
if not getattr(f, "__closure__", None): ko("a função devolvida não é uma closure (não captura variáveis)")
print("OK")
''' % expr)

    t.case("mage_counter: 1, 2, 3, 4, 5", lambda: sn(r'''
c = m.mage_counter()
r = [c() for _ in range(5)]
if r != [1, 2, 3, 4, 5]: ko("esperava [1, 2, 3, 4, 5], veio %r" % r)
print("OK")
'''))
    t.case("mage_counter: dois contadores independentes", lambda: sn(r'''
a, b = m.mage_counter(), m.mage_counter()
r = [a(), a(), b(), a(), b()]
if r != [1, 2, 1, 3, 2]: ko("a, a, b, a, b: esperava [1, 2, 1, 3, 2], veio %r" % r)
print("OK")
'''))
    t.case("mage_counter: devolve uma closure", lambda: closure("m.mage_counter()"))
    t.case("spell_accumulator: exemplo (100 + 20 = 120, + 30 = 150)", lambda: sn(r'''
acc = m.spell_accumulator(100)
r = [acc(20), acc(30)]
if r != [120, 150]: ko("esperava [120, 150], veio %r" % r)
print("OK")
'''))
    t.case("spell_accumulator: começa em 0, soma negativos e zero", lambda: sn(r'''
acc = m.spell_accumulator(0)
r = [acc(5), acc(-8), acc(0)]
if r != [5, -3, -3]: ko("esperava [5, -3, -3], veio %r" % r)
print("OK")
'''))
    t.case("spell_accumulator: dois acumuladores independentes", lambda: sn(r'''
a, b = m.spell_accumulator(100), m.spell_accumulator(10)
r = [a(1), b(1), a(1), b(5)]
if r != [101, 11, 102, 16]: ko("esperava [101, 11, 102, 16], veio %r" % r)
print("OK")
'''))
    t.case("spell_accumulator: devolve uma closure", lambda: closure("m.spell_accumulator(3)"))
    t.case("enchantment_factory: exemplo (Flaming Sword, Frozen Shield)", lambda: sn(r'''
f, g = m.enchantment_factory("Flaming"), m.enchantment_factory("Frozen")
r = [f("Sword"), g("Shield"), f("Axe")]
if r != ["Flaming Sword", "Frozen Shield", "Flaming Axe"]: ko("esperava ['Flaming Sword', 'Frozen Shield', 'Flaming Axe'], veio %r" % r)
print("OK")
'''))
    t.case("enchantment_factory: devolve uma closure", lambda: closure("m.enchantment_factory('Shiny')"))
    t.case("memory_vault: dict com 'store' e 'recall'", lambda: sn(r'''
v = m.memory_vault()
if not isinstance(v, dict) or set(v) != {"store", "recall"}: ko("esperava um dict com 'store' e 'recall', veio %r" % (v,))
if not (callable(v["store"]) and callable(v["recall"])): ko("'store' e 'recall' deviam ser funções")
print("OK")
'''))
    t.case("memory_vault: guarda e lembra; chave desconhecida", lambda: sn(r'''
v = m.memory_vault()
v["store"]("secret", 42)
r = [v["recall"]("secret"), v["recall"]("unknown")]
if r != [42, "Memory not found"]: ko("esperava [42, 'Memory not found'], veio %r" % r)
print("OK")
'''))
    t.case("memory_vault: sobrescreve e guarda valores falsy (0, '', [])", lambda: sn(r'''
v = m.memory_vault()
v["store"]("a", 1)
v["store"]("a", 2)
for k, x in (("zero", 0), ("empty", ""), ("list", [])):
    v["store"](k, x)
r = [v["recall"](k) for k in ("a", "zero", "empty", "list")]
if r != [2, 0, "", []]: ko("esperava [2, 0, '', []], veio %r" % r)
print("OK")
'''))
    t.case("memory_vault: dois cofres independentes (memória privada)", lambda: sn(r'''
a, b = m.memory_vault(), m.memory_vault()
a["store"]("k", "A")
r = b["recall"]("k")
if r != "Memory not found": ko("o segundo cofre enxerga o que foi guardado no primeiro: %r" % (r,))
print("OK")
'''))


# ---- ex3 ---------------------------------------------------------------

def ex3(t):
    t.group("ex3 functools_artifacts")
    t.case("só usa o permitido (sem eval/exec, só stdlib)",
           lambda: t.authorized(EX3, ALL_BUILTINS, STDLIB | {"functools", "operator"}))
    signatures(t, EX3, {"spell_reducer": ["spells", "operation"], "partial_enchanter": ["base_enchantment"],
                        "memoized_fibonacci": ["n"], "spell_dispatcher": []})
    t.case("spell_reducer usa functools.reduce",
           lambda: t.expect(calls(func(t, EX3, "spell_reducer"), "reduce"), "não encontrei reduce(...) em spell_reducer"))

    @t.case("usa operator.add e operator.mul")
    def _():
        tree = t.tree(EX3)
        t.expect(mentions(tree, "operator"), "não importa o módulo operator")
        t.expect(mentions(tree, "add") and mentions(tree, "mul"), "não usa operator.add e operator.mul")

    t.case("partial_enchanter usa functools.partial",
           lambda: t.expect(mentions(func(t, EX3, "partial_enchanter"), "partial"), "não encontrei partial em partial_enchanter"))
    t.case("memoized_fibonacci decorada com lru_cache",
           lambda: t.expect(any(mentions(d, "lru_cache") for d in func(t, EX3, "memoized_fibonacci").decorator_list),
                            "memoized_fibonacci não tem @lru_cache"))
    t.case("spell_dispatcher usa functools.singledispatch",
           lambda: t.expect(mentions(func(t, EX3, "spell_dispatcher"), "singledispatch"), "não encontrei singledispatch"))

    for spells in ([10, 20, 30, 40], [7], [-3, 5, -2]):
        for op, want in (("add", sum(spells)), ("multiply", math.prod(spells)),
                         ("max", max(spells)), ("min", min(spells))):
            t.case("spell_reducer(%s, %r) = %d" % (spells, op, want),
                   lambda s=spells, o=op, w=want: eq(t, EX3, "m.spell_reducer(%r, %r)" % (s, o), w))
    for op in ("add", "multiply", "max", "min"):
        t.case("spell_reducer([], %r) = 0" % op, lambda o=op: eq(t, EX3, "m.spell_reducer([], %r)" % o, 0))

    t.case("spell_reducer: operação desconhecida é tratada", lambda: snippet(t, EX3, r'''
try:
    r = m.spell_reducer([1, 2, 3], "divide")
except (TypeError, NameError, AttributeError, UnboundLocalError, IndexError) as e:
    print("KO operação desconhecida quebrou com %s: %s" % (type(e).__name__, e))
except Exception:
    print("OK")
else:
    print("OK" if r not in (6, 3, 1) else "KO operação 'divide' devolveu %r como se fosse válida" % (r,))
'''))

    BASE = r'''
import functools
calls = []
def base(power, element, target):
    calls.append((power, element, target))
    return "%s|%s|%s" % (power, element, target)
d = m.partial_enchanter(base)
'''
    t.case("partial_enchanter: dict com 3 versões functools.partial", lambda: snippet(t, EX3, BASE + r'''
if not isinstance(d, dict) or len(d) != 3: print("KO esperava um dict com 3 versões, veio %r" % (d,))
elif not all(isinstance(k, str) for k in d): print("KO as chaves deviam ser str: %r" % list(d))
elif not all(isinstance(v, functools.partial) for v in d.values()): print("KO as versões deviam ser functools.partial: %r" % d)
elif calls: print("KO a função base foi chamada ao criar as versões")
else: print("OK")
'''))
    t.case("partial_enchanter: power=50 e um elemento diferente em cada", lambda: snippet(t, EX3, BASE + r'''
r = [v(target="Dragon") for v in d.values()]
if any(c[0] != 50 or c[2] != "Dragon" for c in calls): print("KO esperava base(power=50, element, target='Dragon'), veio %r" % calls)
elif len({c[1] for c in calls}) != 3: print("KO os 3 elementos deviam ser diferentes: %r" % calls)
elif r != ["%s|%s|%s" % c for c in calls]: print("KO devia devolver o resultado da função base, veio %r" % r)
else: print("OK")
'''))

    for n, f in ((0, 0), (1, 1), (2, 1), (10, 55), (15, 610), (30, 832040)):
        t.case("memoized_fibonacci(%d) = %d" % (n, f), lambda n=n, f=f: eq(t, EX3, "m.memoized_fibonacci(%d)" % n, f))
    t.case("memoized_fibonacci(90) é rápido", lambda: eq(t, EX3, "m.memoized_fibonacci(90)", 2880067194370816120))
    t.case("memoized_fibonacci: o cache funciona (cache_info)", lambda: snippet(t, EX3, r'''
f = m.memoized_fibonacci
if not hasattr(f, "cache_info"): print("KO memoized_fibonacci não tem cache_info() (sem @lru_cache?)")
else:
    f.cache_clear()
    f(25)
    a = f.cache_info()
    f(25)
    b = f.cache_info()
    print("OK" if b.hits > a.hits and b.misses == a.misses else "KO repetir f(25) não usou o cache: %r -> %r" % (a, b))
'''))

    DISP = r'''
d = m.spell_dispatcher()
def ko(msg):
    print("KO " + msg)
    raise SystemExit
'''
    t.case("spell_dispatcher: singledispatch com int, str e list", lambda: snippet(t, EX3, DISP + r'''
reg = getattr(d, "registry", None)
if not callable(d) or reg is None: ko("spell_dispatcher() devia devolver uma função singledispatch, veio %r" % (d,))
miss = [c.__name__ for c in (int, str, list) if c not in reg]
if miss: ko("sem versão registrada para: " + ", ".join(miss))
print("OK")
'''))
    t.case("spell_dispatcher: int é feitiço de dano (42)", lambda: snippet(t, EX3, DISP + r'''
a, b = d(42), d(7)
if not (isinstance(a, str) and "42" in a and "7" in b): ko("esperava 'Damage spell: 42 damage', veio %r" % (a,))
print("OK")
'''))
    t.case("spell_dispatcher: str é encantamento (fireball)", lambda: snippet(t, EX3, DISP + r'''
a = d("fireball")
if not (isinstance(a, str) and "fireball" in a): ko("esperava 'Enchantment: fireball', veio %r" % (a,))
print("OK")
'''))
    t.case("spell_dispatcher: list é multi-cast (3 feitiços, 0 feitiços)", lambda: snippet(t, EX3, DISP + r'''
a, b = d(["a", "b", "c"]), d([])
if not (isinstance(a, str) and "3" in a): ko("esperava 'Multi-cast: 3 spells', veio %r" % (a,))
if not (isinstance(b, str) and "0" in b): ko("lista vazia: esperava 'Multi-cast: 0 spells', veio %r" % (b,))
print("OK")
'''))
    t.case("spell_dispatcher: tipo desconhecido (float, dict, None)", lambda: snippet(t, EX3, DISP + r'''
known = {d(42), d("fireball"), d([1, 2, 3])}
for x in (3.5, {"a": 1}, None):
    r = d(x)
    if not isinstance(r, str): ko("d(%r) devia devolver str, veio %r" % (x, r))
    if r in known: ko("d(%r) devolveu %r, igual a um tipo conhecido" % (x, r))
print("OK")
'''))


# ---- ex4 ---------------------------------------------------------------

def ex4(t):
    t.group("ex4 decorator_mastery")
    t.case("só usa o permitido (sem eval/exec, só stdlib, functools só para wraps)",
           lambda: t.authorized(EX4, ALL_BUILTINS, STDLIB | {"functools"}))

    @t.case("do functools só usa wraps")
    def _():
        bad = set()
        for n in ast.walk(t.tree(EX4)):
            if isinstance(n, ast.ImportFrom) and n.module == "functools":
                bad |= {a.name for a in n.names if a.name != "wraps"}
            elif isinstance(n, ast.Attribute) and isinstance(n.value, ast.Name) and n.value.id == "functools" \
                    and n.attr != "wraps":
                bad.add(n.attr)
        t.expect(not bad, "usa do functools: " + ", ".join(sorted(bad)))

    signatures(t, EX4, {"spell_timer": ["func"], "power_validator": ["min_power"], "retry_spell": ["max_attempts"]})
    t.case("MageGuild.cast_spell(self, spell_name, power)",
           lambda: sig(t, method(t, EX4, "MageGuild", "cast_spell"), ["self", "spell_name", "power"]))

    @t.case("MageGuild.validate_mage_name é @staticmethod (name)")
    def _():
        node = method(t, EX4, "MageGuild", "validate_mage_name")
        t.expect(any(mentions(d, "staticmethod") for d in node.decorator_list), "validate_mage_name não tem @staticmethod")
        sig(t, node, ["name"])

    @t.case("cast_spell decorado com @power_validator(10)")
    def _():
        node = method(t, EX4, "MageGuild", "cast_spell")
        ok = [d for d in node.decorator_list if isinstance(d, ast.Call) and mentions(d.func, "power_validator")
              and [a.value for a in d.args if isinstance(a, ast.Constant)] + [k.value.value for k in d.keywords
                                                                               if isinstance(k.value, ast.Constant)] == [10]]
        t.expect(ok, "não encontrei @power_validator(10) (ou min_power=10) em cast_spell")

    def sn(body, timeout=5):
        snippet(t, EX4, "import re\ndef ko(msg):\n    print('KO ' + msg)\n    raise SystemExit\n" + body, timeout)

    WRAPS = r'''
if f.__name__ != "secret_spell": ko("o decorador perdeu o nome da função: __name__ = %r (use functools.wraps)" % f.__name__)
if f.__doc__ != "Secret docs.": ko("o decorador perdeu a docstring: %r (use functools.wraps)" % (f.__doc__,))
if not hasattr(f, "__wrapped__"): ko("sem __wrapped__ (use functools.wraps)")
print("OK")
'''
    t.case("spell_timer: imprime 'Casting nome...' antes e 'Spell completed in X.XXX seconds' depois", lambda: sn(r'''
def fireball():
    print("INSIDE")
    return "Fireball cast!"
r, out = _out(m.spell_timer(fireball))
ls = [x.strip() for x in out.splitlines() if x.strip()]
if r != "Fireball cast!": ko("devia devolver o resultado da função, veio %r" % (r,))
if "INSIDE" not in ls: ko("a função decorada não foi executada: %r" % out)
i = ls.index("INSIDE")
if not any(re.search(r"casting\s+fireball", x, re.I) for x in ls[:i]): ko("esperava 'Casting fireball...' antes da execução: %r" % out)
if not any(re.search(r"spell completed in \d+\.\d{3} seconds", x, re.I) for x in ls[i + 1:]): ko("esperava 'Spell completed in X.XXX seconds' depois da execução: %r" % out)
print("OK")
'''))
    t.case("spell_timer: mede o tempo de verdade (sleep 0.2)", lambda: sn(r'''
import time
def slow():
    time.sleep(0.2)
r, out = _out(m.spell_timer(slow))
s = re.findall(r"(\d+\.\d+)\s*seconds", out)
if not s: ko("não encontrei 'X.XXX seconds' em %r" % out)
if not 0.18 <= float(s[-1]) <= 2: ko("dormiu 0.2s mas mediu %s s" % s[-1])
print("OK")
'''))
    t.case("spell_timer: repassa argumentos posicionais e nomeados", lambda: sn(r'''
r, out = _out(m.spell_timer(lambda a, b, c=0: (a, b, c)), 1, 2, c=3)
if r != (1, 2, 3): ko("esperava (1, 2, 3), veio %r" % (r,))
print("OK")
'''))
    t.case("spell_timer: preserva nome e docstring (functools.wraps)", lambda: sn(r'''
def secret_spell():
    """Secret docs."""
f = m.spell_timer(secret_spell)
''' + WRAPS))

    VAL = r'''
calls = []
def spell(target, power):
    calls.append((target, power))
    return "%s takes %d" % (target, power)
v = m.power_validator(10)(spell)
'''
    t.case("power_validator: power suficiente executa (15 e 10 >= 10)", lambda: sn(VAL + r'''
r = (v("Orc", 15), v("Orc", 10))
if r != ("Orc takes 15", "Orc takes 10"): ko("esperava ('Orc takes 15', 'Orc takes 10'), veio %r" % (r,))
print("OK")
'''))
    t.case("power_validator: power insuficiente não executa", lambda: sn(VAL + r'''
r = (v("Orc", 9), v("Orc", -5))
if r != ("Insufficient power for this spell",) * 2: ko("esperava 'Insufficient power for this spell', veio %r" % (r,))
if calls: ko("a função foi executada com power insuficiente: %r" % calls)
print("OK")
'''))
    t.case("power_validator: cada min_power tem seu próprio limite", lambda: sn(r'''
lo = m.power_validator(1)(lambda target, power: "ok")
hi = m.power_validator(100)(lambda target, power: "ok")
r = (lo("X", 50), hi("X", 50))
if r != ("ok", "Insufficient power for this spell"): ko("power 50 com min 1 e min 100: esperava ('ok', 'Insufficient power for this spell'), veio %r" % (r,))
print("OK")
'''))
    t.case("power_validator: preserva nome e docstring (functools.wraps)", lambda: sn(r'''
def secret_spell(target, power):
    """Secret docs."""
f = m.power_validator(5)(secret_spell)
''' + WRAPS))

    RETRY = r'''
n = [0]
def flaky(ok_at):
    def spell():
        n[0] += 1
        if n[0] < ok_at:
            raise RuntimeError("boom")
        return "Waaaaaaagh spelled !"
    return spell
def attempts(out):
    return [tuple(map(int, x)) for x in re.findall(r"attempt\s*(\d+)\s*/\s*(\d+)", out)]
'''
    t.case("retry_spell: falha sempre (3 tentativas)", lambda: sn(RETRY + r'''
r, out = _out(m.retry_spell(3)(flaky(99)))
if n[0] != 3: ko("a função devia ser tentada 3 vezes, foi %d" % n[0])
if str(r).strip().rstrip(".!").lower() != "spell casting failed after 3 attempts": ko("esperava 'Spell casting failed after 3 attempts', veio %r" % (r,))
a = attempts(out)
if a not in ([(1, 3), (2, 3)], [(1, 3), (2, 3), (3, 3)]): ko("esperava 'Spell failed, retrying... (attempt 1/3)', '(attempt 2/3)', veio %r" % out)
if "spell failed, retrying" not in out.lower(): ko("esperava 'Spell failed, retrying...': %r" % out)
print("OK")
'''))
    t.case("retry_spell: funciona na 2ª tentativa", lambda: sn(RETRY + r'''
r, out = _out(m.retry_spell(4)(flaky(2)))
if r != "Waaaaaaagh spelled !": ko("devia devolver o resultado da tentativa que deu certo, veio %r" % (r,))
if n[0] != 2: ko("devia parar na tentativa que deu certo; tentou %d vezes" % n[0])
if attempts(out) != [(1, 4)]: ko("esperava só '(attempt 1/4)', veio %r" % out)
print("OK")
'''))
    t.case("retry_spell: sucesso de primeira não imprime falha", lambda: sn(RETRY + r'''
r, out = _out(m.retry_spell(3)(flaky(1)), )
if r != "Waaaaaaagh spelled !" or n[0] != 1: ko("esperava o resultado em 1 tentativa, veio %r após %d" % (r, n[0]))
if attempts(out) or "fail" in out.lower(): ko("não devia imprimir falha: %r" % out)
print("OK")
'''))
    t.case("retry_spell: repassa argumentos e max_attempts=1", lambda: sn(RETRY + r'''
r, out = _out(m.retry_spell(2)(lambda a, b=0: a + b), 1, b=2)
if r != 3: ko("esperava 3, veio %r" % (r,))
r, out = _out(m.retry_spell(1)(flaky(99)))
if n[0] != 1 or "1 attempt" not in str(r): ko("max_attempts=1: esperava 1 tentativa e 'failed after 1 attempts', veio %r após %d" % (r, n[0]))
print("OK")
'''))

    GUILD = r'''
G = m.MageGuild
try:
    g = G()
except TypeError:
    g = object.__new__(G)
'''
    t.case("validate_mage_name é staticmethod de verdade", lambda: sn(GUILD + r'''
if not isinstance(G.__dict__.get("validate_mage_name"), staticmethod): ko("validate_mage_name não é staticmethod")
if g.validate_mage_name("Gandalf") is not True or G.validate_mage_name("Gandalf") is not True: ko("'Gandalf' devia ser válido pela classe e pela instância")
print("OK")
'''))
    for name, want in (("Gandalf", True), ("Ali", True), ("Merlin the Wise", True), ("Al", False), ("", False),
                       ("R2D2", False), ("Mer-lin", False), ("Jean_Luc", False), ("Bob!", False)):
        t.case("validate_mage_name(%r) = %s" % (name, want), lambda n=name, w=want: sn(GUILD + r'''
r = G.validate_mage_name(%r)
if r is not %s: ko("esperava %s, veio %%r" %% (r,))
print("OK")
''' % (n, w, w)))
    for spell, power, want in (("Lightning", 15, "Successfully cast Lightning with 15 power"),
                               ("Fire", 10, "Successfully cast Fire with 10 power"),
                               ("Spark", 9, "Insufficient power for this spell"),
                               ("Spark", 0, "Insufficient power for this spell")):
        t.case("cast_spell(%r, %d)" % (spell, power), lambda s=spell, p=power, w=want: sn(GUILD + r'''
r = g.cast_spell(%r, %d)
if r != %r: ko("esperava %r, veio %%r" %% (r,))
print("OK")
''' % (s, p, w, w)))
