"""Python Module 03 (Data Quest): lists, tuples, sets, dicts, generators,
comprehensions."""

import ast
import builtins
import re
from collections import Counter

from lt import common, show

EX0 = "ex0/ft_command_quest.py"
EX1 = "ex1/ft_score_analytics.py"
EX2 = "ex2/ft_coordinate_system.py"
EX3 = "ex3/ft_achievement_tracker.py"
EX4 = "ex4/ft_inventory_system.py"
EX5 = "ex5/ft_data_stream.py"
EX6 = "ex6/ft_data_alchemist.py"
FILES = [EX0, EX1, EX2, EX3, EX4, EX5, EX6]

# str/int/float are allowed everywhere; raising/catching exceptions too.
EXC = {n for n in dir(builtins) if isinstance(getattr(builtins, n), type)
       and issubclass(getattr(builtins, n), BaseException)}
BASE = {"str", "int", "float"} | EXC

NUM = r"-?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?"

# Loads only the definitions of a student file (imports, functions, classes,
# constant assignments) so its main code doesn't run.
LOADER = r'''
import ast, types
def _load(path):
    tree = ast.parse(open(path, encoding="utf-8").read())
    funcs = {n.name for n in tree.body
             if isinstance(n, (ast.FunctionDef, ast.ClassDef))}
    def risky(n):
        for x in ast.walk(n):
            if (isinstance(x, ast.Call) and isinstance(x.func, ast.Name)
                    and (x.func.id in funcs
                         or x.func.id in ("input", "print", "next"))):
                return True
        return False
    mod = types.ModuleType("student")
    mod.__file__ = path
    for n in tree.body:
        if (isinstance(n, (ast.Import, ast.ImportFrom, ast.FunctionDef,
                           ast.ClassDef))
                or (isinstance(n, (ast.Assign, ast.AnnAssign))
                    and not risky(n))):
            try:
                exec(compile(ast.Module(body=[n], type_ignores=[]), path,
                             "exec"), mod.__dict__)
            except Exception:
                pass
    return mod
'''


def norm(s):
    return re.sub(r"\s+", " ", s.lower()).replace(" :", ":").strip()


def lines(out):
    return [norm(x) for x in out.splitlines() if x.strip()]


def has_line(out, want):
    return norm(want) in lines(out)


def last_num(line):
    found = re.findall(NUM, line)
    return float(found[-1]) if found else None


def num(out, *keys):
    """Last number on the first line that has every key (lowercase)."""
    for x in lines(out):
        if all(k in x for k in keys):
            return last_num(x)
    return None


def close(a, b, tol):
    return a is not None and abs(a - b) <= tol


def literal(line):
    """The first Python literal ([..], {..}, (..), set()) on a line."""
    s = line.rstrip()
    if s.endswith("set()"):
        return set()
    for i, c in enumerate(s):
        if c in "[{(":
            for cand in (s[i:], s[i:].rstrip(" .!")):
                try:
                    return ast.literal_eval(cand)
                except (ValueError, SyntaxError, TypeError, MemoryError, RecursionError):
                    pass
    return None


def literals(out):
    res = []
    for x in out.splitlines():
        v = literal(x)
        if v is not None:
            res.append((x, v))
    return res


def word(name, line):
    return re.search(r"(?<![\w])" + re.escape(name.lower()) + r"(?![\w])", line.lower()) is not None


def snippet(t, ex, rel, body, stdin=""):
    """Runs body with the student's definitions loaded as m; body prints OK
    or KO <message>."""
    code = LOADER + "m = _load(%r)\n" % rel.split("/")[-1] + body
    r = t.py(code, ex, stdin=stdin, files=(rel.split("/")[-1],))
    out = [x for x in r.out.strip().splitlines() if x.strip()]
    last = out[-1].strip() if out else ""
    if last.startswith("KO "):
        t.expect(False, last[3:])
    t.expect(last == "OK", "saída inesperada do teste: " + show(r.out[-300:]))


def has_node(t, rel, kind):
    return any(isinstance(n, kind) for n in ast.walk(t.tree(rel)))


def suite(t):
    common(t, FILES, typed=FILES)
    ex0(t)
    ex1(t)
    ex2(t)
    ex3(t)
    ex4(t)
    ex5(t)
    ex6(t)


# ---- ex0 ---------------------------------------------------------------

def ex0(t):
    t.group("ex0 ft_command_quest")
    t.case("só usa o permitido (sys, len, print)",
           lambda: t.authorized(EX0, BASE | {"len", "print", "list"}, {"sys"}))

    def args_ok(args):
        out = t.run(EX0, args=args).out
        n = len(args)
        t.expect(has_line(out, "Arguments received: %d" % n),
                 "esperava 'Arguments received: %d', veio %s" % (n, show(out)))
        for i, a in enumerate(args, 1):
            want = "Argument %d: %s" % (i, a)
            t.expect(has_line(out, want), "esperava %s, veio %s" % (show(want), show(out)))
        t.expect(has_line(out, "Total arguments: %d" % (n + 1)),
                 "esperava 'Total arguments: %d', veio %s" % (n + 1, show(out)))
        got = [x for x in lines(out) if re.match(r"argument \d+:", x)]
        t.expect(len(got) == n, "esperava %d linhas 'Argument N:', veio %d (o nome do programa conta como argumento?)"
                 % (n, len(got)))
        t.expect("no arguments" not in norm(out), "diz 'No arguments provided!' mesmo recebendo argumentos")
        return out

    @t.case("sem argumentos")
    def _():
        out = t.run(EX0).out
        t.expect("no arguments provided" in norm(out), "esperava 'No arguments provided!', veio " + show(out))
        t.expect(has_line(out, "Total arguments: 1"), "esperava 'Total arguments: 1', veio " + show(out))
        t.expect("arguments received" not in norm(out), "mostra 'Arguments received' sem argumentos")

    @t.case("mostra o nome do programa")
    def _():
        out = t.run(EX0).out
        ok = [x for x in lines(out) if x.startswith("program name:") and x.endswith("ft_command_quest.py")]
        t.expect(ok, "esperava 'Program name: ft_command_quest.py', veio " + show(out))

    t.case("exemplo: hello world 42", lambda: args_ok(["hello", "world", "42"]))
    t.case("exemplo: \"Data Quest\" (um argumento com espaço)", lambda: args_ok(["Data Quest"]))
    t.case("um argumento", lambda: args_ok(["solo"]))
    t.case("15 argumentos", lambda: args_ok([str(i * 7) for i in range(1, 16)]))
    t.case("argumentos estranhos (-n, *, a:b, número negativo)", lambda: args_ok(["-n", "*", "a:b", "-42"]))

    @t.case("argumento vazio (\"\") conta")
    def _():
        out = t.run(EX0, args=["", "x"]).out
        t.expect(has_line(out, "Arguments received: 2"), "esperava 'Arguments received: 2', veio " + show(out))
        t.expect(has_line(out, "Total arguments: 3"), "esperava 'Total arguments: 3', veio " + show(out))


# ---- ex1 ---------------------------------------------------------------

def ex1(t):
    t.group("ex1 ft_score_analytics")
    t.case("só usa o permitido (sys, len, sum, max, min, print)",
           lambda: t.authorized(EX1, BASE | {"len", "sum", "max", "min", "print", "list"}, {"sys"}))

    @t.case("usa try/except para validar")
    def _():
        t.expect(has_node(t, EX1, ast.Try), "não encontrei nenhum try/except")

    def stats(args, scores):
        out = t.run(EX1, args=args).out
        lists = [v for _, v in literals(out) if isinstance(v, list)]
        t.expect(scores in lists, "esperava a lista %s, veio %s" % (scores, show(out)))
        want = [("total players", ("total", "player"), len(scores)),
                ("total score", ("total", "score"), sum(scores)),
                ("average", ("average",), sum(scores) / len(scores)),
                ("high score", ("high",), max(scores)),
                ("low score", ("low",), min(scores)),
                ("score range", ("range",), max(scores) - min(scores))]
        for label, keys, exp in want:
            got = num(out, *keys)
            t.expect(close(got, exp, 0.01), "%s: esperava %s, veio %s" % (label, exp, got))
        return out

    def no_scores(out):
        o = norm(out)
        t.expect("no scores" in o or "usage" in o, "esperava 'No scores provided. Usage: ...', veio " + show(out))
        t.expect(num(out, "total", "score") is None, "mostra estatísticas sem nenhum score válido: " + show(out))

    t.case("exemplo: 1500 2300 1800 2100 1950",
           lambda: stats(["1500", "2300", "1800", "2100", "1950"], [1500, 2300, 1800, 2100, 1950]))

    t.case("sem argumentos", lambda: no_scores(t.run(EX1).out))

    @t.case("exemplo: ab ac (todos inválidos)")
    def _():
        out = t.run(EX1, args=["ab", "ac"]).out
        for bad in ("ab", "ac"):
            t.expect(any(word(bad, x) for x in out.splitlines()), "não avisou que %s é inválido: %s" % (show(bad), show(out)))
        no_scores(out)

    @t.case("válidos e inválidos misturados: 10 abc 20")
    def _():
        out = stats(["10", "abc", "20"], [10, 20])
        t.expect(any(word("abc", x) for x in out.splitlines()), "não avisou que 'abc' é inválido: " + show(out))

    t.case("um score só: 42", lambda: stats(["42"], [42]))
    t.case("repetidos: 100 100 100", lambda: stats(["100", "100", "100"], [100, 100, 100]))
    t.case("média não inteira: 1 2", lambda: stats(["1", "2"], [1, 2]))
    t.case("zero: 0 0", lambda: stats(["0", "0"], [0, 0]))
    t.case("número negativo não quebra: -5 10", lambda: t.run(EX1, args=["-5", "10"]))
    t.case("decimal / vazio não quebram: 3.5 '' 7", lambda: t.run(EX1, args=["3.5", "", "7"]))


# ---- ex2 ---------------------------------------------------------------

def ex2(t):
    t.group("ex2 ft_coordinate_system")
    t.case("só usa o permitido (math, input, round, print)",
           lambda: t.authorized(EX2, BASE | {"input", "round", "print", "list", "tuple"}, {"math"}))
    t.case("define get_player_pos()", lambda: t.defines(EX2, "get_player_pos"))

    def flow(stdin, first, center, between):
        out = t.run(EX2, stdin=stdin).out
        tuples = [v for _, v in literals(out) if isinstance(v, tuple) and len(v) == 3]
        t.expect(first in tuples, "esperava a tupla %s, veio %s" % (first, show(out)))
        flat = norm(out).replace(" ", "")
        for axis, v in zip("xyz", first):
            t.expect(("%s=%s" % (axis, v)) in flat or ("%s:%s" % (axis, v)) in flat, "esperava '%s=%s' em 'It includes', veio %s"
                     % (axis.upper(), v, show(out)))
        got = num(out, "center")
        t.expect(close(got, center, 0.006), "distância ao centro: esperava %.4f, veio %s" % (center, got))
        got = num(out, "between")
        t.expect(close(got, between, 0.006), "distância entre as duas: esperava %.4f, veio %s" % (between, got))
        return out

    @t.case("exemplo do subject (hello world / 1.0 , 2.5, 3.0 / 4,abc,5 / 4,5,6)")
    def _():
        out = flow("hello world\n1.0 , 2.5, 3.0\n4,abc,5\n4,5,6\n", (1.0, 2.5, 3.0), 4.0311, 4.9244)
        t.expect(any(word("abc", x) for x in out.splitlines()), "não avisou do erro em 'abc': " + show(out))

    t.case("inteiros: 3,4,0 e 0,0,0", lambda: flow("3,4,0\n0,0,0\n", (3.0, 4.0, 0.0), 5.0, 5.0))
    t.case("negativos: -1,-2,-2 e 1,2,2", lambda: flow("-1,-2,-2\n1,2,2\n", (-1.0, -2.0, -2.0), 3.0, 6.0))
    t.case("origem: 0,0,0 duas vezes", lambda: flow("0,0,0\n0,0,0\n", (0.0, 0.0, 0.0), 0.0, 0.0))
    t.case("repete com 2 valores, 4 valores e linha vazia",
           lambda: flow("1,2\n1,2,3,4\n\n1,2,2\n,,\n1,2,2\n", (1.0, 2.0, 2.0), 3.0, 0.0))
    t.case("repete com separador errado e texto",
           lambda: flow("1;2;3\n1 2 3\nx,y,z\n2,3,6\n2,3,6\n", (2.0, 3.0, 6.0), 7.0, 0.0))

    @t.case("get_player_pos() devolve tupla de floats")
    def _():
        snippet(t, "ex2", EX2, r'''
p = m.get_player_pos()
print()
if not isinstance(p, tuple):
    print("KO devolveu %s, esperava tuple" % type(p).__name__)
elif p != (1.0, 2.0, 3.0) or not all(isinstance(v, float) for v in p):
    print("KO esperava (1.0, 2.0, 3.0), veio %r" % (p,))
else:
    print("OK")
''', stdin="abc\n1, 2 ,3\n")

    @t.case("get_player_pos() repete até vir um valor válido")
    def _():
        snippet(t, "ex2", EX2, r'''
p = m.get_player_pos()
print()
print("OK" if p == (7.0, 8.0, 9.0) else "KO esperava (7.0, 8.0, 9.0), veio %r" % (p,))
''', stdin="x,y,z\n1;2;3\n1,2\n7,8,9\n")


# ---- ex3 ---------------------------------------------------------------

def parse_ex3(t, out):
    players, only, missing = {}, {}, {}
    union = common_ = None
    rows = []
    for line, v in literals(out):
        if v == {}:
            v = set()
        if not isinstance(v, (set, frozenset)):
            continue
        label = line[:line.find("set()") if line.rstrip().endswith("set()") else min(
            i for i in (line.find("{"), line.find("set(")) if i >= 0)].strip().rstrip(":").strip()
        rows.append((label, set(v)))
    for label, v in rows:
        low = label.lower()
        if low.startswith("player"):
            players[label[6:].strip()] = v
    t.expect(len(players) >= 4, "esperava pelo menos 4 linhas 'Player <nome>: {...}', veio %d: %s"
             % (len(players), show(out)))

    def who(low):
        for p in players:
            if word(p, low):
                return p
        return None

    for label, v in rows:
        low = label.lower()
        if low.startswith("player"):
            continue
        if "missing" in low and who(low):
            missing[who(low)] = v
        elif ("only" in low or "exclusive" in low) and who(low):
            only[who(low)] = v
        elif "common" in low or "shared" in low:
            common_ = v
        elif "distinct" in low or "unique" in low or "all" in low:
            union = v
    return players, union, common_, only, missing


def ex3(t):
    t.group("ex3 ft_achievement_tracker")
    t.case("só usa o permitido (len, print, random, set)",
           lambda: t.authorized(EX3, BASE | {"len", "print", "set", "list", "tuple"}, {"random"}))
    t.case("define gen_player_achievements()", lambda: t.defines(EX3, "gen_player_achievements"))

    @t.case("gen_player_achievements() devolve set, aleatório e de tamanho variável")
    def _():
        snippet(t, "ex3", EX3, r'''
res = [m.gen_player_achievements() for _ in range(60)]
bad = [r for r in res if not isinstance(r, set)]
if bad:
    print("KO devolveu %s, esperava set" % type(bad[0]).__name__)
elif len({frozenset(r) for r in res}) < 2:
    print("KO sempre devolve o mesmo set: %r" % (res[0],))
elif len({len(r) for r in res}) < 2:
    print("KO a quantidade de conquistas é sempre %d (devia ser aleatória)" % len(res[0]))
elif not any(res):
    print("KO só devolve sets vazios")
else:
    print("OK")
''')

    runs = []

    def parsed():
        if not runs:
            for _ in range(3):
                runs.append(parse_ex3(t, t.run(EX3).out))
        return runs

    @t.case("execução: pelo menos 4 jogadores")
    def _():
        parsed()

    @t.case("execução: todas as conquistas distintas = união")
    def _():
        for players, union, *_ in parsed():
            exp = set().union(*players.values())
            t.expect(union is not None, "não encontrei a linha 'All distinct achievements: {...}'")
            t.expect(union == exp, "esperava %s, veio %s" % (show(exp), show(union)))

    @t.case("execução: conquistas comuns = interseção")
    def _():
        for players, _, common_, *_ in parsed():
            exp = set.intersection(*players.values())
            t.expect(common_ is not None, "não encontrei a linha 'Common achievements: {...}'")
            t.expect(common_ == exp, "esperava %s, veio %s" % (show(exp), show(common_)))

    @t.case("execução: 'Only X has' de cada jogador")
    def _():
        for players, _, _, only, _ in parsed():
            for p, s in players.items():
                others = set().union(*[v for q, v in players.items() if q != p])
                t.expect(p in only, "não encontrei a linha 'Only %s has: ...'" % p)
                t.expect(only[p] == s - others, "Only %s has: esperava %s, veio %s" % (p, show(s - others), show(only[p])))

    @t.case("execução: 'X is missing' de cada jogador")
    def _():
        for players, _, _, _, missing in parsed():
            every = set().union(*players.values())
            for p, s in players.items():
                t.expect(p in missing, "não encontrei a linha '%s is missing: ...'" % p)
                t.expect(not (missing[p] & s), "%s is missing inclui conquistas que ele tem: %s" % (p, show(missing[p] & s)))
                t.expect(every - s <= missing[p], "%s is missing: faltam %s" % (p, show((every - s) - missing[p])))

    @t.case("execução: os sets mudam entre execuções")
    def _():
        t.expect(len({str(sorted((k, sorted(v)) for k, v in r[0].items())) for r in parsed()}) > 1,
                 "três execuções deram exatamente os mesmos jogadores/conquistas")


# ---- ex4 ---------------------------------------------------------------

def ex4(t):
    t.group("ex4 ft_inventory_system")
    t.case("só usa o permitido (sys, len, print, sum, list, round, dict.*)",
           lambda: t.authorized(EX4, BASE | {"len", "print", "sum", "list", "round", "dict", "tuple", "set"}, {"sys"}))

    cache = {}

    def run(args):
        key = tuple(args)
        if key not in cache:
            cache[key] = t.run(EX4, args=args).out
        return cache[key]

    def dicts(out):
        return [(x, v) for x, v in literals(out) if isinstance(v, dict)]

    EXAMPLE = ["sword:1", "potion:5", "shield:2", "armor:3", "helmet:1", "sword:2", "hello", "key:value"]
    INV = {"sword": 1, "potion": 5, "shield": 2, "armor": 3, "helmet": 1}

    def first_dict(out, exp):
        ds = dicts(out)
        t.expect(ds, "não encontrei o inventário {...} na saída: " + show(out))
        got = ds[0][1]
        t.expect(got == exp and list(got) == list(exp), "inventário: esperava %s, veio %s" % (exp, got))
        return ds

    t.case("exemplo: inventário", lambda: first_dict(run(EXAMPLE), INV))

    @t.case("exemplo: quantidades guardadas como int")
    def _():
        ds = dicts(run(EXAMPLE))
        t.expect(ds and all(type(v) is int for v in ds[0][1].values()), "valores não são int: " + show(ds[0][0] if ds else ""))

    @t.case("exemplo: avisa os parâmetros descartados (sword repetido, hello, key:value)")
    def _():
        out = run(EXAMPLE)
        msgs = [x for x in out.splitlines() if "{" not in x and "[" not in x]
        for bad in ("sword", "hello", "key"):
            t.expect(any(word(bad, x) for x in msgs), "nenhuma mensagem de erro sobre %s: %s" % (show(bad), show(out)))

    @t.case("exemplo: lista de itens")
    def _():
        out = run(EXAMPLE)
        ls = [v for _, v in literals(out) if isinstance(v, list)]
        t.expect(list(INV) in ls, "esperava %s, veio %s" % (list(INV), show(out)))

    @t.case("exemplo: quantidade total 12")
    def _():
        got = num(run(EXAMPLE), "total")
        t.expect(close(got, 12, 0), "esperava 12, veio %s" % got)

    @t.case("exemplo: porcentagens")
    def _():
        out = run(EXAMPLE)
        for name, q in INV.items():
            pl = [x for x in out.splitlines() if word(name, x) and "%" in x]
            t.expect(pl, "não encontrei a porcentagem de %s: %s" % (name, show(out)))
            got = float(re.findall(r"(" + NUM + r")\s*%", pl[0])[-1]) if re.findall(r"(" + NUM + r")\s*%", pl[0]) else None
            exp = q * 100 / 12
            t.expect(close(got, exp, 0.051), "%s: esperava %.1f%%, veio %s" % (name, exp, show(pl[0])))

    def extremes(out, most, mq, least, lq, others=()):
        ml = [x for x in lines(out) if "most" in x]
        ll = [x for x in lines(out) if "least" in x]
        t.expect(ml and word(most, ml[0]) and close(last_num(ml[0]), mq, 0),
                 "mais abundante: esperava %s (%d), veio %s" % (most, mq, show(ml[0] if ml else out)))
        t.expect(ll and word(least, ll[0]) and close(last_num(ll[0]), lq, 0),
                 "menos abundante: esperava %s (%d), veio %s" % (least, lq, show(ll[0] if ll else out)))

    t.case("exemplo: mais e menos abundante (empate fica com o primeiro)",
           lambda: extremes(run(EXAMPLE), "potion", 5, "sword", 1))

    @t.case("exemplo: inventário atualizado com um item novo")
    def _():
        ds = dicts(run(EXAMPLE))
        t.expect(len(ds) >= 2, "não encontrei o inventário atualizado")
        last = ds[-1][1]
        t.expect(len(last) == 6 and all(last.get(k) == v for k, v in INV.items()),
                 "esperava o inventário original + 1 item, veio %s" % last)

    @t.case("empate no mais abundante: gem:2 orb:5 rune:5 axe:2")
    def _():
        out = run(["gem:2", "orb:5", "rune:5", "axe:2"])
        extremes(out, "orb", 5, "gem", 2)

    @t.case("repetido fica com o primeiro valor: x:3 x:9")
    def _():
        first_dict(run(["x:3", "x:9"]), {"x": 3})

    @t.case("um item só: gem:4 (100%)")
    def _():
        out = run(["gem:4"])
        first_dict(out, {"gem": 4})
        t.expect(any(re.search(r"100(\.0+)?\s*%", x) for x in out.splitlines()), "esperava 100%, veio " + show(out))
        extremes(out, "gem", 4, "gem", 4)

    @t.case("sem argumentos (inventário vazio)")
    def _():
        out = run([])
        ds = dicts(out)
        t.expect(ds and ds[0][1] == {}, "esperava o inventário vazio {}, veio " + show(out))
        t.expect(len(ds[-1][1]) == 1, "esperava o inventário atualizado com 1 item, veio " + show(ds[-1][0]))

    t.case("só inválidos não quebra: hello a:b :5 a:1:2", lambda: run(["hello", "a:b", ":5", "a:1:2"]))
    t.case("quantidades zero não quebram (divisão por zero): a:0 b:0", lambda: run(["a:0", "b:0"]))
    t.case("quantidade negativa não quebra: a:-3 b:5", lambda: run(["a:-3", "b:5"]))


# ---- ex5 ---------------------------------------------------------------

def ex5(t):
    t.group("ex5 ft_data_stream")
    t.case("só usa o permitido (next, range, len, print, typing, random)",
           lambda: t.authorized(EX5, BASE | {"next", "range", "len", "print", "list", "tuple", "set", "dict"},
                                {"typing", "random"}))
    t.case("define gen_event() e consume_event()", lambda: t.defines(EX5, "gen_event", "consume_event"))

    @t.case("gen_event() é um gerador (yield)")
    def _():
        snippet(t, "ex5", EX5, r'''
import inspect, types
if not inspect.isgeneratorfunction(m.gen_event):
    print("KO gen_event não é uma função geradora (sem yield?)")
elif not isinstance(m.gen_event(), types.GeneratorType):
    print("KO gen_event() não devolve um gerador")
else:
    print("OK")
''')

    @t.case("gen_event() é infinito e dá tuplas (nome, ação) aleatórias")
    def _():
        snippet(t, "ex5", EX5, r'''
g = m.gen_event()
ev = []
try:
    for _ in range(5000):
        ev.append(next(g))
except StopIteration:
    print("KO o gerador acabou depois de %d eventos (devia ser infinito)" % len(ev))
    raise SystemExit
bad = [e for e in ev if not (isinstance(e, tuple) and len(e) == 2 and all(isinstance(x, str) for x in e))]
print()
if bad:
    print("KO esperava tupla (nome, ação) de strings, veio %r" % (bad[0],))
elif len({e[0] for e in ev}) < 2 or len({e[1] for e in ev}) < 2:
    print("KO os eventos não variam (nomes/ações sempre iguais)")
else:
    print("OK")
''', )

    @t.case("consume_event() é um gerador (yield)")
    def _():
        snippet(t, "ex5", EX5, r'''
import inspect
print("OK" if inspect.isgeneratorfunction(m.consume_event)
      else "KO consume_event não é uma função geradora (sem yield?)")
''')

    @t.case("consume_event() tira um evento da lista a cada next() até esvaziar")
    def _():
        snippet(t, "ex5", EX5, r'''
from collections import Counter
orig = [("alice", "run"), ("bob", "eat"), ("alice", "run"), ("carl", "swim"), ("dan", "use"),
        ("bob", "eat"), ("eve", "move"), ("fay", "grab"), ("gus", "sleep"), ("hal", "climb")]
lst = list(orig)
g = m.consume_event(lst)
first = next(g)
print()
if len(lst) != 9:
    print("KO depois do 1º next() a lista devia ter 9 eventos, tem %d" % len(lst))
    raise SystemExit
got = [first] + list(g)
print()
if Counter(got) != Counter(orig):
    print("KO esperava receber os 10 eventos da lista, veio %r" % (got,))
elif lst:
    print("KO a lista devia terminar vazia, sobrou %r" % (lst,))
else:
    print("OK")
''')

    @t.case("consume_event() com lista vazia não dá nada")
    def _():
        snippet(t, "ex5", EX5, r'''
lst = []
got = list(m.consume_event(lst))
print()
print("OK" if got == [] else "KO esperava nada, veio %r" % (got,))
''')

    @t.case("consume_event() escolhe aleatoriamente")
    def _():
        snippet(t, "ex5", EX5, r'''
orders = set()
for _ in range(30):
    lst = [("p%d" % i, "a%d" % i) for i in range(10)]
    orders.add(tuple(m.consume_event(lst)))
print()
print("OK" if len(orders) > 1 else "KO 30 rodadas deram sempre a mesma ordem (não é aleatório)")
''')

    @t.case("consume_event() é usado direto num for .. in")
    def _():
        ok = any(isinstance(n, ast.For) and isinstance(n.iter, ast.Call) and isinstance(n.iter.func, ast.Name)
                 and n.iter.func.id == "consume_event" for n in ast.walk(t.tree(EX5)))
        t.expect(ok, "não encontrei 'for ... in consume_event(...)'")

    cache = []

    def run():
        if not cache:
            cache.append(t.run(EX5, timeout=10).out)
        return cache[0]

    @t.case("execução: mostra 1000 eventos")
    def _():
        out = run()
        n = len([x for x in lines(out) if re.search(r"event\s*#?\d+", x)])
        t.expect(n == 1000, "esperava 1000 linhas 'Event N: ...', veio %d" % n)

    @t.case("execução: lista de 10 eventos consumida até ficar vazia")
    def _():
        out = run()
        lits = literals(out)
        start = next((i for i, (_, v) in enumerate(lits) if isinstance(v, list) and len(v) == 10), None)
        t.expect(start is not None, "não encontrei a lista de 10 eventos: " + show(out[-500:]))
        built = lits[start][1]
        t.expect(all(isinstance(e, tuple) and len(e) == 2 for e in built), "a lista devia ter tuplas (nome, ação): " + show(built))
        got = [v for _, v in lits[start + 1:] if isinstance(v, tuple)]
        rem = [v for _, v in lits[start + 1:] if isinstance(v, list)]
        t.expect(len(got) == 10, "esperava 10 'Got event from list', veio %d" % len(got))
        t.expect(Counter(got) == Counter(built), "os eventos consumidos não batem com a lista: %s" % show(got))
        if rem:
            left = Counter(built)
            for g, r in zip(got, rem):
                left -= Counter([g])
                t.expect(Counter(r) == left, "depois de tirar %s a lista devia ser %s, veio %s"
                         % (g, show(list(left.elements())), show(r)))
            t.expect(rem[-1] == [], "a lista devia terminar vazia, veio " + show(rem[-1]))


# ---- ex6 ---------------------------------------------------------------

def ex6(t):
    t.group("ex6 ft_data_alchemist")
    t.case("só usa o permitido (random, print, len, sum, round)",
           lambda: t.authorized(EX6, BASE | {"print", "len", "sum", "round", "list", "dict", "set", "tuple"}, {"random"}))

    @t.case("usa list comprehensions (2) e dict comprehensions (2)")
    def _():
        nodes = list(ast.walk(t.tree(EX6)))
        lc = sum(isinstance(n, ast.ListComp) for n in nodes)
        dc = sum(isinstance(n, ast.DictComp) for n in nodes)
        t.expect(lc >= 2, "esperava 2 list comprehensions, achei %d" % lc)
        t.expect(dc >= 2, "esperava 2 dict comprehensions, achei %d" % dc)

    cache = []

    def runs():
        if not cache:
            for _ in range(3):
                out = t.run(EX6).out
                lits = literals(out)
                ls = [v for _, v in lits if isinstance(v, list)]
                ds = [v for _, v in lits if isinstance(v, dict)]
                t.expect(len(ls) >= 3, "esperava 3 listas (inicial, capitalizada, só capitalizados), veio " + show(out))
                t.expect(len(ds) >= 2, "esperava 2 dicts (scores, high scores), veio " + show(out))
                cache.append((out, ls, ds))
        return cache

    def cap(n):
        return {n.capitalize(), n.title(), n[:1].upper() + n[1:]}

    @t.case("lista inicial mistura nomes capitalizados e não")
    def _():
        init = runs()[0][1][0]
        t.expect(all(isinstance(n, str) for n in init), "lista inicial devia ter strings: " + show(init))
        t.expect(any(n[:1].isupper() for n in init) and any(n[:1].islower() for n in init),
                 "a lista inicial devia ter nomes capitalizados e não capitalizados: " + show(init))

    @t.case("lista com todos os nomes capitalizados")
    def _():
        init, allcap = runs()[0][1][:2]
        t.expect(len(allcap) == len(init) and all(a in cap(n) for n, a in zip(init, allcap)),
                 "esperava %s, veio %s" % ([n.capitalize() for n in init], allcap))

    @t.case("lista só com os nomes já capitalizados")
    def _():
        init, _, only = runs()[0][1][:3]
        a = [n for n in init if n == n.capitalize()]
        b = [n for n in init if n[:1].isupper()]
        t.expect(only in (a, b), "esperava %s, veio %s" % (b, only))

    @t.case("dict de scores: um score numérico por nome capitalizado")
    def _():
        for _, ls, ds in runs():
            scores = ds[0]
            t.expect(list(scores) == list(dict.fromkeys(ls[1])), "chaves esperadas %s, veio %s" % (ls[1], list(scores)))
            t.expect(all(type(v) in (int, float) for v in scores.values()), "scores devem ser números: " + show(scores))

    @t.case("média dos scores")
    def _():
        for out, _, ds in runs():
            s = ds[0]
            exp = sum(s.values()) / len(s)
            got = num(out, "average")
            t.expect(close(got, exp, 0.01), "esperava %.2f, veio %s" % (exp, got))

    @t.case("high scores = scores acima da média")
    def _():
        for _, _, ds in runs():
            s = ds[0]
            avg = sum(s.values()) / len(s)
            exp = {k: v for k, v in s.items() if v > avg}
            t.expect(ds[1] == exp, "esperava %s, veio %s" % (exp, ds[1]))

    @t.case("scores aleatórios mudam entre execuções")
    def _():
        t.expect(len({str(r[2][0]) for r in runs()}) > 1, "três execuções deram os mesmos scores")
