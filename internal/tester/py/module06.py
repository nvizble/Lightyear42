"""Python Module 06 (The Codex): imports, packages, circular imports."""

import ast
import builtins
import re

from lt import common, show

A4 = "ft_alembic_4.py"   # the subject wants a mypy error here, on purpose
ALEMBIC = ["ft_alembic_%d.py" % i for i in range(6)]
DISTILLATION = ["ft_distillation_0.py", "ft_distillation_1.py"]
TRANSMUTATION = ["ft_transmutation_%d.py" % i for i in range(3)]
KABOOM = ["ft_kaboom_0.py", "ft_kaboom_1.py"]
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
FILES = PACKAGE + ["elements.py"] + ALEMBIC + DISTILLATION + KABOOM + TRANSMUTATION

# Only the project's own modules may be imported (typing is tolerated for
# annotations). lt.authorized ignores the dots of a relative import, so the
# submodule names are listed too.
MODULES = {"alchemy", "elements", "potions", "transmutation", "recipes", "grimoire",
           "light_spellbook", "light_validator", "dark_spellbook", "dark_validator", "typing"}
BUILTINS = set(dir(builtins)) - {"eval", "exec"}

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


# ---- imports (ast) -----------------------------------------------------


def imports(t, rel, top_only=False):
    """[(kind, module, names)] for every import in rel. kind is 'import' or
    'from'; a relative module keeps its dots ('.elements', '..potions').
    top_only skips the imports inside functions and classes."""
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


def need_import(t, rel, ok, want):
    t.need(rel)
    imps = imports(t, rel)
    t.expect(ok(imps), "esperado `%s`, achei: %s" % (want, code(imps)))
    return imps


def only_import_alchemy(t, rel):
    """`import alchemy` and nothing deeper (no alchemy.x, no from alchemy)."""
    imps = need_import(t, rel, lambda i: has(i, "import", "alchemy"), "import alchemy")
    deeper = [i for i in imps if i[1].startswith("alchemy.") or (i[0] == "from" and i[1] == "alchemy")]
    t.expect(not deeper, "o subject pede só `import alchemy`, mas também tem %s" % code(deeper))


def uses_attr(t, rel, obj, attr):
    """obj.attr appears in rel (also through `import obj as x`)."""
    tree = t.tree(rel)
    names = {obj} | {a.asname for n in ast.walk(tree) if isinstance(n, ast.Import)
                     for a in n.names if a.name == obj and a.asname}
    return any(isinstance(n, ast.Attribute) and n.attr == attr and isinstance(n.value, ast.Name)
               and n.value.id in names for n in ast.walk(tree))


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


def no_hardcode(t, rel, *words):
    t.need(rel)
    src = low(t.source(rel))
    found = [w for w in words if w in src]
    t.expect(not found, "%s escreve %s direto no código: deveria chamar as funções importadas"
             % (rel, show(found)))


def static(t, rels):
    for rel in rels:
        t.case("só importa módulos do projeto, sem eval/exec: " + rel,
               lambda rel=rel: (t.need(rel), t.authorized(rel, allowed=BUILTINS, imports=MODULES)))


# ---- Part I: the alembic -----------------------------------------------


def alembic(t):
    t.group("alembic")

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

    a0, a1, a2, a3, a4, a5 = ALEMBIC

    @t.case("ft_alembic_0 usa `import elements` (sem from)")
    def _():
        imps = need_import(t, a0, lambda i: has(i, "import", "elements"), "import elements")
        t.expect(not has(imps, "from", "elements"), "o ex 0 é com `import ...`, não `from ... import`")

    t.case("ft_alembic_0 cria o fogo", lambda: run_shows(t, a0, FIRE))

    @t.case("ft_alembic_1 usa `from elements import create_water`")
    def _():
        imps = need_import(t, a1, lambda i: has(i, "from", "elements", "create_water"),
                           "from elements import create_water")
        t.expect(not has(imps, "import", "elements"), "o ex 1 é com `from ... import`, não `import elements`")

    t.case("ft_alembic_1 cria a água", lambda: run_shows(t, a1, WATER))
    t.case("ft_alembic_2 usa `import alchemy.elements`",
           lambda: need_import(t, a2, lambda i: has(i, "import", "alchemy.elements"), "import alchemy.elements"))
    t.case("ft_alembic_2 cria a terra", lambda: run_shows(t, a2, EARTH))
    t.case("ft_alembic_3 usa `from alchemy.elements import ...`",
           lambda: need_import(t, a3, lambda i: has(i, "from", "alchemy.elements")
                               or has(i, "from", "alchemy", "elements"), "from alchemy.elements import create_air"))
    t.case("ft_alembic_3 cria o ar", lambda: run_shows(t, a3, AIR))
    t.case("ft_alembic_4 usa só `import alchemy`", lambda: only_import_alchemy(t, a4))

    @t.case("ft_alembic_4 tenta chamar alchemy.create_earth()")
    def _():
        t.need(a4)
        t.expect(uses_attr(t, a4, "alchemy", "create_earth"),
                 "esperado uma chamada a alchemy.create_earth() para mostrar que ela não é exposta")

    @t.case("ft_alembic_4 cria o ar e create_earth dá AttributeError")
    def _():
        r = run_shows(t, a4, AIR, allow_crash=True)
        both = r.out + r.err
        t.expect("create_earth" in both and ("AttributeError" in both or "has no attribute" in both),
                 "esperado o AttributeError de create_earth (no stderr ou capturado), veio %s / %s"
                 % (show(r.out), show(r.err[-300:])))
        t.expect(norm(EARTH) not in norm(r.out), "create_earth() não deveria funcionar via alchemy: %s"
                 % show(r.out))

    t.case("ft_alembic_5 usa `from alchemy import create_air`",
           lambda: need_import(t, a5, lambda i: has(i, "from", "alchemy", "create_air"),
                               "from alchemy import create_air"))
    t.case("ft_alembic_5 cria o ar", lambda: run_shows(t, a5, AIR))
    static(t, ["elements.py", ELEM] + ALEMBIC)


# ---- the alchemy package ------------------------------------------------


def package(t):
    t.group("pacote alchemy")

    attrs = ("with contextlib.redirect_stdout(io.StringIO()):\n"
             "    import alchemy\n"
             "done(sorted(n for n in dir(alchemy) if not n.startswith('__')))\n")

    def names():
        return value(t, attrs, files=(INIT,))

    @t.case("`import alchemy` funciona")
    def _():
        names()

    @t.case("alchemy expõe create_air")
    def _():
        got = names()
        t.expect("create_air" in got, "esperado alchemy.create_air, alchemy tem %s" % show(got))
        same(t, call(t, "alchemy", "create_air"), AIR, "alchemy.create_air()")

    @t.case("alchemy NÃO expõe create_earth")
    def _():
        got = names()
        t.expect("create_earth" not in got, "o subject quer create_earth escondida de `import alchemy`, "
                 "mas alchemy tem %s" % show(got))

    @t.case("`from alchemy import create_earth` dá ImportError")
    def _():
        t.need(INIT)
        r = t.py("from alchemy import create_earth", ex=".", allow_crash=True)
        t.expect("ImportError" in r.err, "esperado ImportError, veio %s" % show(r.err[-200:] or r.out))

    @t.case("alchemy expõe strength_potion")
    def _():
        same(t, call(t, "alchemy", "strength_potion"), STRENGTH, "alchemy.strength_potion()")

    @t.case("alchemy.heal é o alias de healing_potion")
    def _():
        got = names()
        t.expect("heal" in got, "esperado alchemy.heal (alias do healing potion), alchemy tem %s" % show(got))
        same(t, call(t, "alchemy", "heal"), HEAL, "alchemy.heal()")

    @t.case("`import alchemy.transmutation` chega em lead_to_gold")
    def _():
        t.need(TINIT)
        body = ("with contextlib.redirect_stdout(io.StringIO()):\n"
                "    import alchemy.transmutation as m\n"
                "    f = getattr(m, 'lead_to_gold', None) or getattr(getattr(m, 'recipes', None), 'lead_to_gold', None)\n"
                "    r = f() if f else None\n"
                "done(r)\n")
        got = value(t, body)
        t.expect(got is not None, "alchemy/transmutation/__init__.py deveria expor lead_to_gold "
                 "(ou recipes) para o ft_transmutation_1")
        same(t, got, GOLD, "lead_to_gold()")

    @t.case("só `import alchemy` já chega em transmutation")
    def _():
        body = ("with contextlib.redirect_stdout(io.StringIO()):\n"
                "    import alchemy as a\n"
                "    tr = getattr(a, 'transmutation', None)\n"
                "    f = (getattr(a, 'lead_to_gold', None) or getattr(tr, 'lead_to_gold', None)\n"
                "         or getattr(getattr(tr, 'recipes', None), 'lead_to_gold', None))\n"
                "    r = f() if f else None\n"
                "done(r)\n")
        got = value(t, body, files=(INIT,))
        t.expect(got is not None, "alchemy/__init__.py deveria importar transmutation para o ft_transmutation_2")
        same(t, got, GOLD, "lead_to_gold() via alchemy")

    @t.case("`import alchemy` não carrega o grimório sombrio (não explode)")
    def _():
        body = ("import sys\nwith contextlib.redirect_stdout(io.StringIO()):\n"
                "    import alchemy\n"
                "done([m for m in sys.modules if m.startswith('alchemy.grimoire.dark')])\n")
        got = value(t, body, files=(INIT,))
        t.expect(not got, "`import alchemy` já importa %s" % show(got))

    static(t, [INIT, TINIT, GINIT])


# ---- Part II: distillation ----------------------------------------------


def distillation(t):
    t.group("distillation")
    d0, d1 = DISTILLATION

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

    t.case("ft_distillation_0 usa `from alchemy.potions import ...`",
           lambda: need_import(t, d0, lambda i: has(i, "from", "alchemy.potions")
                               or has(i, "from", "alchemy", "potions"),
                               "from alchemy.potions import healing_potion, strength_potion"))
    t.case("ft_distillation_0 prepara as duas poções", lambda: run_shows(t, d0, STRENGTH, HEAL))
    t.case("ft_distillation_1 usa só `import alchemy`", lambda: only_import_alchemy(t, d1))

    @t.case("ft_distillation_1 usa alchemy.heal e alchemy.strength_potion")
    def _():
        t.need(d1)
        for attr in ("heal", "strength_potion"):
            t.expect(uses_attr(t, d1, "alchemy", attr), "esperado alchemy.%s() no ft_distillation_1" % attr)

    t.case("ft_distillation_1 prepara as duas poções", lambda: run_shows(t, d1, STRENGTH, HEAL))
    static(t, [POTIONS] + DISTILLATION)


# ---- Part III: transmutation --------------------------------------------


def transmutation(t):
    t.group("transmutation")
    t0, t1, t2 = TRANSMUTATION

    t.case("recipes.py define lead_to_gold", lambda: (t.need(RECIPES), t.defines(RECIPES, "lead_to_gold")))
    t.case("lead_to_gold() retorna a receita completa",
           lambda: same(t, call(t, "alchemy.transmutation.recipes", "lead_to_gold"), GOLD, "lead_to_gold()"))
    t.case("recipes.py chama as funções (não copia as strings)",
           lambda: no_hardcode(t, RECIPES, "element created", "potion brewed"))

    @t.case("recipes.py tem pelo menos um import relativo")
    def _():
        need_import(t, RECIPES, lambda i: any(m.startswith(".") for k, m, ns in i),
                           "from ..potions import strength_potion (com ponto)")

    @t.case("recipes.py tem pelo menos um import absoluto")
    def _():
        need_import(t, RECIPES, lambda i: any(not m.startswith(".") and m.split(".")[0] in ("alchemy", "elements")
                                              for k, m, ns in i),
                    "from alchemy.elements import create_air (sem ponto)")

    t.case("ft_transmutation_0 usa `import alchemy.transmutation.recipes`",
           lambda: need_import(t, t0, lambda i: has(i, "import", "alchemy.transmutation.recipes"),
                               "import alchemy.transmutation.recipes"))
    t.case("ft_transmutation_0 transforma chumbo em ouro", lambda: run_shows(t, t0, GOLD))

    @t.case("ft_transmutation_1 usa `import alchemy.transmutation`")
    def _():
        imps = need_import(t, t1, lambda i: has(i, "import", "alchemy.transmutation"), "import alchemy.transmutation")
        other = [i for i in imps if i[0] == "from" or i[1] == "alchemy.transmutation.recipes"]
        t.expect(not other, "o ex 1 importa o módulo transmutation, não recipes: tem %s" % code(other))

    t.case("ft_transmutation_1 transforma chumbo em ouro", lambda: run_shows(t, t1, GOLD))
    t.case("ft_transmutation_2 usa só `import alchemy`", lambda: only_import_alchemy(t, t2))
    t.case("ft_transmutation_2 transforma chumbo em ouro", lambda: run_shows(t, t2, GOLD))
    static(t, [RECIPES] + TRANSMUTATION)


# ---- Part IV: kaboom ----------------------------------------------------


def mentions(imps, name):
    return any(name in m or name in ns for k, m, ns in imps)


def kaboom(t):
    t.group("kaboom")
    k0, k1 = KABOOM

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
        t.expect(any("validat" in n for n in top), "esperado uma função validate_... , achei %s" % show(top))

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

    for ing in ("Earth, wind and fire", "AIR", "Water"):
        @t.case("validate_ingredients(%s) é VALID (sem diferenciar maiúsculas)" % show(ing))
        def _(ing=ing):
            got = validate(ing)
            t.expect(isinstance(got, str) and low(ing) in low(got), "esperado os ingredientes %s no retorno, veio %s"
                     % (show(ing), show(got)))
            t.expect("VALID" in got and "INVALID" not in got, "esperado VALID, veio %s" % show(got))

    for ing in ("bats, frogs", "salt"):
        @t.case("validate_ingredients(%s) é INVALID" % show(ing))
        def _(ing=ing):
            got = validate(ing)
            t.expect(isinstance(got, str) and low(ing) in low(got), "esperado os ingredientes %s no retorno, veio %s"
                     % (show(ing), show(got)))
            t.expect("INVALID" in got, "esperado INVALID, veio %s" % show(got))

    def record(name, ing):
        return call(t, "alchemy.grimoire.light_spellbook", "light_spell_record", name, ing)

    @t.case("light_spell_record com ingrediente da luz: gravado")
    def _():
        got = record("Fantasy", "Earth, wind and fire")
        t.expect(isinstance(got, str), "esperado uma str, veio %s" % show(got))
        t.expect("Fantasy" in got, "esperado o nome do feitiço no retorno, veio %s" % show(got))
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

    t.case("ft_kaboom_0 acessa o módulo grimoire",
           lambda: need_import(t, k0, lambda i: has(i, "import", "alchemy.grimoire")
                               or has(i, "from", "alchemy.grimoire") or has(i, "from", "alchemy", "grimoire"),
                               "import alchemy.grimoire (ou from alchemy.grimoire import ...)"))

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
        t.expect("spell recorded" not in low(r.out), "o feitiço das trevas não deveria ser gravado: %s"
                 % show(r.out))

    static(t, [LIGHT_BOOK, LIGHT_VAL, DARK_BOOK, DARK_VAL] + KABOOM)


def suite(t):
    # ft_alembic_4 has a mypy error on purpose: checked apart, without mypy.
    rest = [f for f in FILES if f != A4]
    common(t, rest, typed=FILES)
    t.group("entrega")
    t.case(A4, lambda: t.need(A4))

    @t.case("flake8 em " + A4)
    def _():
        t.need(A4)
        r = t.tool("flake8", [A4])
        t.expect(r.code == 0, r.out.strip() or r.err.strip())

    t.case("sintaxe de " + A4, lambda: (t.need(A4), t.tree(A4)))
    alembic(t)
    package(t)
    distillation(t)
    transmutation(t)
    kaboom(t)
