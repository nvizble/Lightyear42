"""Python Module 09 (Cosmic Data): Pydantic v2 models, Field constraints,
model_validator, enums and nested models."""

import ast
import builtins
import re
import subprocess
import sys

from lt import common, show

EX0 = "ex0/space_station.py"
EX1 = "ex1/alien_contact.py"
EX2 = "ex2/space_crew.py"
FILES = [EX0, EX1, EX2]

# Pydantic plus any standard library module ("Standard library modules
# (json, csv, datetime, etc.) are allowed"); every builtin is allowed.
IMPORTS = {"pydantic"} | set(getattr(sys, "stdlib_module_names", ())) | {
    "datetime", "enum", "typing", "json", "csv", "sys", "os", "re", "abc",
    "collections", "dataclasses", "decimal", "math", "functools", "itertools",
    "typing_extensions", "__future__"}

NUM = r"-?\d+(?:\.\d+)?"
DROP = object()  # removes a key from the base data

# Runs in the student's process. mk(f) builds a model and reports either
# the fields it ended up with or the exception it raised.
PROBE = r'''
import contextlib, enum, io
import pydantic
with contextlib.redirect_stdout(io.StringIO()):
    import %s as m

def summ(o):
    names = getattr(type(o), "model_fields", None) or vars(o)
    d = {}
    for k in names:
        v = getattr(o, k, None)
        if isinstance(v, enum.Enum):
            val = v.value if isinstance(v.value, (str, int)) else repr(v.value)
        elif isinstance(v, list):
            val = [type(x).__name__ for x in v]
        elif isinstance(v, (str, int, float, bool, type(None))):
            val = v
        else:
            val = repr(v)
        d[k] = (type(v).__name__, val, isinstance(v, enum.Enum))
    return d

def mk(f):
    try:
        with contextlib.redirect_stdout(io.StringIO()):
            o = f()
    except pydantic.ValidationError as e:
        return {"ok": False, "ve": True, "msg": str(e)}
    except Exception as e:
        return {"ok": False, "ve": False, "type": type(e).__name__, "msg": str(e)}
    try:
        return {"ok": True, "fields": summ(o)}
    except Exception as e:
        return {"ok": True, "fields": {}, "why": repr(e)}

def meta(models, enums):
    res = {}
    for n in models:
        c = getattr(m, n, None)
        res[n] = {"exists": c is not None,
                  "base": isinstance(c, type) and issubclass(c, pydantic.BaseModel),
                  "fields": sorted(getattr(c, "model_fields", {}) or {})}
    for n in enums:
        c = getattr(m, n, None)
        res[n] = {"exists": c is not None,
                  "enum": isinstance(c, type) and issubclass(c, enum.Enum),
                  "values": sorted(str(x.value) for x in c)
                  if isinstance(c, type) and issubclass(c, enum.Enum) else None}
    return res

R = {"@meta": meta(%r, %r)}
'''


def pydantic_status(t):
    """None when pydantic 2.x imports in the python running the suite,
    otherwise the reason to skip behaviour cases."""
    code = "import pydantic, sys; sys.exit(0 if pydantic.VERSION.startswith('2') else 3)"
    try:
        p = subprocess.run([t.python, "-c", code], capture_output=True, timeout=30)
    except Exception:
        return "pydantic não está instalado (pip install pydantic)"
    if p.returncode == 3:
        return "precisa do pydantic 2.x (pip install 'pydantic>=2')"
    if p.returncode != 0:
        return "pydantic não está instalado (pip install pydantic)"
    return None


class Batch:
    """All the model probes of one exercise, run in one student process the
    first time a case asks for a result."""

    def __init__(self, t, pyd, ex, mod, models, enums):
        self.t, self.pyd, self.ex, self.mod = t, pyd, ex, mod
        self.models, self.enums = models, enums
        self.probes = {}
        self.res = None
        self.err = None

    def add(self, key, expr):
        self.probes[key] = expr

    def get(self, key):
        t = self.t
        if self.pyd:
            t.skip(self.pyd)
        if self.err is not None:
            raise self.err
        if self.res is None or key not in self.res:
            code = PROBE % (self.mod, self.models, self.enums)
            for k, expr in self.probes.items():
                if self.res is None or k not in self.res:
                    code += "R[%r] = mk(lambda: %s)\n" % (k, expr)
            code += 'print("@@" + repr(R))\n'
            try:
                r = t.py(code, self.ex, timeout=60, allow_crash=True, files=(self.mod + ".py",))
                out = [x for x in r.out.splitlines() if x.startswith("@@")]
                t.expect(out, "não consegui importar %s.py: %s" % (self.mod, show(exc_line(r))))
                self.res = dict(self.res or {}, **ast.literal_eval(out[-1][2:]))
            except Exception as e:
                self.err = e
                raise
        return self.res[key]


def exc_line(r):
    """The 'SomeError: message' line of a traceback (pydantic adds lines
    after it), else the last line."""
    found = [x.strip() for x in r.err.splitlines() if re.match(r"^[\w.]+(Error|Exception)\b", x.strip())]
    return found[-1] if found else r.last_error()


def merged(base, over):
    d = dict(base)
    for k, v in over.items():
        if v is DROP:
            d.pop(k, None)
        else:
            d[k] = v
    return d


def norm(s):
    return re.sub(r"\s+", " ", s.lower()).strip()


def value_of(out, label):
    """The text after 'label ...:' on the first line that has it."""
    for line in out.splitlines():
        mt = re.search(r"\b" + label + r"\b[^:\n]*:\s*(.*)", line, re.I)
        if mt and mt.group(1).strip():
            return mt.group(1).strip()
    return None


def first_num(s):
    mt = re.search(NUM, s or "")
    return float(mt.group()) if mt else None


ERR_HINTS = ("input should", "string should", "list should", "value error",
             "field required", "validation error", "must", "requires", "should")


def suite(t):
    common(t, FILES, typed=FILES)
    pyd = pydantic_status(t)
    ex0(t, pyd)
    ex1(t, pyd)
    ex2(t, pyd)


# ---- shared helpers ----------------------------------------------------

def decorators(t, rel):
    """(name, keyword dict) of every decorator in rel."""
    res = []
    for n in ast.walk(t.tree(rel)):
        if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef)):
            for d in n.decorator_list:
                f = d.func if isinstance(d, ast.Call) else d
                name = f.attr if isinstance(f, ast.Attribute) else getattr(f, "id", "")
                kw = {}
                if isinstance(d, ast.Call):
                    for k in d.keywords:
                        if isinstance(k.value, ast.Constant):
                            kw[k.arg] = k.value.value
                    if d.args and isinstance(d.args[0], ast.Constant):
                        kw.setdefault("mode", d.args[0].value)
                res.append((name, kw))
    return res


def static_cases(t, rel, names, validator):
    t.case("define " + ", ".join(names), lambda: (t.need(rel), t.defines(rel, *names)))
    t.case("só importa pydantic e a biblioteca padrão",
           lambda: (t.need(rel), t.authorized(rel, allowed=set(dir(builtins)), imports=IMPORTS)))

    @t.case("não usa os decorators do Pydantic v1 (@validator, @root_validator)")
    def _():
        t.need(rel)
        old = {"validator", "root_validator"}
        bad = [n for n, _ in decorators(t, rel) if n in old]
        for n in ast.walk(t.tree(rel)):
            if isinstance(n, ast.ImportFrom) and (n.module or "").startswith("pydantic"):
                bad += [a.name for a in n.names if a.name in old]
                if (n.module or "").startswith("pydantic.v1"):
                    bad.append(n.module)
        t.expect(not bad, "usa API do Pydantic v1: " + ", ".join(sorted(set(bad))))

    if validator:
        @t.case("usa @model_validator(mode='after')")
        def _():
            t.need(rel)
            ok = [kw for n, kw in decorators(t, rel) if n == "model_validator"]
            t.expect(ok, "não encontrei nenhum @model_validator")
            t.expect(any(kw.get("mode") == "after" for kw in ok),
                     "o @model_validator precisa de mode='after', veio %s" % show([kw.get("mode") for kw in ok]))
    else:
        @t.case("usa Field(...) para as restrições")
        def _():
            t.need(rel)
            calls = [n for n in ast.walk(t.tree(rel)) if isinstance(n, ast.Call)
                     and (getattr(n.func, "id", None) == "Field" or getattr(n.func, "attr", None) == "Field")]
            t.expect(calls, "não encontrei nenhuma chamada a Field(...)")


def model_meta_cases(t, b, models, fields, enums):
    for name in models:
        @t.case("%s herda de pydantic.BaseModel" % name)
        def _(name=name):
            r = b.get("@meta")[name]
            t.expect(r["exists"], "%s não existe no módulo" % name)
            t.expect(r["base"], "%s não é um BaseModel do pydantic" % name)

        @t.case("%s tem os campos do subject" % name)
        def _(name=name):
            r = b.get("@meta")[name]
            t.expect(r["base"], "%s não é um BaseModel do pydantic" % name)
            missing = [f for f in fields[name] if f not in r["fields"]]
            t.expect(not missing, "faltam os campos %s (tem %s)" % (", ".join(missing), show(r["fields"])))
    for name, values in enums.items():
        @t.case("%s é um Enum com os valores %s" % (name, ", ".join(values)))
        def _(name=name, values=values):
            r = b.get("@meta")[name]
            t.expect(r["exists"], "%s não existe no módulo" % name)
            t.expect(r["enum"], "%s não é um Enum" % name)
            t.expect(sorted(values) == r["values"], "esperava os valores %s, veio %s" % (show(sorted(values)), show(r["values"])))


def model_cases(t, b, cls, base, specs):
    """specs: (case name, overrides, check). check is "err", ("err", hints)
    or a function(fields) returning an error message or None (accepted)."""
    for i, (name, over, check) in enumerate(specs):
        key = "%s#%d" % (cls, i)
        expr = over if isinstance(over, str) else "m.%s(**%r)" % (cls, merged(base, over))
        b.add(key, expr)
    for i, (name, over, check) in enumerate(specs):
        key = "%s#%d" % (cls, i)

        @t.case(name)
        def _(key=key, check=check):
            r = b.get(key)
            if check == "err" or isinstance(check, tuple):
                t.expect(not r["ok"], "aceitou dados inválidos (esperava ValidationError)")
                t.expect(r["ve"], "lançou %s(%s) em vez de ValidationError" % (r.get("type"), show(r["msg"])))
                if isinstance(check, tuple):
                    low = r["msg"].lower()
                    t.expect(any(h in low for h in check[1]),
                             "a mensagem devia falar de %s, veio %s" % (" / ".join(check[1]), show(r["msg"])))
                return
            t.expect(r["ok"], "rejeitou dados válidos: %s" % show(r["msg"]) if not r["ok"] else "")
            msg = check(r["fields"]) if check else None
            t.expect(not msg, msg)


def accepted(**want):
    """A check: the built model has these (type name, value) per field."""
    def check(f):
        for k, (tname, val) in want.items():
            if k not in f:
                return "o modelo não tem o campo %s" % k
            gt, gv, _ = f[k]
            if tname and gt != tname:
                return "%s: esperava o tipo %s, veio %s (%s)" % (k, tname, gt, show(gv))
            if val is not None and gv != val:
                return "%s: esperava %s, veio %s" % (k, show(val), show(gv))
        return None
    return check


def enum_field(field, value):
    def check(f):
        if field not in f:
            return "o modelo não tem o campo %s" % field
        gt, gv, is_enum = f[field]
        if not is_enum and gt != "str":  # str: model_config use_enum_values=True
            return "%s devia virar um membro do Enum, veio %s %s" % (field, gt, show(gv))
        if gv != value:
            return "%s: esperava o valor %s, veio %s" % (field, show(value), show(gv))
        return None
    return check


def length_specs(field, lo, hi, prefix="", fill="x"):
    def s(n):
        return (prefix + fill * n)[:n] if n >= len(prefix) else prefix[:n]
    return [
        ("%s com %d caracteres é rejeitado (mín %d)" % (field, lo - 1, lo), {field: s(lo - 1)}, "err"),
        ("%s com %d caracteres é aceito" % (field, lo), {field: s(lo)}, None),
        ("%s com %d caracteres é aceito" % (field, hi), {field: s(hi)}, None),
        ("%s com %d caracteres é rejeitado (máx %d)" % (field, hi + 1, hi), {field: s(hi + 1)}, "err"),
    ]


def range_specs(field, lo, hi, step):
    fmt = "%g"
    return [
        (("%s=" + fmt + " é rejeitado (mín " + fmt + ")") % (field, lo - step, lo), {field: lo - step}, "err"),
        (("%s=" + fmt + " é aceito") % (field, lo), {field: lo}, None),
        (("%s=" + fmt + " é aceito") % (field, hi), {field: hi}, None),
        (("%s=" + fmt + " é rejeitado (máx " + fmt + ")") % (field, hi + step, hi), {field: hi + step}, "err"),
    ]


def missing_specs(fields):
    return [("sem %s é rejeitado (campo obrigatório)" % f, {f: DROP}, "err") for f in fields]


def program(t, b, rel):
    """Runs the program once (cached) and returns its output."""
    if b.pyd:
        t.skip(b.pyd)
    if not hasattr(b, "out"):
        r = t.run(rel, timeout=15)
        t.expect(r.code == 0, "o programa terminou com código %d: %s" % (r.code, show(r.err[-300:])))
        t.expect(r.out.strip(), "o programa não imprimiu nada")
        b.out = r.out
    return b.out


def error_shown(t, out, extra=()):
    tail = norm(out)
    t.expect(any(h in tail for h in ERR_HINTS + tuple(extra)),
             "não encontrei a mensagem do erro de validação na saída: " + show(out))
    t.expect("traceback" not in tail, "a saída tem um traceback: " + show(out))


# ---- ex0 ---------------------------------------------------------------

STATION = {
    "station_id": "ISS001", "name": "International Space Station",
    "crew_size": 6, "power_level": 85.5, "oxygen_level": 92.3,
    "last_maintenance": "2024-01-15T10:30:00",
}


def ex0(t, pyd):
    t.group("ex0 space_station")
    static_cases(t, EX0, ["SpaceStation", "main"], validator=False)
    b = Batch(t, pyd, "ex0", "space_station", ["SpaceStation"], [])
    S = "SpaceStation"
    specs = [
        ("dados do subject (ISS001) são aceitos com os tipos certos", {},
         accepted(station_id=("str", "ISS001"), crew_size=("int", 6), power_level=("float", 85.5),
                  oxygen_level=("float", 92.3), last_maintenance=("datetime", None))),
        ("is_operational vale True por padrão", {}, accepted(is_operational=("bool", True))),
        ("notes vale None por padrão", {}, accepted(notes=("NoneType", None))),
        ("is_operational=False e notes são aceitos", {"is_operational": False, "notes": "Solar panel repair"},
         accepted(is_operational=("bool", False), notes=("str", "Solar panel repair"))),
        ("last_maintenance em string ISO vira datetime", {"last_maintenance": "2023-12-01T08:00:00"},
         accepted(last_maintenance=("datetime", None))),
    ]
    specs += length_specs("station_id", 3, 10, "ISS")
    specs += length_specs("name", 1, 50, "Station")
    specs += range_specs("crew_size", 1, 20, 1)
    specs += range_specs("power_level", 0.0, 100.0, 0.1)
    specs += range_specs("oxygen_level", 0.0, 100.0, 0.1)
    specs += [
        ("notes com 200 caracteres é aceito", {"notes": "n" * 200}, None),
        ("notes com 201 caracteres é rejeitado (máx 200)", {"notes": "n" * 201}, "err"),
        ("notes=None explícito é aceito", {"notes": None}, None),
        ("crew_size='six' é rejeitado (não é inteiro)", {"crew_size": "six"}, "err"),
        ("crew_size=6.5 é rejeitado (não é inteiro)", {"crew_size": 6.5}, "err"),
        ("power_level='high' é rejeitado (não é número)", {"power_level": "high"}, "err"),
        ("oxygen_level=None é rejeitado", {"oxygen_level": None}, "err"),
        ("last_maintenance='yesterday' é rejeitado (não é data)", {"last_maintenance": "yesterday"}, "err"),
        ("is_operational='maybe' é rejeitado (não é booleano)", {"is_operational": "maybe"}, "err"),
        ("crew_size=21 dá a mensagem do pydantic (less than or equal to 20)", {"crew_size": 21},
         ("err", ("less than or equal to 20",))),
    ]
    specs += missing_specs(["station_id", "name", "crew_size", "power_level", "oxygen_level"])
    model_meta_cases(t, b, [S], {S: list(STATION) + ["is_operational", "notes"]}, {})
    model_cases(t, b, S, STATION, specs)

    @t.case("python3 %s roda sem erro" % EX0)
    def _():
        program(t, b, EX0)

    @t.case("execução: mostra a estação válida (ID, nome, tripulação, energia, oxigênio, status)")
    def _():
        out = program(t, b, EX0)
        for label in ("id", "name", "crew", "power", "oxygen", "status"):
            t.expect(value_of(out, label), "não encontrei a linha '%s: ...' na saída: %s" % (label.capitalize(), show(out)))
        crew = first_num(value_of(out, "crew"))
        t.expect(crew is not None and 1 <= crew <= 20, "Crew: esperava um número entre 1 e 20, veio %s" % show(value_of(out, "crew")))
        for label in ("power", "oxygen"):
            v = first_num(value_of(out, label))
            t.expect(v is not None and 0 <= v <= 100, "%s: esperava 0..100, veio %s" % (label, show(value_of(out, label))))
        st = norm(value_of(out, "status"))
        t.expect("operational" in st or "maintenance" in st or "offline" in st,
                 "Status: esperava 'Operational' (ou equivalente), veio %s" % show(st))

    @t.case("execução: mostra o erro de validação da estação inválida")
    def _():
        out = program(t, b, EX0)
        error_shown(t, out)


# ---- ex1 ---------------------------------------------------------------

CONTACT = {
    "contact_id": "AC_2024_001", "timestamp": "2024-01-15T22:30:00",
    "location": "Area 51, Nevada", "contact_type": "radio",
    "signal_strength": 8.5, "duration_minutes": 45, "witness_count": 5,
    "message_received": "Greetings from Zeta Reticuli",
}
CONTACT_TYPES = ["radio", "visual", "physical", "telepathic"]


def ex1(t, pyd):
    t.group("ex1 alien_contact")
    static_cases(t, EX1, ["ContactType", "AlienContact", "main"], validator=True)
    b = Batch(t, pyd, "ex1", "alien_contact", ["AlienContact"], ["ContactType"])
    A = "AlienContact"
    weak = {"signal_strength": 3.0, "message_received": None}
    specs = [
        ("dados do subject (AC_2024_001) são aceitos com os tipos certos", {},
         accepted(contact_id=("str", "AC_2024_001"), timestamp=("datetime", None), signal_strength=("float", 8.5),
                  duration_minutes=("int", 45), witness_count=("int", 5))),
        ("contact_type='radio' é aceito como ContactType", {}, enum_field("contact_type", "radio")),
        ("is_verified vale False por padrão", {}, accepted(is_verified=("bool", False))),
        ("message_received vale None por padrão (sinal fraco)", dict(weak, message_received=DROP),
         accepted(message_received=("NoneType", None))),
    ]
    for ct in CONTACT_TYPES:
        extra = {"is_verified": True} if ct == "physical" else {}
        specs.append(("contact_type=%s é aceito" % show(ct), dict({"contact_type": ct}, **extra), enum_field("contact_type", ct)))
    specs += [
        ("contact_type='laser' é rejeitado (fora do Enum)", {"contact_type": "laser"}, "err"),
        ("contact_id com 4 caracteres é rejeitado (mín 5)", {"contact_id": "AC12"}, "err"),
        ("contact_id com 5 caracteres é aceito", {"contact_id": "AC123"}, None),
        ("contact_id com 15 caracteres é aceito", {"contact_id": "AC" + "1" * 13}, None),
        ("contact_id com 16 caracteres é rejeitado (máx 15)", {"contact_id": "AC" + "1" * 14}, "err"),
    ]
    specs += length_specs("location", 3, 100, "Area")
    specs += range_specs("signal_strength", 0.0, 10.0, 0.1)
    specs += range_specs("duration_minutes", 1, 1440, 1)
    specs += range_specs("witness_count", 1, 100, 1)
    specs += [
        ("message_received com 500 caracteres é aceito", {"message_received": "m" * 500}, None),
        ("message_received com 501 caracteres é rejeitado (máx 500)", {"message_received": "m" * 501}, "err"),
        ("timestamp='soon' é rejeitado (não é data)", {"timestamp": "soon"}, "err"),
        ("witness_count='five' é rejeitado (não é inteiro)", {"witness_count": "five"}, "err"),
        ("signal_strength='loud' é rejeitado (não é número)", {"signal_strength": "loud"}, "err"),
        ("duration_minutes=45.5 é rejeitado (não é inteiro)", {"duration_minutes": 45.5}, "err"),
        # business rules
        ("regra: contact_id sem 'AC' no início é rejeitado (XC_2024_001)", {"contact_id": "XC_2024_001"}, "err"),
        ("regra: contact_id sem 'AC' no início é rejeitado (ID_AC_0001)", {"contact_id": "ID_AC_0001"}, "err"),
        ("regra: contato physical não verificado é rejeitado", {"contact_type": "physical"}, "err"),
        ("regra: contato physical verificado é aceito", {"contact_type": "physical", "is_verified": True}, None),
        ("regra: contato visual não verificado é aceito", {"contact_type": "visual"}, None),
        ("regra: telepathic com 2 testemunhas é rejeitado (mín 3)",
         {"contact_type": "telepathic", "witness_count": 2}, ("err", ("witness", "3"))),
        ("regra: telepathic com 1 testemunha é rejeitado",
         {"contact_type": "telepathic", "witness_count": 1}, "err"),
        ("regra: telepathic com 3 testemunhas é aceito", {"contact_type": "telepathic", "witness_count": 3}, None),
        ("regra: radio com 1 testemunha é aceito (o mínimo de 3 é só telepathic)", {"witness_count": 1}, None),
        ("regra: sinal 8.5 sem mensagem é rejeitado", {"message_received": None}, "err"),
        ("regra: sinal 7.1 sem mensagem é rejeitado", {"signal_strength": 7.1, "message_received": DROP}, "err"),
        ("regra: sinal 10.0 sem mensagem é rejeitado", {"signal_strength": 10.0, "message_received": None}, "err"),
        ("regra: sinal 7.0 sem mensagem é aceito (só > 7.0 exige)", {"signal_strength": 7.0, "message_received": None}, None),
        ("regra: sinal 3.0 sem mensagem é aceito", weak, None),
    ]
    specs += missing_specs(["contact_id", "location", "contact_type", "signal_strength", "duration_minutes", "witness_count"])
    model_meta_cases(t, b, [A], {A: list(CONTACT) + ["is_verified"]}, {"ContactType": CONTACT_TYPES})
    model_cases(t, b, A, CONTACT, specs)

    @t.case("python3 %s roda sem erro" % EX1)
    def _():
        program(t, b, EX1)

    @t.case("execução: mostra o contato válido (ID, tipo, local, sinal, duração, testemunhas)")
    def _():
        out = program(t, b, EX1)
        for label in ("id", "type", "location", "signal", "duration", "witness(?:es)?"):
            t.expect(value_of(out, label), "não encontrei a linha '%s: ...' na saída: %s"
                     % (label.split("(")[0].capitalize(), show(out)))
        cid = value_of(out, "id")
        t.expect(cid.startswith("AC"), "ID: o contato válido devia começar com 'AC', veio %s" % show(cid))
        ty = norm(value_of(out, "type"))
        t.expect("contacttype." not in ty, "Type: mostra o membro do Enum (%s) em vez do valor (ex.: 'radio')" % show(ty))
        t.expect(any(c in ty for c in CONTACT_TYPES), "Type: esperava radio/visual/physical/telepathic, veio %s" % show(ty))
        sig = first_num(value_of(out, "signal"))
        t.expect(sig is not None and 0 <= sig <= 10, "Signal: esperava 0..10, veio %s" % show(value_of(out, "signal")))

    @t.case("execução: mostra o erro de um contato inválido")
    def _():
        out = program(t, b, EX1)
        error_shown(t, out)


# ---- ex2 ---------------------------------------------------------------

MEMBER = {
    "member_id": "CM001", "name": "Sarah Connor", "rank": "commander", "age": 45,
    "specialization": "Mission Command", "years_experience": 20,
}


def cm(**kw):
    return merged(MEMBER, kw)


CREW = [
    cm(),
    cm(member_id="CM002", name="John Smith", rank="lieutenant", age=35, specialization="Navigation", years_experience=8),
    cm(member_id="CM003", name="Alice Johnson", rank="officer", age=29, specialization="Engineering", years_experience=3),
]
MISSION = {
    "mission_id": "M2024_MARS", "mission_name": "Mars Colony Establishment",
    "destination": "Mars", "launch_date": "2024-09-01T00:00:00",
    "duration_days": 900, "crew": CREW, "budget_millions": 2500.0,
}
RANKS = ["cadet", "officer", "lieutenant", "captain", "commander"]


def ex2(t, pyd):
    t.group("ex2 space_crew")
    static_cases(t, EX2, ["Rank", "CrewMember", "SpaceMission", "main"], validator=True)
    b = Batch(t, pyd, "ex2", "space_crew", ["CrewMember", "SpaceMission"], ["Rank"])
    C, M = "CrewMember", "SpaceMission"
    model_meta_cases(t, b, [C, M], {C: list(MEMBER) + ["is_active"], M: list(MISSION) + ["mission_status"]},
                     {"Rank": RANKS})

    member_specs = [
        ("CrewMember: dados válidos são aceitos com os tipos certos", {},
         accepted(member_id=("str", "CM001"), age=("int", 45), years_experience=("int", 20))),
        ("CrewMember: rank='commander' é aceito como Rank", {}, enum_field("rank", "commander")),
        ("CrewMember: is_active vale True por padrão", {}, accepted(is_active=("bool", True))),
        ("CrewMember: rank='admiral' é rejeitado (fora do Enum)", {"rank": "admiral"}, "err"),
        ("CrewMember: age='old' é rejeitado (não é inteiro)", {"age": "old"}, "err"),
    ]
    for r in RANKS:
        member_specs.append(("CrewMember: rank=%s é aceito" % show(r), {"rank": r}, enum_field("rank", r)))
    for name, over, check in (length_specs("member_id", 3, 10, "CM") + length_specs("name", 2, 50, "Sa")
                              + range_specs("age", 18, 80, 1) + length_specs("specialization", 3, 30, "Nav")
                              + range_specs("years_experience", 0, 50, 1)
                              + missing_specs(["member_id", "name", "rank", "age", "specialization"])):
        member_specs.append(("CrewMember: " + name, over, check))
    model_cases(t, b, C, MEMBER, member_specs)

    short = {"duration_days": 30}
    mission_specs = [
        ("SpaceMission: missão do subject (M2024_MARS) é aceita com os tipos certos", {},
         accepted(mission_id=("str", "M2024_MARS"), launch_date=("datetime", None), duration_days=("int", 900),
                  budget_millions=("float", 2500.0), crew=("list", ["CrewMember"] * 3))),
        ("SpaceMission: mission_status vale 'planned' por padrão", {}, accepted(mission_status=("str", "planned"))),
        ("SpaceMission: mission_status='active' é aceito", {"mission_status": "active"},
         accepted(mission_status=("str", "active"))),
        ("SpaceMission: aceita a tripulação como objetos CrewMember",
         "m.SpaceMission(**dict(%r, crew=[m.CrewMember(**c) for c in %r]))" % (MISSION, CREW),
         accepted(crew=("list", ["CrewMember"] * 3))),
        ("SpaceMission: tripulação vazia é rejeitada (mín 1)", {"crew": []}, "err"),
        ("SpaceMission: tripulação de 1 comandante é aceita", dict(short, crew=[cm()]), None),
        ("SpaceMission: tripulação de 12 é aceita", dict(short, crew=[cm()] * 12), None),
        ("SpaceMission: tripulação de 13 é rejeitada (máx 12)", dict(short, crew=[cm()] * 13), "err"),
        ("SpaceMission: membro inválido dentro da missão (age 17) é rejeitado",
         {"crew": [cm(), cm(member_id="CM009", age=17)]}, "err"),
        ("SpaceMission: membro com rank inválido dentro da missão é rejeitado",
         {"crew": [cm(), cm(member_id="CM009", rank="admiral")]}, "err"),
        ("SpaceMission: crew que não é lista é rejeitado", {"crew": "Sarah Connor"}, "err"),
        ("SpaceMission: mission_id com 4 caracteres é rejeitado (mín 5)", {"mission_id": "M202"}, "err"),
        ("SpaceMission: mission_id com 5 caracteres é aceito", {"mission_id": "M2024"}, None),
        ("SpaceMission: mission_id com 15 caracteres é aceito", {"mission_id": "M" + "1" * 14}, None),
        ("SpaceMission: mission_id com 16 caracteres é rejeitado (máx 15)", {"mission_id": "M" + "1" * 15}, "err"),
    ]
    for name, over, check in (length_specs("mission_name", 3, 100, "Mars") + length_specs("destination", 3, 50, "Mars")
                              + range_specs("budget_millions", 1.0, 10000.0, 0.1)):
        mission_specs.append(("SpaceMission: " + name, over, check))
    mission_specs += [
        ("SpaceMission: duration_days=0 é rejeitado (mín 1)", {"duration_days": 0}, "err"),
        ("SpaceMission: duration_days=1 é aceito", {"duration_days": 1}, None),
        ("SpaceMission: duration_days=3650 é aceito", {"duration_days": 3650}, None),
        ("SpaceMission: duration_days=3651 é rejeitado (máx 3650)", {"duration_days": 3651}, "err"),
        ("SpaceMission: launch_date='tomorrow' é rejeitado (não é data)", {"launch_date": "tomorrow"}, "err"),
        ("SpaceMission: budget_millions='lots' é rejeitado (não é número)", {"budget_millions": "lots"}, "err"),
    ]
    mission_specs += [("SpaceMission: " + n, o, c) for n, o, c in
                      missing_specs(["mission_id", "mission_name", "destination", "duration_days", "crew", "budget_millions"])]
    lt_, of_ = CREW[1], CREW[2]
    mission_specs += [
        ("regra: mission_id sem 'M' no início é rejeitado (X2024_MARS)", {"mission_id": "X2024_MARS"}, "err"),
        ("regra: mission_id sem 'M' no início é rejeitado (2024_M)", {"mission_id": "2024_M"}, "err"),
        ("regra: sem Commander nem Captain é rejeitado", {"crew": [lt_, of_]}, ("err", ("commander", "captain"))),
        ("regra: tripulação só de cadetes é rejeitada", dict(short, crew=[cm(rank="cadet")] * 2), "err"),
        ("regra: com Captain (sem Commander) é aceito", {"crew": [cm(rank="captain"), lt_, of_]}, None),
        ("regra: missão longa (900 dias) com 1/3 experiente é rejeitada",
         {"crew": [cm(years_experience=2), lt_, of_]}, "err"),
        ("regra: missão de 366 dias sem ninguém experiente é rejeitada",
         {"duration_days": 366, "crew": [cm(years_experience=1), cm(rank="cadet", years_experience=0)]}, "err"),
        ("regra: missão de 365 dias sem ninguém experiente é aceita (só > 365 exige)",
         {"duration_days": 365, "crew": [cm(years_experience=1), cm(rank="cadet", years_experience=0)]}, None),
        ("regra: missão longa com exatamente 50% experiente é aceita",
         {"duration_days": 400, "crew": [cm(years_experience=10), cm(rank="cadet", years_experience=0)]}, None),
        ("regra: missão longa: 5 anos de experiência já conta como experiente",
         {"duration_days": 400, "crew": [cm(years_experience=5)]}, None),
        ("regra: missão longa: 4 anos de experiência não conta",
         {"duration_days": 400, "crew": [cm(years_experience=4)]}, "err"),
        ("regra: membro inativo na tripulação é rejeitado", dict(short, crew=[cm(), cm(rank="cadet", is_active=False)]), "err"),
        ("regra: comandante inativo é rejeitado", dict(short, crew=[cm(is_active=False), cm(rank="captain")]), "err"),
    ]
    model_cases(t, b, M, MISSION, mission_specs)

    @t.case("python3 %s roda sem erro" % EX2)
    def _():
        program(t, b, EX2)

    @t.case("execução: mostra a missão válida (nome, ID, destino, duração, orçamento, tripulação)")
    def _():
        out = program(t, b, EX2)
        for label in ("mission", "id", "destination", "duration", "budget", "crew size"):
            t.expect(value_of(out, label), "não encontrei a linha '%s: ...' na saída: %s" % (label.capitalize(), show(out)))
        mid = value_of(out, "id")
        t.expect(mid.startswith("M"), "ID: a missão válida devia começar com 'M', veio %s" % show(mid))

    @t.case("execução: lista os tripulantes como '- Nome (rank) - especialidade'")
    def _():
        out = program(t, b, EX2)
        t.expect("rank." not in norm(out), "mostra o membro do Enum (Rank.X) em vez do valor (ex.: 'commander'): " + show(out))
        rows = [x for x in out.splitlines() if re.search(r"\((?:%s)\)" % "|".join(RANKS), x, re.I)]
        size = first_num(value_of(out, "crew size"))
        t.expect(rows, "não encontrei linhas '- Nome (rank) - especialidade': " + show(out))
        t.expect(size is None or len(rows) >= size, "Crew size diz %s, mas só %d tripulantes listados" % (size, len(rows)))
        t.expect(any(re.search(r"\((commander|captain)\)", x, re.I) for x in rows),
                 "a missão válida devia ter um commander ou captain na lista: " + show(out))

    @t.case("execução: mostra o erro de uma missão inválida")
    def _():
        out = program(t, b, EX2)
        error_shown(t, out)
