---
title: Contribuir
order: 11
description: Como o código está organizado e como adicionar exercícios, testes e funcionalidades.
---

O lightyear é open source (MIT) e aceita contribuições: correções, exercícios do simulador, suítes de teste para outros projetos. O código está em [github.com/nvizble/Lightyear42](https://github.com/nvizble/Lightyear42), e o [AGENTS.md](https://github.com/nvizble/Lightyear42/blob/main/AGENTS.md) é a referência técnica completa.

## Arquitetura {#arquitetura}

Go com Clean Architecture. A regra de ouro: **nada de lógica de negócio nos comandos Cobra**. Os comandos leem argumentos e flags e chamam os services; os services têm as regras; os repositories falam com a API e com o GitHub Releases.

```text
cmd/              comandos Cobra: só parseiam args/flags e chamam services
cmd/lightyear/    entrypoint (package main)
internal/
  api/            cliente HTTP da API da 42 (retries, erros tipados)
  auth/           OAuth2 e keyring
  cache/          cache local em SQLite
  config/         Viper e caminhos (XDG)
  models/         domínio
  services/       regras de negócio
  repository/     acesso à API e ao GitHub Releases
  notify/         push via ntfy e o estado do que já foi avisado
  update/         download, extração e troca do binário
  subjects/       catálogo embutido de PDFs de subjects
  exam/           exercícios embutidos e o grader do simulador
  tester/         lightyear test: checagens do subject e suítes por projeto
  editor/         núcleo do editor: buffer, cursor, viewport, undo
  vim/            controller estilo Vim: modos, contadores, motions, operadores
  syntax/         realce de sintaxe com Tree-sitter (cgo)
  lsp/            cliente LSP (só stdlib)
  tui/            Bubble Tea: app, dashboard, editorview, testview
pkg/              só APIs públicas exportáveis
```

A stack: Go 1.25+, Cobra, Viper, Bubble Tea e Lip Gloss, `golang.org/x/oauth2`, SQLite sem CGO (`modernc.org/sqlite`) e Tree-sitter, a única parte em C (por isso o build precisa de um compilador C).

## Ambiente de desenvolvimento {#ambiente}

Precisa de **Go 1.25+** e de um **compilador C**. Para os testes do `internal/tester`, também de `cc`, `make`, `nm`, `ar` e `python3` (os testes que dependem deles são pulados se faltarem).

```bash
git clone https://github.com/nvizble/Lightyear42.git
cd Lightyear42
make build     # compila ./lightyear
make install   # instala em ~/go/bin/lightyear
make test      # go test ./...
make lint      # golangci-lint
make fmt       # gofmt (e goimports, se instalado)
make vet       # go vet
```

O CI roda `go vet`, `go test` e o build no Ubuntu e no macOS, e o golangci-lint, em todo pull request.

## Adicionar exercícios ao simulador {#exercicios}

Cada exercício é uma pasta em `internal/exam/exercises/rank<NN>/level<N>/<nome>/`:

```text
internal/exam/exercises/rank02/level1/first_word/
├── subject.en.txt    enunciado em inglês (obrigatório)
├── subject.pt.txt    enunciado em português
├── meta.json         arquivos a entregar e os argumentos de cada teste
├── ref/              solução de referência, com os mesmos nomes de arquivo de "files"
├── provided/         (opcional) arquivos que a prova entrega, como um header
└── main.c            (opcional) main de teste, para exercícios de função
```

O `meta.json` lista os arquivos da entrega e, em `tests`, os argumentos (sem o nome do programa) de cada execução:

```json
{
  "files": ["first_word.c"],
  "tests": [
    [],
    ["FOR PONY"],
    ["   lorem,ipsum   "],
    ["a", "b"]
  ]
}
```

O grader compila a entrega e a `ref/` com `cc -Wall -Wextra -Werror` e compara a saída padrão das duas em cada teste. Nos exercícios de função, o `main.c` chama a função e imprime o resultado (veja `level1/ft_strlen/`).

Um exercício em Python tem `"lang": "python"`, o nome da função em `function`, um único arquivo `.py` e, opcionalmente, o que é proibido em `forbidden` (`"sorted"` para uma builtin ou um módulo, `"list.sort"` para um método). Cada teste é uma lista de chamadas, cada uma com os argumentos como expressão Python:

```json
{
  "lang": "python",
  "files": ["py_cryptic_sorter.py"],
  "function": "cryptic_sorter",
  "forbidden": ["sorted", "list.sort"],
  "tests": [
    ["[3, 1, 2]", "[]"],
    ["list(range(500, 0, -1))"]
  ]
}
```

O grader roda as chamadas de um teste num processo `python3` só e compara o `repr()` de cada retorno (ou `raise <Exceção>`) com o da `ref/`.

Os testes do pacote (`go test ./internal/exam/`) carregam o catálogo e conferem que **toda solução de referência passa** nos próprios testes. Um rank novo aparece no `lightyear exam list` e vale em `lightyear exam start --rank <NN>`. Se usar material de terceiros, credite em `internal/exam/CREDITS.md`.

## Adicionar uma suíte de testes {#suites}

Os testes do `lightyear test` ficam em `internal/tester/` e são embutidos no binário:

```text
internal/tester/
├── tester.go           a lista de projetos (var projects)
├── runner.go           cópia do projeto, make, norminette, nm, compilação das suítes
├── libft.go …          um arquivo Go por projeto: as checagens do subject
├── c/lt.h, c/lt.c      runtime das suítes em C: fork por caso, malloc instrumentado
├── c/<projeto>/        suítes em C, um arquivo por função
├── py/lt.py            runtime das suítes de Python
├── py/moduleNN.py      uma suíte por módulo de Python
├── sh/born2beroot.sh   o teste do born2beroot (roda sozinho também)
└── testdata/<projeto>/ uma solução de referência de cada projeto
```

### Em C

Cada função tem um arquivo `c/<projeto>/<função>.c` com uma função `lt_suite_<função>()`. Os casos usam as macros do `lt.h`: `TEST(nome)` roda o corpo num processo próprio e `CHECK(condição, formato, ...)` falha o caso com a mensagem. O runtime já acusa crash, timeout, vazamento, double free e overflow; `lt_alloc_fails()` faz cada `malloc` falhar, um de cada vez.

```c
#include "helpers.h"

void	lt_suite_ft_strlen(void)
{
	lt_group("ft_strlen");
	TEST("ft_strlen(\"hello\")")
		CHECK(ft_strlen("hello") == 5, "esperado 5, recebido %zu", ft_strlen("hello"));
}
```

Cada suíte é compilada sozinha contra o header do aluno, então um protótipo errado derruba só os testes daquela função. Um projeto novo também precisa de um `<projeto>.go` com as checagens (README, norminette, Makefile, funções permitidas…) e de uma entrada em `var projects`, no `tester.go`.

### Em Python

Uma suíte é um `py/moduleNN.py` que define `suite(t)`. O `common(t, files, typed)` do `lt.py` faz as checagens que todo módulo tem (entrega, flake8, mypy, type hints, sintaxe); o resto são casos com `@t.case("nome")` que rodam o código do aluno num processo próprio (`t.run`, `t.py`) e conferem com `t.expect`. Registre o módulo em `var projects` com `pythonModule("NN")` (passe os marcadores da raiz, se não for `ex0`).

### Solução de referência e bugs plantados

Toda suíte se testa contra uma **solução de referência** em `testdata/<projeto>/`, que tem que passar inteira (`TestLibftReferencePasses`, `TestPythonModulesReferencePasses`…).

Para provar que os testes pegam erros, os projetos em C também têm **bugs plantados**: em `libft_test.go`, `printf_test.go` e `gnl_test.go`, cada caso aplica uma troca de texto na referência (por exemplo, `malloc(a + b + 1)` vira `malloc(a + b)`) e confere que o grupo certo falha com o status e a mensagem esperados (`KO` com "heap overflow").

{% raw %}
```go
{"strjoin forgets the +1",
	[]edit{{"part2.c", "malloc(a + b + 1)", "malloc(a + b)"}},
	"ft_strjoin", KO, "heap overflow"},
```
{% endraw %}

## Changelog {#changelog}

Toda mudança visível para quem usa entra no `CHANGELOG.md`, na seção da próxima versão (a próxima canary ou estável), escrita para o usuário:

```markdown
## [1.3.0-canary.36] — 2026-10-12

- `lightyear test push_swap`: …
```

O `lightyear --version` mostra a seção da versão instalada, e um teste quebra se o arquivo sair desse formato.

## Releases {#releases}

Um push de tag `v*` dispara o GoReleaser, que publica os binários (Linux, macOS e Windows), o `.deb` e o `checksums.txt` no GitHub Releases.

- **Canary:** as novidades saem como tags `v1.3.0-canary.N`, publicadas como pre-release.
- **Estável:** as correções da série estável saem da branch `release/1.2.x`.

O build roda num runner macOS: clang para darwin e Zig para Linux (estático, com musl) e Windows. Pull requests que mexem no build (`.goreleaser.yaml`, o workflow de release, `go.mod`, `go.sum`) rodam um build de teste, sem publicar nada.

## Pull requests {#pr}

1. Faça um fork e crie uma branch a partir da `main`.
2. Mantenha a mudança pequena e focada: um assunto por PR.
3. Rode `make test` e `make lint`.
4. Adicione a entrada no `CHANGELOG.md`, se a mudança for visível.
5. Abra o PR contra a `main` explicando o problema e a solução.
