"""Python Module 07 (DataDeck): abstract factory, capabilities, strategy."""

import ast
import builtins
import glob
import os
import re

from lt import common, show

EXS = ["ex0", "ex1", "ex2"]
MAIN = {"ex0": "battle.py", "ex1": "capacitor.py", "ex2": "tournament.py"}
ALLOWED = set(dir(builtins)) - {"eval", "exec"}
CONCRETE = ["Flameling", "Pyrodon", "Aquabub", "Torragon",
            "Sproutling", "Bloomelle", "Shiftling", "Morphagon"]
# (factory, kind) -> (class it must create, type shown by describe())
FAMILY = {
    ("FlameFactory", "base"): ("Flameling", "Fire"),
    ("FlameFactory", "evolved"): ("Pyrodon", "Fire/Flying"),
    ("AquaFactory", "base"): ("Aquabub", "Water"),
    ("AquaFactory", "evolved"): ("Torragon", "Water"),
    ("HealingCreatureFactory", "base"): ("Sproutling", "Grass"),
    ("HealingCreatureFactory", "evolved"): ("Bloomelle", "Grass/Fairy"),
    ("TransformCreatureFactory", "base"): ("Shiftling", "Normal"),
    ("TransformCreatureFactory", "evolved"): ("Morphagon", "Normal/Dragon"),
}
FACTORIES = {"ex0": ["FlameFactory", "AquaFactory"],
             "ex1": ["HealingCreatureFactory", "TransformCreatureFactory"]}
HEAL = [k for k in FAMILY if k[0] == "HealingCreatureFactory"]
MORPH = [k for k in FAMILY if k[0] == "TransformCreatureFactory"]
STRATS = ["NormalStrategy", "AggressiveStrategy", "DefensiveStrategy"]

# Runs in the student's process, with the project root on sys.path. Each
# fact(key, fn) stores what fn returned (or raised, and what it printed) in R,
# so one broken piece fails only the cases that use it.
PRELUDE = r'''
import contextlib, importlib, inspect, io, pkgutil
R = {}
CLS = {}

def lit(v):
    if v is None or isinstance(v, (bool, int, float, str)):
        return v
    if isinstance(v, (list, tuple, set, frozenset)):
        return [lit(x) for x in v]
    if isinstance(v, dict):
        return {str(k): lit(x) for k, x in v.items()}
    return repr(v)

def err(e):
    return {"type": type(e).__name__, "msg": str(e)[:300], "module": type(e).__module__,
            "mro": [c.__name__ for c in type(e).__mro__]}

def fact(key, fn):
    buf = io.StringIO()
    try:
        with contextlib.redirect_stdout(buf):
            v = fn()
        R[key] = {"ok": True, "v": lit(v), "out": buf.getvalue()}
    except BaseException as e:
        R[key] = dict(err(e), ok=False, out=buf.getvalue())

def load(name):
    with contextlib.redirect_stdout(io.StringIO()):
        p = importlib.import_module(name)
        mods = [p]
        for info in pkgutil.walk_packages(list(getattr(p, "__path__", [])), name + "."):
            mods.append(importlib.import_module(info.name))
    for m in mods:
        for k, v in list(vars(m).items()):
            if inspect.isclass(v) and (v.__module__ or "").split(".")[0] == name:
                CLS.setdefault(k, v)
    return {"file": getattr(p, "__file__", None),
            "exposed": sorted(k for k in vars(p) if not k.startswith("_"))}

def cls(name):
    if name not in CLS:
        raise LookupError("classe %s não encontrada no pacote" % name)
    return CLS[name]

def new(c):
    try:
        c()
    except Exception as e:
        return "%s: %s" % (type(e).__name__, e)
    return "ok"

def abstract(name):
    c = cls(name)
    return {"abstract": inspect.isabstract(c), "methods": sorted(getattr(c, "__abstractmethods__", ())),
            "new": new(c), "creature": "Creature" in CLS and issubclass(c, CLS["Creature"])}

def isinst(o, name):
    return name in CLS and isinstance(o, CLS[name])

def strs(o):
    out = []
    for a in dir(o):
        if not a.startswith("__"):
            try:
                v = getattr(o, a)
            except Exception:
                continue
            if isinstance(v, str):
                out.append(v)
    return out

def state(o):
    s = {}
    for a in dir(o):
        if not a.startswith("__"):
            try:
                v = getattr(o, a)
            except Exception:
                continue
            if not callable(v):
                s[a] = repr(v)
    return s

def nreq(m):
    try:
        ps = inspect.signature(m).parameters.values()
    except (TypeError, ValueError):
        return 0
    return len([p for p in ps if p.default is p.empty and p.kind in (p.POSITIONAL_ONLY, p.POSITIONAL_OR_KEYWORD)])

def make(fac, kind):
    return getattr(cls(fac)(), "create_" + kind)()

def info(fac, kind):
    o = make(fac, kind)
    return {"cls": type(o).__name__, "mro": [c.__name__ for c in type(o).__mro__],
            "creature": isinst(o, "Creature"), "heal_cap": isinst(o, "HealCapability"),
            "transform_cap": isinst(o, "TransformCapability"),
            "describe": o.describe(), "attack": o.attack(), "strs": strs(o)}

def heal(fac, kind):
    o = make(fac, kind)
    return o.heal(*[o] * nreq(o.heal))

def morph(fac, kind):
    o, other = make(fac, kind), make(fac, kind)
    a1, s1 = o.attack(), state(o)
    t = o.transform()
    a2, s2, oa = o.attack(), state(o), other.attack()
    r = o.revert()
    return {"a1": a1, "t": t, "a2": a2, "r": r, "a3": o.attack(), "other": oa, "changed": s1 != s2}

def factory(name):
    return {"sub": "CreatureFactory" in CLS and issubclass(cls(name), CLS["CreatureFactory"]),
            "new": new(cls(name))}

def text(v):
    if v is None:
        return ""
    if isinstance(v, (list, tuple)):
        return "\n".join(str(x) for x in v)
    return str(v)

def act(strat, fac, kind):
    o = make(fac, kind)
    log = []
    for n in ("attack", "heal", "transform", "revert", "describe"):
        m = getattr(o, n, None)
        if callable(m):
            def w(*a, _m=m, _n=n, **k):
                log.append(_n)
                return _m(*a, **k)
            setattr(o, n, w)
    s = cls(strat)()
    buf = io.StringIO()
    res = {"err": None, "ret": ""}
    try:
        with contextlib.redirect_stdout(buf):
            res["ret"] = text(s.act(*[o] * max(1, nreq(s.act))))
    except Exception as e:
        res["err"] = err(e)
    res["log"] = list(log)
    res["out"] = buf.getvalue()
    with contextlib.redirect_stdout(io.StringIO()):
        res["after"] = o.attack()
    return res

def valid(strat, o):
    return cls(strat)().is_valid(o)

def fake(cap):
    base = cls("Creature")
    c = cls(cap)
    names = set(getattr(base, "__abstractmethods__", ())) | set(getattr(c, "__abstractmethods__", ()))
    ns = {n: (lambda self, *a, **k: "fake") for n in names}
    ns["__init__"] = lambda self, *a, **k: None
    return type("LtFake" + cap, (base, c), ns)()
'''

BODY = {}
BODY["ex0"] = r'''
fact("load", lambda: load("ex0"))
for n in ("Creature", "CreatureFactory"):
    fact(n, lambda n=n: abstract(n))
fact("describe", lambda: {"own": "describe" in vars(cls("Creature")),
                          "abstract": "describe" in getattr(cls("Creature"), "__abstractmethods__", ())})
for f in ("FlameFactory", "AquaFactory"):
    fact(f, lambda f=f: factory(f))
    for k in ("base", "evolved"):
        fact(f + "." + k, lambda f=f, k=k: info(f, k))
'''
BODY["ex1"] = r'''
fact("load0", lambda: load("ex0"))
fact("load", lambda: load("ex1"))
for n in ("HealCapability", "TransformCapability"):
    fact(n, lambda n=n: abstract(n))
for f in ("HealingCreatureFactory", "TransformCreatureFactory"):
    fact(f, lambda f=f: factory(f))
    for k in ("base", "evolved"):
        fact(f + "." + k, lambda f=f, k=k: info(f, k))
for k in ("base", "evolved"):
    fact("heal." + k, lambda k=k: heal("HealingCreatureFactory", k))
    fact("morph." + k, lambda k=k: morph("TransformCreatureFactory", k))
'''
BODY["ex2"] = r'''
for p in ("ex0", "ex1", "ex2"):
    fact("load" + p[-1], lambda p=p: load(p))
R["load"] = R["load2"]
fact("BattleStrategy", lambda: abstract("BattleStrategy"))
FAM = [(f, k) for f in ("FlameFactory", "AquaFactory", "HealingCreatureFactory", "TransformCreatureFactory")
       for k in ("base", "evolved")]
for s in STRATS:
    fact(s, lambda s=s: {"sub": "BattleStrategy" in CLS and issubclass(cls(s), CLS["BattleStrategy"]),
                         "new": new(cls(s))})
    for f, k in FAM:
        fact("valid.%s.%s.%s" % (s, f, k), lambda s=s, f=f, k=k: valid(s, make(f, k)))
        fact("act.%s.%s.%s" % (s, f, k), lambda s=s, f=f, k=k: act(s, f, k))
    for cap in ("HealCapability", "TransformCapability"):
        fact("fake.%s.%s" % (s, cap), lambda s=s, cap=cap: valid(s, fake(cap)))
for f, k in FAM:
    fact(f + "." + k, lambda f=f, k=k: info(f, k))
fact("heal", lambda: heal("HealingCreatureFactory", "base"))
fact("morph", lambda: morph("TransformCreatureFactory", "base"))
'''


def pkg_files(t, ex):
    """The .py files of package ex, __init__.py first (even if missing)."""
    found = sorted(os.path.relpath(p, t.root) for p in glob.glob(os.path.join(t.root, ex, "**", "*.py"), recursive=True))
    init = ex + "/__init__.py"
    return [init] + [f for f in found if f != init]


def seq(out, items):
    """Index of the first item not found in order in out, or None."""
    pos = 0
    for i, s in enumerate(items):
        p = out.find(s, pos)
        if p < 0:
            return i
        pos = p + len(s)
    return None


def suite(t):
    pkg = {ex: pkg_files(t, ex) for ex in EXS}
    files = []
    for ex in EXS:
        files += [MAIN[ex]] + pkg[ex]
    common(t, files, typed=files)

    cache = {}

    def facts(ex):
        if ex not in cache:
            try:
                r = t.py(PRELUDE + "STRATS = %r\n" % STRATS + BODY[ex] + '\nprint("@@" + repr(R))\n',
                         ex=".", timeout=20)
                lines = [ln for ln in r.out.splitlines() if ln.startswith("@@")]
                t.expect(lines, "o teste não terminou: %s" % show((r.out + r.err)[-300:]))
                cache[ex] = ast.literal_eval(lines[-1][2:])
            except Exception as e:
                cache[ex] = e
        if isinstance(cache[ex], Exception):
            raise cache[ex]
        return cache[ex]

    def raw(ex, key):
        r = facts(ex).get(key)
        t.expect(r is not None, "o teste não chegou a %s" % key)
        return r

    def ok(r, what):
        if not r["ok"]:
            t.expect(False, "%s falhou: %s: %s" % (what, r["type"], r["msg"]))

    def get(ex, key, what=None):
        r = raw(ex, key)
        ok(r, what or key)
        return r["v"]

    def run(rel):
        r = t.run(rel, timeout=10)
        t.expect(r.code == 0, "o programa terminou com código %d: %s" % (r.code, show(r.err[-300:])))
        t.expect(r.out.strip(), "o programa não imprimiu nada")
        return r.out

    def returns_str(v, call, r):
        t.expect(isinstance(v, str) and v.strip(),
                 "%s deve retornar uma string, retornou %s%s"
                 % (call, show(v), " (e imprimiu %s: imprime em vez de retornar?)" % show(r["out"]) if r["out"] else ""))

    def creature(ex, fac, kind, need=("describe", "attack")):
        """The creature's info dict; fails unless the methods in need return strings."""
        r = raw(ex, fac + "." + kind)
        name = FAMILY[(fac, kind)][0]
        ok(r, "%s().create_%s()" % (fac, kind))
        v = r["v"]
        for m in need:
            returns_str(v[m], "%s.%s()" % (name, m), r)
        return v

    # ---- shared ------------------------------------------------------------

    def package_cases(ex):
        @t.case("%s é um pacote (tem __init__.py)" % ex)
        def _():
            t.need(ex + "/__init__.py")
            v = get(ex, "load", "import " + ex)
            t.expect(v["file"], "%s não foi importado como pacote normal" % ex)

        @t.case("import %s funciona (e todos os módulos do pacote)" % ex)
        def _():
            get(ex, "load", "import " + ex)

    def authorized_cases(ex):
        own = EXS[:EXS.index(ex) + 1]
        imports = {"typing", "abc", "__future__"} | set(own)
        # lt.authorized ignores the dots of relative imports: allow the
        # package's own module names ("from .creature import ...").
        for p in own:
            for f in glob.glob(os.path.join(t.root, p, "**", "*"), recursive=True):
                imports.add(os.path.splitext(os.path.basename(f))[0])
        for rel in [MAIN[ex]] + pkg[ex]:
            if t.exists(rel):
                t.case("só importa typing, abc e os pacotes do projeto, sem eval/exec: " + rel,
                       lambda rel=rel: t.authorized(rel, allowed=ALLOWED, imports=imports))

    def no_concrete_import(ex):
        rel = MAIN[ex]

        @t.case("%s não importa Creature concretas (o pacote só expõe fábricas)" % rel)
        def _():
            t.need(rel)
            names = {a.name for n in ast.walk(t.tree(rel)) if isinstance(n, ast.ImportFrom) for a in n.names}
            bad = sorted(names & set(CONCRETE))
            t.expect(not bad, "importa %s direto" % ", ".join(bad))

    def exposure_cases(ex):
        @t.case("%s expõe %s" % (ex, " e ".join(FACTORIES[ex])))
        def _():
            v = get(ex, "load", "import " + ex)
            miss = [f for f in FACTORIES[ex] if f not in v["exposed"]]
            t.expect(not miss, "from %s import %s não funciona" % (ex, ", ".join(miss)))

        @t.case("%s não expõe Creature concretas, só fábricas" % ex)
        def _():
            v = get(ex, "load", "import " + ex)
            bad = [c for c in CONCRETE if c in v["exposed"]]
            t.expect(not bad, "%s.%s está acessível direto no pacote" % (ex, ", ".join(bad)))

    def abstract_cases(ex, name, methods, from_creature=None):
        @t.case("%s é abstrata (%s() lança TypeError)" % (name, name))
        def _():
            v = get(ex, name)
            t.expect(v["abstract"] and v["new"].startswith("TypeError"),
                     "%s() deveria lançar TypeError (classe abstrata), resultado: %s" % (name, show(v["new"])))

        @t.case("%s tem %s como método%s abstrato%s" % (name, " e ".join(methods),
                                                       "s" if len(methods) > 1 else "", "s" if len(methods) > 1 else ""))
        def _():
            v = get(ex, name)
            miss = [m for m in methods if m not in v["methods"]]
            t.expect(not miss, "esperado %s abstrato(s) (@abstractmethod), abstratos: %s" % (", ".join(miss), v["methods"]))

        if from_creature is False:
            @t.case("%s não herda de Creature" % name)
            def _():
                v = get(ex, name)
                t.expect(not v["creature"], "%s herda de Creature; a capacidade deve ser separada" % name)

    def factory_cases(ex, fac):
        @t.case("%s herda de CreatureFactory (do ex0) e instancia sem argumentos" % fac)
        def _():
            v = get(ex, fac)
            t.expect(v["sub"], "%s não herda de CreatureFactory%s" % (fac, " do ex0" if ex != "ex0" else ""))
            t.expect(v["new"] == "ok", "%s() falhou: %s" % (fac, show(v["new"])))

        bases = []
        for kind in ("base", "evolved"):
            name, typ = FAMILY[(fac, kind)]
            bases.append(name)

            @t.case("%s().create_%s() cria um %s (uma Creature)" % (fac, kind, name))
            def _(kind=kind, name=name):
                r = raw(ex, fac + "." + kind)
                ok(r, "%s().create_%s()" % (fac, kind))
                v = r["v"]
                t.expect(v["cls"] == name, "esperado um %s, recebido um %s" % (name, v["cls"]))
                t.expect(v["creature"], "%s não herda de Creature%s (mro: %s)"
                         % (name, " do ex0" if ex != "ex0" else "", " > ".join(v["mro"])))

            @t.case("%s.describe() retorna nome e tipo (%s)" % (name, typ))
            def _(kind=kind, name=name, typ=typ):
                v = creature(ex, fac, kind)
                d = v["describe"]
                t.expect(name in d and typ in d, "esperado algo como %s, recebido %s"
                         % (show("%s is a %s type Creature" % (name, typ)), show(d)))

            @t.case("%s.attack() retorna uma string com o nome" % name)
            def _(kind=kind, name=name):
                v = creature(ex, fac, kind, ("attack",))
                t.expect(name in v["attack"], "esperado o nome %s no ataque, recebido %s" % (show(name), show(v["attack"])))
                t.expect(v["attack"] != v["describe"], "attack() retorna o mesmo que describe(): %s" % show(v["attack"]))

            @t.case("%s guarda nome e tipo em atributos" % name)
            def _(kind=kind, name=name, typ=typ):
                v = creature(ex, fac, kind, ())
                t.expect(name in v["strs"], "nenhum atributo com o nome %s" % show(name))
                t.expect(any(typ in s for s in v["strs"] if s != v["describe"]), "nenhum atributo com o tipo %s" % show(typ))

    # ---- ex0 ---------------------------------------------------------------

    t.group("ex0 Creature Factory")
    package_cases("ex0")
    exposure_cases("ex0")
    abstract_cases("ex0", "Creature", ["attack"])

    @t.case("describe é um método concreto de Creature")
    def _():
        v = get("ex0", "describe")
        t.expect(v["own"], "Creature não define describe()")
        t.expect(not v["abstract"], "describe() não pode ser abstrato (é o método genérico)")

    abstract_cases("ex0", "CreatureFactory", ["create_base", "create_evolved"])
    for fac in FACTORIES["ex0"]:
        factory_cases("ex0", fac)
    authorized_cases("ex0")
    no_concrete_import("ex0")

    def func_calls(fn):
        return {n.func.attr for n in ast.walk(fn) if isinstance(n, ast.Call) and isinstance(n.func, ast.Attribute)}

    def funcs(rel):
        return [n for n in ast.walk(t.tree(rel)) if isinstance(n, ast.FunctionDef)]

    @t.case("battle.py tem uma função que recebe uma fábrica e cria base e evolução")
    def _():
        t.need("battle.py")
        ok = [f for f in funcs("battle.py") if {"create_base", "create_evolved"} <= func_calls(f)]
        t.expect(ok, "nenhuma função de battle.py chama create_base() e create_evolved()")

    @t.case("battle.py tem uma função de batalha que recebe as duas fábricas")
    def _():
        t.need("battle.py")
        ok = [f for f in funcs("battle.py") if "create_base" in func_calls(f)
              and (len(f.args.args) >= 2 or f.args.vararg is not None)]
        t.expect(ok, "nenhuma função de battle.py com 2 fábricas como parâmetro chama create_base()")

    @t.case("python3 battle.py roda sem erro")
    def _():
        run("battle.py")

    for fac in FACTORIES["ex0"]:
        @t.case("battle.py testa a %s: describe e attack da base e da evolução" % fac)
        def _(fac=fac):
            b, e = creature("ex0", fac, "base"), creature("ex0", fac, "evolved")
            out = run("battle.py")
            items = [b["describe"], b["attack"], e["describe"], e["attack"]]
            i = seq(out, items)
            t.expect(i is None, "esperado %s (nessa ordem), faltou %s, saída: %s"
                     % (show(items), show(items[i] if i is not None else ""), show(out)))

    @t.case("battle.py faz as Creature base das duas fábricas lutarem")
    def _():
        f, a = creature("ex0", "FlameFactory", "base"), creature("ex0", "AquaFactory", "base")
        te = creature("ex0", "AquaFactory", "evolved")
        out = run("battle.py")
        start = out.find(te["attack"])
        t.expect(start >= 0, "a AquaFactory não foi testada antes da batalha: %s" % show(out))
        rest = out[start + len(te["attack"]):]
        ds = [rest.find(f["describe"]), rest.find(a["describe"])]
        t.expect(min(ds) >= 0, "a batalha deve mostrar %s e %s, depois dos testes veio %s"
                 % (show(f["describe"]), show(a["describe"]), show(rest)))
        after = rest[max(ds):]
        t.expect(f["attack"] in after and a["attack"] in after,
                 "depois de apresentar os dois, esperado %s e %s, veio %s" % (show(f["attack"]), show(a["attack"]), show(after)))

    # ---- ex1 ---------------------------------------------------------------

    t.group("ex1 Capabilities")
    package_cases("ex1")
    exposure_cases("ex1")
    abstract_cases("ex1", "HealCapability", ["heal"], from_creature=False)
    abstract_cases("ex1", "TransformCapability", ["transform", "revert"], from_creature=False)
    for fac in FACTORIES["ex1"]:
        factory_cases("ex1", fac)
    for (fac, kind), cap in [(k, "HealCapability") for k in HEAL] + [(k, "TransformCapability") for k in MORPH]:
        name = FAMILY[(fac, kind)][0]

        @t.case("%s herda de Creature e de %s" % (name, cap))
        def _(fac=fac, kind=kind, name=name, cap=cap):
            v = get("ex1", fac + "." + kind)
            t.expect(v["creature"], "%s não herda de Creature do ex0 (mro: %s)" % (name, " > ".join(v["mro"])))
            t.expect(v["heal_cap" if cap == "HealCapability" else "transform_cap"],
                     "%s não herda de %s (mro: %s)" % (name, cap, " > ".join(v["mro"])))

    for fac, kind in HEAL:
        name = FAMILY[(fac, kind)][0]

        @t.case("%s.heal() retorna uma string com o nome" % name)
        def _(kind=kind, name=name):
            r = raw("ex1", "heal." + kind)
            ok(r, "%s.heal()" % (name))
            returns_str(r["v"], name + ".heal()", r)
            t.expect(name in r["v"], "esperado o nome %s na cura, recebido %s" % (show(name), show(r["v"])))

    for fac, kind in MORPH:
        name = FAMILY[(fac, kind)][0]

        def morph(kind, name):
            r = raw("ex1", "morph." + kind)
            ok(r, "attack/transform/revert de %s" % (name))
            v = r["v"]
            for k, call in [("a1", "attack()"), ("t", "transform()"), ("a2", "attack()"), ("r", "revert()")]:
                returns_str(v[k], name + "." + call, r)
            return v

        @t.case("%s.transform() e revert() retornam strings diferentes com o nome" % name)
        def _(kind=kind, name=name):
            v = morph(kind, name)
            t.expect(name in v["t"] and name in v["r"], "esperado o nome %s em %s e %s" % (show(name), show(v["t"]), show(v["r"])))
            t.expect(v["t"] != v["r"], "transform() e revert() retornam o mesmo: %s" % show(v["t"]))

        @t.case("%s: transform() muda o attack()" % name)
        def _(kind=kind, name=name):
            v = morph(kind, name)
            t.expect(v["a2"] != v["a1"], "attack() é igual antes e depois de transform(): %s" % show(v["a1"]))
            t.expect(name in v["a2"], "esperado o nome %s no ataque transformado, recebido %s" % (show(name), show(v["a2"])))

        @t.case("%s: revert() volta ao attack() normal" % name)
        def _(kind=kind, name=name):
            v = morph(kind, name)
            t.expect(v["a3"] == v["a1"], "depois de revert(), esperado %s, recebido %s" % (show(v["a1"]), show(v["a3"])))

        @t.case("%s: o estado da transformação fica num atributo da instância" % name)
        def _(kind=kind, name=name):
            v = morph(kind, name)
            t.expect(v["changed"], "nenhum atributo muda com transform()")
            t.expect(v["other"] == v["a1"], "transformar um %s mudou o ataque de outro %s: %s"
                     % (name, name, show(v["other"])))

    authorized_cases("ex1")
    no_concrete_import("ex1")

    @t.case("python3 capacitor.py roda sem erro")
    def _():
        run("capacitor.py")

    def heal_mark(ex, kind, key):
        """The heal text if it names the creature, else the creature name."""
        name = FAMILY[("HealingCreatureFactory", kind)][0]
        r = raw(ex, key)
        return name if not r["ok"] or not isinstance(r["v"], str) or name in r["v"] else r["v"]

    @t.case("capacitor.py: base e evolução curativas se descrevem, atacam e curam")
    def _():
        items = []
        for fac, kind in HEAL:
            v = creature("ex1", fac, kind)
            items += [v["describe"], v["attack"], heal_mark("ex1", kind, "heal." + kind)]
        out = run("capacitor.py")
        i = seq(out, items)
        t.expect(i is None, "esperado %s (nessa ordem), faltou %s, saída: %s"
                 % (show(items), show(items[i] if i is not None else ""), show(out)))

    @t.case("capacitor.py: base e evolução se descrevem, atacam, transformam, atacam e revertem")
    def _():
        items = []
        for fac, kind in MORPH:
            v = creature("ex1", fac, kind)
            m = get("ex1", "morph." + kind)
            items += [v["describe"], m["a1"], m["t"], m["a2"], m["r"]]
        out = run("capacitor.py")
        i = seq(out, items)
        t.expect(i is None, "esperado %s (nessa ordem), faltou %s, saída: %s"
                 % (show(items), show(items[i] if i is not None else ""), show(out)))

    # ---- ex2 ---------------------------------------------------------------

    t.group("ex2 Abstract Strategy")
    package_cases("ex2")
    abstract_cases("ex2", "BattleStrategy", ["act", "is_valid"])
    for s in STRATS:
        @t.case("%s herda de BattleStrategy e instancia sem argumentos" % s)
        def _(s=s):
            v = get("ex2", s)
            t.expect(v["sub"], "%s não herda de BattleStrategy" % s)
            t.expect(v["new"] == "ok", "%s() falhou: %s" % (s, show(v["new"])))

    def suits(s, key):
        if s == "NormalStrategy":
            return True
        return key in (MORPH if s == "AggressiveStrategy" else HEAL)

    for s in STRATS:
        @t.case("%s.is_valid() retorna um bool certo para as 8 Creature" % s)
        def _(s=s):
            bad = []
            for key in FAMILY:
                v = get("ex2", "valid.%s.%s.%s" % ((s,) + key), "%s.is_valid(%s)" % (s, FAMILY[key][0]))
                if v is not suits(s, key):
                    bad.append("%s: esperado %s, recebido %s" % (FAMILY[key][0], suits(s, key), show(v)))
            t.expect(not bad, "; ".join(bad))

    for s, cap, want in [("AggressiveStrategy", "TransformCapability", True),
                         ("DefensiveStrategy", "HealCapability", True),
                         ("AggressiveStrategy", "HealCapability", False),
                         ("DefensiveStrategy", "TransformCapability", False),
                         ("NormalStrategy", "HealCapability", True)]:
        @t.case("%s.is_valid() é %s para uma Creature nova com %s" % (s, want, cap))
        def _(s=s, cap=cap, want=want):
            v = get("ex2", "fake.%s.%s" % (s, cap), "%s.is_valid(criatura de teste)" % s)
            t.expect(v is want, "uma classe de teste (Creature + %s) deu %s, esperado %s: "
                     "is_valid deve olhar a capacidade, não o nome da classe" % (cap, show(v), want))

    def acted(s, key):
        name = FAMILY[key][0]
        r = raw("ex2", "act.%s.%s.%s" % ((s,) + key))
        ok(r, "o teste de %s.act(%s)" % (s, name))
        v = r["v"]
        e = v["err"]
        t.expect(e is None, "%s.act(%s) lançou %s: %s" % (s, name, e and e["type"], show(e and e["msg"])))
        return v, v["out"] + "\n" + v["ret"]

    def actions(log):
        return [x for x in log if x != "describe"]

    for key in [("FlameFactory", "base"), ("TransformCreatureFactory", "base"), ("HealingCreatureFactory", "base")]:
        name = FAMILY[key][0]

        @t.case("NormalStrategy.act(%s) só ataca" % name)
        def _(key=key, name=name):
            v, out = acted("NormalStrategy", key)
            a = get("ex2", "%s.%s" % key)["attack"]
            t.expect(actions(v["log"]) == ["attack"], "esperado chamar só attack(), chamou %s" % actions(v["log"]))
            t.expect(a in out, "esperado mostrar %s, saída: %s" % (show(a), show(out)))

    for key in MORPH:
        name = FAMILY[key][0]

        @t.case("AggressiveStrategy.act(%s): transform, attack, revert" % name)
        def _(key=key, name=name):
            v, out = acted("AggressiveStrategy", key)
            t.expect(actions(v["log"]) == ["transform", "attack", "revert"],
                     "esperado transform(), attack(), revert(), chamou %s" % actions(v["log"]))
            m = raw("ex2", "morph")
            if key[1] == "base" and m["ok"]:
                items = [m["v"]["t"], m["v"]["a2"], m["v"]["r"]]
                t.expect(seq(out, items) is None, "esperado mostrar %s, saída: %s" % (show(items), show(out)))

        @t.case("AggressiveStrategy.act(%s) deixa a Creature revertida" % name)
        def _(key=key, name=name):
            v, _out = acted("AggressiveStrategy", key)
            normal = get("ex2", "%s.%s" % key)["attack"]
            t.expect(v["after"] == normal, "depois de act(), attack() deu %s, esperado %s" % (show(v["after"]), show(normal)))

    for key in HEAL:
        name = FAMILY[key][0]

        @t.case("DefensiveStrategy.act(%s): attack e depois heal" % name)
        def _(key=key, name=name):
            v, out = acted("DefensiveStrategy", key)
            t.expect(actions(v["log"]) == ["attack", "heal"], "esperado attack(), heal(), chamou %s" % actions(v["log"]))
            a = get("ex2", "%s.%s" % key)["attack"]
            items = [a, heal_mark("ex2", "base", "heal") if key[1] == "base" else name]
            t.expect(seq(out, items) is None, "esperado mostrar %s, saída: %s" % (show(items), show(out)))

    BAD = [("AggressiveStrategy", ("FlameFactory", "base")), ("AggressiveStrategy", ("HealingCreatureFactory", "base")),
           ("DefensiveStrategy", ("AquaFactory", "base")), ("DefensiveStrategy", ("TransformCreatureFactory", "evolved"))]
    for s, key in BAD:
        name = FAMILY[key][0]

        @t.case("%s.act(%s) lança uma exceção própria com mensagem" % (s, name))
        def _(s=s, key=key, name=name):
            r = raw("ex2", "act.%s.%s.%s" % ((s,) + key))
            ok(r, "o teste de %s.act(%s)" % (s, name))
            e = r["v"]["err"]
            t.expect(e is not None, "combinação inválida: esperado uma exceção, mas act() terminou (saída: %s)"
                     % show(r["v"]["out"]))
            t.expect(e["type"] not in ("Exception", "AttributeError", "TypeError", "NotImplementedError"),
                     "esperado uma exceção dedicada (ex.: class StrategyError(Exception)), recebido %s: %s"
                     % (e["type"], show(e["msg"])))
            t.expect(e["msg"].strip(), "a exceção %s não tem mensagem" % e["type"])

        @t.case("%s.act(%s) recusa antes de agir" % (s, name))
        def _(s=s, key=key, name=name):
            r = raw("ex2", "act.%s.%s.%s" % ((s,) + key))
            t.expect(r["ok"] and r["v"]["err"] is not None, "combinação inválida não lançou exceção")
            t.expect(not actions(r["v"]["log"]), "chamou %s antes de lançar a exceção" % actions(r["v"]["log"]))

    authorized_cases("ex2")
    no_concrete_import("ex2")

    @t.case("tournament.py tem uma função de batalha que usa a estratégia (act)")
    def _():
        t.need("tournament.py")
        ok = [f for f in funcs("tournament.py") if "act" in func_calls(f)]
        t.expect(ok, "nenhuma função de tournament.py chama .act()")

    @t.case("python3 tournament.py roda sem erro (combinação inválida tratada)")
    def _():
        run("tournament.py")

    def strings():
        f = creature("ex2", "FlameFactory", "base")
        s = creature("ex2", "HealingCreatureFactory", "base")
        a = creature("ex2", "AquaFactory", "base")
        sh = creature("ex2", "TransformCreatureFactory", "base")
        m = get("ex2", "morph")
        return f, s, a, sh, m, heal_mark("ex2", "base", "heal")

    @t.case("torneio 0: Flameling (Normal) contra Sproutling (Defensive)")
    def _():
        f, s, _a, _sh, _m, h = strings()
        out = run("tournament.py")
        items = [f["describe"], s["describe"], f["attack"], s["attack"], h]
        i = seq(out, items)
        t.expect(i is None, "esperado %s (nessa ordem), faltou %s, saída: %s"
                 % (show(items), show(items[i] if i is not None else ""), show(out)))

    @t.case("torneio 1: Flameling + Aggressive mostra o erro e o programa continua")
    def _():
        f, s, _a, _sh, m, _h = strings()
        out = run("tournament.py")
        first = seq(out, [f["describe"], s["describe"], f["attack"]])
        t.expect(first is None, "o torneio 0 não apareceu inteiro: %s" % show(out))
        cut = out.find(f["attack"]) + len(f["attack"])
        rest = out[cut:]
        hit = re.search(r"error|erro|invalid|inválid", rest, re.I)
        t.expect(hit, "nenhuma mensagem de erro para Flameling + Aggressive, saída: %s" % show(rest))
        t.expect(f["attack"] not in rest[:hit.start()], "Flameling atacou com a estratégia agressiva antes do erro")
        t.expect(m["t"] in rest[hit.end():], "depois do erro o programa deveria seguir para o torneio 2, saída: %s" % show(rest))

    @t.case("torneio 2: Aquabub, Sproutling e Shiftling lutam todos contra todos")
    def _():
        _f, s, a, sh, m, _h = strings()
        out = run("tournament.py")
        for what, txt, n in [("Aquabub se apresentar", a["describe"], 2), ("Aquabub atacar", a["attack"], 2),
                             ("Shiftling se apresentar", sh["describe"], 2), ("Shiftling se transformar", m["t"], 2),
                             ("Shiftling atacar transformado", m["a2"], 2), ("Shiftling reverter", m["r"], 2),
                             ("Sproutling atacar", s["attack"], 3)]:
            got = out.count(txt)
            t.expect(got >= n, "esperado %s pelo menos %d vezes (%s), apareceu %d, saída: %s"
                     % (what, n, show(txt), got, show(out)))

    @t.case("torneio 2: Shiftling transforma, ataca e reverte nessa ordem")
    def _():
        _f, _s, _a, _sh, m, _h = strings()
        out = run("tournament.py")
        items = [m["t"], m["a2"], m["r"]] * 2
        t.expect(seq(out, items) is None, "esperado %s duas vezes, saída: %s" % (show(items[:3]), show(out)))
