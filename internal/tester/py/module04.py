"""Python Module 04 (Data Archivist): file I/O, streams, context managers."""

import ast
import os

from lt import common, show

EX0 = "ex0/ft_ancient_text.py"
EX1 = "ex1/ft_archive_creation.py"
EX2 = "ex2/ft_stream_management.py"
EX3 = "ex3/ft_vault_security.py"
FILES = [EX0, EX1, EX2, EX3]

LINES = [
    "[FRAGMENT 001] Digital preservation protocols established 2087",
    "[FRAGMENT 002] Knowledge must survive the entropy wars",
    "[FRAGMENT 003] Every byte saved is a victory against oblivion",
]
FRAGMENT = "\n".join(LINES) + "\n"
DATA = "lt_fragment.txt"   # every file the suite creates starts with lt_
OUT = "lt_saved.txt"
TYPES = {"str", "int", "float", "list", "dict", "set", "tuple"}
ROOT = hasattr(os, "geteuid") and os.geteuid() == 0


class Files:
    """Creates files in an exercise folder and removes them (and anything
    else the run created there) on exit, so the project is left as it was."""

    def __init__(self, t, ex, files=None, mode=None):
        self.t, self.dir, self.files, self.mode = t, t.path(ex), files or {}, mode

    def __enter__(self):
        self.before = set(os.listdir(self.dir)) if os.path.isdir(self.dir) else set()
        for name, content in self.files.items():
            with open(os.path.join(self.dir, name), "w") as f:
                f.write(content)
            if self.mode is not None:
                os.chmod(os.path.join(self.dir, name), self.mode)
        return self

    def read(self, name):
        p = os.path.join(self.dir, name)
        if not os.path.isfile(p):
            return None
        with open(p, errors="replace") as f:
            return f.read()

    def __exit__(self, *exc):
        if not os.path.isdir(self.dir):
            return False
        for name in set(os.listdir(self.dir)) - self.before | set(self.files):
            p = os.path.join(self.dir, name)
            if os.path.isfile(p):
                os.chmod(p, 0o644)
                os.remove(p)
        return False


def low(s):
    return s.lower()


def missing_msg(s):
    s = low(s)
    return any(k in s for k in ("no such file", "errno 2", "not found", "does not exist", "not exist"))


def denied_msg(s):
    s = low(s)
    return "permission denied" in s or "errno 13" in s


def no_root(t):
    if ROOT:
        t.skip("rodando como root: chmod não bloqueia a leitura")


def static_no_with(t, rel):
    tree = t.tree(rel)
    t.expect(not any(isinstance(n, (ast.With, ast.AsyncWith)) for n in ast.walk(tree)),
             "o subject proíbe `with` antes do ex3 (use open() + close())")


def static_close(t, rel):
    tree = t.tree(rel)
    t.expect(any(isinstance(n, ast.Call) and isinstance(n.func, ast.Attribute) and n.func.attr == "close"
                 for n in ast.walk(tree)),
             "nenhuma chamada a .close(): o arquivo aberto nunca é fechado")


def check_shown(t, out, lines):
    for line in lines:
        t.expect(line in out, "o conteúdo do arquivo não aparece na saída: esperado %s em %s"
                 % (show(line), show(out)))
    pos = [out.find(line) for line in lines]
    t.expect(pos == sorted(pos), "as linhas aparecem fora de ordem: %s" % show(out))


# ---- ex0 ---------------------------------------------------------------


def ex0(t):
    t.group("ex0 ft_ancient_text")
    ex = "ex0"

    t.case("só usa o que é permitido", lambda: t.authorized(
        EX0, allowed={"len", "open", "print"} | TYPES, imports={"sys", "typing"}))
    t.case("não usa `with` (só no ex3)", lambda: static_no_with(t, EX0))
    t.case("fecha o arquivo com .close()", lambda: static_close(t, EX0))

    @t.case("sem argumento: mostra o Usage")
    def _():
        r = t.run(EX0)
        t.expect("usage" in low(r.out + r.err), "esperado uma mensagem 'Usage: ...', veio %s" % show(r.out + r.err))

    @t.case("lê o arquivo do exemplo como o cat")
    def _():
        with Files(t, ex, {DATA: FRAGMENT}):
            r = t.run(EX0, args=(DATA,))
        t.expect(FRAGMENT.rstrip("\n") in r.out,
                 "esperado o conteúdo exato do arquivo %s na saída, veio %s" % (show(FRAGMENT), show(r.out)))
        check_shown(t, r.out, LINES)

    @t.case("estrutura: cabeçalho, '---' em volta do conteúdo, mensagem de fechado")
    def _():
        with Files(t, ex, {DATA: FRAGMENT}):
            r = t.run(EX0, args=(DATA,))
        t.expect(DATA in r.out, "esperado o nome do arquivo (%s) na saída: %s" % (show(DATA), show(r.out)))
        t.expect(r.out.count("---") >= 2, "esperado '---' antes e depois do conteúdo: %s" % show(r.out))
        t.expect("closed" in low(r.out), "esperado a mensagem 'File ... closed.' no fim: %s" % show(r.out))
        t.expect(r.out.rfind("closed") > r.out.find(LINES[-1]), "'closed' deveria vir depois do conteúdo")

    @t.case("arquivo de várias linhas sem \\n no final")
    def _():
        with Files(t, ex, {DATA: "alpha\nbeta\ngamma"}):
            r = t.run(EX0, args=(DATA,))
        check_shown(t, r.out, ["alpha", "beta", "gamma"])

    @t.case("arquivo vazio: não quebra")
    def _():
        with Files(t, ex, {DATA: ""}):
            r = t.run(EX0, args=(DATA,))
        t.expect("closed" in low(r.out), "esperado 'File ... closed.' mesmo com arquivo vazio: %s" % show(r.out))

    @t.case("arquivo inexistente: mensagem de erro, sem traceback")
    def _():
        r = t.run(EX0, args=("lt_nao_existe.txt",))
        both = r.out + r.err
        t.expect(missing_msg(both), "esperado algo como \"No such file or directory\", veio %s" % show(both))
        t.expect("lt_nao_existe.txt" in both, "a mensagem deveria citar o arquivo: %s" % show(both))
        t.expect("closed" not in low(r.out), "não deveria dizer 'closed' de um arquivo que nem abriu")

    @t.case("arquivo sem permissão de leitura: mensagem de erro, sem traceback")
    def _():
        no_root(t)
        with Files(t, ex, {DATA: FRAGMENT}, mode=0):
            r = t.run(EX0, args=(DATA,))
        both = r.out + r.err
        t.expect(denied_msg(both), "esperado \"Permission denied\", veio %s" % show(both))
        t.expect(LINES[0] not in r.out, "mostrou o conteúdo de um arquivo sem permissão?")

    @t.case("diretório no lugar do arquivo: não quebra")
    def _():
        r = t.run(EX0, args=(".",))
        t.expect((r.out + r.err).strip() != "", "não imprimiu nenhuma mensagem de erro")


# ---- ex1 / ex2 -----------------------------------------------------------


def transformer(t, rel, ex, stream):
    """Cases shared by ex1 (errors anywhere, input()) and ex2 (errors on
    stderr, sys.stdin)."""

    def run(files, args, stdin, mode=None):
        with Files(t, ex, files, mode) as fs:
            r = t.run(rel, args=args, stdin=stdin)
            saved = fs.read(OUT)
            listing = set(os.listdir(t.path(ex)))
            return r, saved, listing - fs.before - set(files)

    @t.case("lê e mostra o arquivo original")
    def _():
        r, _, _ = run({DATA: FRAGMENT}, (DATA,), "\n")
        check_shown(t, r.out, LINES)
        t.expect("closed" in low(r.out), "esperado 'File ... closed.' depois da leitura: %s" % show(r.out))

    @t.case("mostra o conteúdo transformado (# no fim de cada linha)")
    def _():
        r, _, _ = run({DATA: FRAGMENT}, (DATA,), "\n")
        check_shown(t, r.out, [line + "#" for line in LINES])
        for line in LINES:
            t.expect(r.out.find(line + "\n") < r.out.find(line + "#"),
                     "o conteúdo original deveria aparecer antes do transformado")
        t.expect("##" not in r.out, "uma linha ganhou dois '#': %s" % show(r.out))

    @t.case("nome vazio: não salva nada")
    def _():
        r, saved, new = run({DATA: FRAGMENT}, (DATA,), "\n")
        t.expect(not new, "com nome vazio não deveria criar arquivo, criou %s" % show(sorted(new)))
        t.expect("sav" in low(r.out.split(LINES[-1] + "#")[-1]),
                 "esperado uma mensagem tipo 'Not saving data.', veio %s" % show(r.out))

    @t.case("salva no arquivo pedido (exemplo do subject)")
    def _():
        r, saved, _ = run({DATA: FRAGMENT}, (DATA,), OUT + "\n")
        t.expect(saved is not None, "o arquivo %s não foi criado (o nome lido veio com '\\n'?)" % show(OUT))
        want = [line + "#" for line in LINES]
        t.expect(saved.splitlines() == want, "conteúdo salvo: esperado %s, veio %s"
                 % (show("\n".join(want) + "\n"), show(saved)))
        tail = r.out.split(LINES[-1] + "#")[-1]
        t.expect(OUT in tail and "saved" in low(tail),
                 "esperado 'Saving data to ...' / 'Data saved in file ...', veio %s" % show(tail))

    @t.case("substitui o arquivo de destino se ele já existe")
    def _():
        r, saved, _ = run({DATA: FRAGMENT, OUT: "conteudo antigo\n" * 50}, (DATA,), OUT + "\n")
        want = "\n".join(line + "#" for line in LINES)
        t.expect(saved is not None and saved.rstrip("\n") == want,
                 "esperado o arquivo substituído por %s, veio %s" % (show(want + "\n"), show(saved)))

    @t.case("arquivo sem \\n no final: # em todas as linhas")
    def _():
        r, saved, _ = run({DATA: "alpha\nbeta"}, (DATA,), OUT + "\n")
        t.expect(saved is not None and saved.splitlines() == ["alpha#", "beta#"],
                 "esperado %s, veio %s" % (show("alpha#\nbeta#\n"), show(saved)))

    @t.case("linhas vazias no meio também ganham #")
    def _():
        r, saved, _ = run({DATA: "a\n\nb\n"}, (DATA,), OUT + "\n")
        t.expect(saved is not None and saved.splitlines() == ["a#", "#", "b#"],
                 "esperado %s, veio %s" % (show("a#\n#\nb#\n"), show(saved)))

    @t.case("arquivo vazio: não quebra")
    def _():
        run({DATA: ""}, (DATA,), "\n")

    @t.case("arquivo inexistente: mensagem de erro, sem traceback")
    def _():
        r, _, new = run({}, ("lt_nao_existe.txt",), "\n")
        where = r.err if stream == "err" else r.out + r.err
        t.expect(missing_msg(where), "esperado \"No such file or directory\" %s, veio stdout=%s stderr=%s"
                 % ("no stderr" if stream == "err" else "na saída", show(r.out), show(r.err)))
        t.expect("#" not in r.out, "não deveria transformar nada se a leitura falhou: %s" % show(r.out))

    @t.case("arquivo sem permissão de leitura: mensagem de erro, sem traceback")
    def _():
        no_root(t)
        r, _, _ = run({DATA: FRAGMENT}, (DATA,), "\n", mode=0)
        where = r.err if stream == "err" else r.out + r.err
        t.expect(denied_msg(where), "esperado \"Permission denied\" %s, veio stdout=%s stderr=%s"
                 % ("no stderr" if stream == "err" else "na saída", show(r.out), show(r.err)))

    @t.case("erro ao salvar (pasta inexistente): avisa e não quebra")
    def _():
        bad = "lt_pasta_inexistente/out.txt"
        r, _, _ = run({DATA: FRAGMENT}, (DATA,), bad + "\n")
        where = r.err if stream == "err" else r.out + r.err
        t.expect(missing_msg(where), "esperado o erro \"No such file or directory\" %s, veio stdout=%s stderr=%s"
                 % ("no stderr" if stream == "err" else "na saída", show(r.out), show(r.err)))
        t.expect("data saved" not in low(r.out), "disse que salvou mesmo com erro: %s" % show(r.out))

    @t.case("erro ao salvar (destino sem permissão): avisa e não quebra")
    def _():
        no_root(t)
        with Files(t, ex, {OUT: "protegido\n"}, mode=0o444):
            r, _, _ = run({DATA: FRAGMENT}, (DATA,), OUT + "\n")
        where = r.err if stream == "err" else r.out + r.err
        t.expect(denied_msg(where), "esperado \"Permission denied\" %s, veio stdout=%s stderr=%s"
                 % ("no stderr" if stream == "err" else "na saída", show(r.out), show(r.err)))
        t.expect("data saved" not in low(r.out), "disse que salvou mesmo com erro: %s" % show(r.out))

    return run


def ex1(t):
    t.group("ex1 ft_archive_creation")
    t.case("só usa o que é permitido", lambda: t.authorized(
        EX1, allowed={"len", "open", "print", "input"} | TYPES, imports={"sys", "typing"}))
    t.case("não usa `with` (só no ex3)", lambda: static_no_with(t, EX1))
    t.case("fecha os arquivos com .close()", lambda: static_close(t, EX1))

    @t.case("sem argumento: mostra o Usage")
    def _():
        r = t.run(EX1, stdin="\n")
        t.expect("usage" in low(r.out + r.err), "esperado uma mensagem 'Usage: ...', veio %s" % show(r.out + r.err))

    transformer(t, EX1, "ex1", "any")


def ex2(t):
    t.group("ex2 ft_stream_management")
    t.case("só usa o que é permitido (sem input())", lambda: t.authorized(
        EX2, allowed={"len", "open", "print"} | TYPES, imports={"sys", "typing"}))
    t.case("não usa `with` (só no ex3)", lambda: static_no_with(t, EX2))
    t.case("fecha os arquivos com .close()", lambda: static_close(t, EX2))

    @t.case("sem argumento: mostra o Usage")
    def _():
        r = t.run(EX2, stdin="\n")
        t.expect("usage" in low(r.out + r.err), "esperado uma mensagem 'Usage: ...', veio %s" % show(r.out + r.err))

    run = transformer(t, EX2, "ex2", "err")

    @t.case("erro vai só para o stderr, não para o stdout")
    def _():
        r, _, _ = run({}, ("lt_nao_existe.txt",), "\n")
        t.expect(not missing_msg(r.out), "a mensagem de erro apareceu no stdout: %s" % show(r.out))
        t.expect(r.out.strip() != "", "o cabeçalho ('=== ... ===', 'Accessing ...') deveria ir para o stdout")

    @t.case("erro no stderr com prefixo claro ([STDERR])")
    def _():
        r, _, _ = run({}, ("lt_nao_existe.txt",), "\n")
        t.expect("stderr" in low(r.err), "esperado o prefixo '[STDERR]' na mensagem de erro, veio %s" % show(r.err))

    @t.case("conteúdo normal vai para o stdout, stderr vazio")
    def _():
        r, _, _ = run({DATA: FRAGMENT}, (DATA,), OUT + "\n")
        check_shown(t, r.out, LINES + [line + "#" for line in LINES])
        t.expect(r.err.strip() == "", "nada deu errado, mas o stderr tem %s" % show(r.err))

    @t.case("pergunta o nome no stdout e lê do stdin")
    def _():
        r, saved, _ = run({DATA: FRAGMENT}, (DATA,), OUT + "\n")
        t.expect(saved is not None, "o nome lido do stdin não foi usado para salvar (sobrou '\\n' no nome?)")
        tail = r.out.split(LINES[-1] + "#")[-1]
        t.expect(tail.strip() != "", "o prompt do nome do arquivo não apareceu no stdout")

    @t.case("stdin fechado (EOF) sem nome: não quebra, não salva")
    def _():
        r, _, new = run({DATA: FRAGMENT}, (DATA,), "")
        t.expect(not new, "com stdin vazio não deveria criar arquivo, criou %s" % show(sorted(new)))


# ---- ex3 ---------------------------------------------------------------

PROBE_READ = """
import os
from ft_vault_security import secure_archive
found = "NONE"
for c in ("DEFAULT", "r", "read", "R", "READ", "Read", 0, 1, 2):
    p = "lt_probe_r.txt"
    with open(p, "w") as f:
        f.write("probe\\n")
    try:
        r = secure_archive(p) if c == "DEFAULT" else secure_archive(p, c)
    except Exception:
        r = None
    with open(p) as f:
        same = f.read() == "probe\\n"
    os.remove(p)
    if same and isinstance(r, tuple) and len(r) == 2 and r[0] is True and r[1] == "probe\\n":
        found = c
        break
print("@@" + repr(found))
"""

PROBE_WRITE = """
import os
from ft_vault_security import secure_archive
found = "NONE"
for c in ("w", "write", "W", "WRITE", "Write", "wt", 1, 2, 0):
    p = "lt_probe_w.txt"
    if os.path.exists(p):
        os.remove(p)
    try:
        r = secure_archive(p, c, "abc\\ndef\\n")
    except Exception:
        r = None
    ok = False
    if os.path.exists(p):
        with open(p) as f:
            ok = f.read() == "abc\\ndef\\n"
        os.remove(p)
    if ok and isinstance(r, tuple) and len(r) == 2 and r[0] is True:
        found = c
        break
print("@@" + repr(found))
"""


def marker(t, r):
    lines = [line for line in r.out.splitlines() if line.startswith("@@")]
    t.expect(lines, "o teste não terminou: %s" % show(r.out + r.err))
    return ast.literal_eval(lines[-1][2:])


def ex3(t):
    t.group("ex3 ft_vault_security")
    ex = "ex3"
    t.case("só usa o que é permitido (open/print, sem imports)", lambda: t.authorized(
        EX3, allowed={"open", "print"} | TYPES, imports=set()))
    t.case("define secure_archive()", lambda: t.defines(EX3, "secure_archive"))

    @t.case("todo open() está dentro de um `with`")
    def _():
        tree = t.tree(EX3)
        in_with = {id(item.context_expr) for n in ast.walk(tree) if isinstance(n, ast.With) for item in n.items}
        opens = [n for n in ast.walk(tree) if isinstance(n, ast.Call)
                 and isinstance(n.func, ast.Name) and n.func.id == "open"]
        t.expect(opens, "secure_archive não chama open()")
        bad = [n.lineno for n in opens if id(n) not in in_with]
        t.expect(not bad, "open() fora de `with` na(s) linha(s) %s" % ", ".join(map(str, bad)))

    @t.case("secure_archive tem os parâmetros pedidos (nome, ação opcional, conteúdo opcional)")
    def _():
        tree = t.tree(EX3)
        fn = [n for n in tree.body if isinstance(n, ast.FunctionDef) and n.name == "secure_archive"]
        t.expect(fn, "não define secure_archive")
        a = fn[0].args
        pos = a.posonlyargs + a.args
        t.expect(len(pos) >= 3, "esperado 3 parâmetros, tem %d" % len(pos))
        t.expect(len(a.defaults) >= len(pos) - 1 and len(a.defaults) >= 2,
                 "a ação e o conteúdo deveriam ser opcionais (com valor padrão)")

    state = {}

    def probe(code, key):
        if key not in state:
            with Files(t, ex):
                state[key] = marker(t, t.py(code, ex=ex, files=("ft_vault_security.py",)))
        return state[key]

    def read_args(p):
        c = probe(PROBE_READ, "r")
        t.expect(c != "NONE", "não achei como ler com secure_archive (nem padrão, nem 'r'/'read'/0/1)")
        return (p,) if c == "DEFAULT" else (p, c)

    def write_args(p, content):
        c = probe(PROBE_WRITE, "w")
        t.expect(c != "NONE", "não achei como escrever com secure_archive (tentei 'w', 'write', 1, 2, 0)")
        return (p, c, content)

    def call(args, files=None, mode=None, keep=None):
        code = "from ft_vault_security import secure_archive\nprint('@@' + repr(secure_archive(*%r)))\n" % (args,)
        with Files(t, ex, files, mode) as fs:
            res = marker(t, t.py(code, ex=ex, files=("ft_vault_security.py",)))
            return res, fs.read(keep) if keep else None

    def tup(res):
        t.expect(isinstance(res, tuple) and len(res) == 2 and isinstance(res[0], bool) and isinstance(res[1], str),
                 "esperado uma tupla (bool, str), veio %s" % show(res))

    @t.case("lê um arquivo normal: (True, conteúdo)")
    def _():
        res, _ = call(read_args(DATA), {DATA: FRAGMENT})
        tup(res)
        t.expect(res == (True, FRAGMENT), "esperado %s, veio %s" % (show((True, FRAGMENT)), show(res)))

    @t.case("lê arquivo vazio: (True, '')")
    def _():
        res, _ = call(read_args(DATA), {DATA: ""})
        t.expect(res == (True, ""), "esperado (True, ''), veio %s" % show(res))

    @t.case("lê arquivo inexistente: (False, mensagem)")
    def _():
        res, _ = call(read_args("/not/existing/file"))
        tup(res)
        t.expect(res[0] is False and missing_msg(res[1]),
                 "esperado (False, \"[Errno 2] No such file or directory: ...\"), veio %s" % show(res))

    @t.case("lê arquivo sem permissão: (False, mensagem)")
    def _():
        no_root(t)
        res, _ = call(read_args(DATA), {DATA: FRAGMENT}, mode=0)
        tup(res)
        t.expect(res[0] is False and denied_msg(res[1]),
                 "esperado (False, \"[Errno 13] Permission denied: ...\"), veio %s" % show(res))

    @t.case("lê um diretório: (False, mensagem), sem exceção")
    def _():
        res, _ = call(read_args("."))
        tup(res)
        t.expect(res[0] is False, "esperado (False, ...), veio %s" % show(res))

    @t.case("escreve num arquivo novo: (True, mensagem) e o arquivo tem o conteúdo")
    def _():
        res, got = call(write_args(OUT, FRAGMENT), keep=OUT)
        tup(res)
        t.expect(res[0] is True, "esperado (True, ...), veio %s" % show(res))
        t.expect(got == FRAGMENT, "conteúdo escrito: esperado %s, veio %s" % (show(FRAGMENT), show(got)))

    @t.case("escrever substitui um arquivo que já existe")
    def _():
        _, got = call(write_args(OUT, "novo\n"), {OUT: "antigo\n" * 20}, keep=OUT)
        t.expect(got == "novo\n", "esperado %s, veio %s" % (show("novo\n"), show(got)))

    @t.case("escreve numa pasta inexistente: (False, mensagem)")
    def _():
        res, _ = call(write_args("/not/existing/dir/file.txt", "x"))
        tup(res)
        t.expect(res[0] is False and missing_msg(res[1]), "esperado (False, \"[Errno 2] ...\"), veio %s" % show(res))

    @t.case("escreve num arquivo sem permissão: (False, mensagem)")
    def _():
        no_root(t)
        res, _ = call(write_args(OUT, "x"), {OUT: "protegido\n"}, mode=0o444)
        tup(res)
        t.expect(res[0] is False and denied_msg(res[1]), "esperado (False, \"[Errno 13] ...\"), veio %s" % show(res))

    @t.case("ler e depois escrever o mesmo conteúdo (exemplo do subject)")
    def _():
        ra, wa = read_args(DATA), write_args(OUT, "")
        code = ("from ft_vault_security import secure_archive\n"
                "ok, data = secure_archive(*%r)\n"
                "print('@@' + repr(secure_archive(%r, %r, data)))\n" % (ra, OUT, wa[1]))
        with Files(t, ex, {DATA: FRAGMENT}) as fs:
            res = marker(t, t.py(code, ex=ex))
            got = fs.read(OUT)
        t.expect(res[0] is True and got == FRAGMENT, "esperado (True, ...) e %s copiado, veio %s / %s"
                 % (show(OUT), show(res), show(got)))

    @t.case("o programa roda a demonstração sem quebrar")
    def _():
        with Files(t, ex, {"ancient_fragment.txt": FRAGMENT} if not t.exists("ex3/ancient_fragment.txt") else {}):
            r = t.run(EX3)
        t.expect("(False," in r.out and "(True," in r.out,
                 "esperado as tuplas (False, ...) e (True, ...) na saída, veio %s" % show(r.out))


def suite(t):
    common(t, FILES, typed=FILES)
    ex0(t)
    ex1(t)
    ex2(t)
    ex3(t)
