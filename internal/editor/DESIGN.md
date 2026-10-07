# Editor de Código Modal Embutido para TUI em Go

> **Status:** Fases 1 (Editor Core, `internal/editor`), 2 (edição modal,
> `internal/vim`), 3 (command parser: contadores, operador × motion), 4
> (Visual Mode), 5 (Registers) e 6 (Syntax Highlighting, `internal/syntax`)
> implementadas, e a Fase 7 (LSP, `internal/lsp`) também: diagnostics,
> hover, go-to-definition e completion. A Fase 8 (avançados) também está
> implementada, com os limites registrados abaixo. O
> componente fica em `internal/tui/editorview` (comando experimental
> escondido `lightyear edit <arquivo>`, modal por padrão; `--plain` para o
> editor sem modos) e roda dentro do app, na aba Exam (`e`): subject só
> leitura à esquerda, entrega à direita, `:wq` volta ao app.
>
> **Desvios desta implementação em relação ao texto abaixo:**
> - o componente visual mora em `internal/tui/editorview` (a TUI do projeto
>   fica em `internal/tui`), não em `internal/ui/editor`;
> - `Selection` entra só na Fase 4 (Visual Mode);
> - colunas do `Position` contam runes; a expansão de tabs (tab = 4) é usada
>   no viewport e nos movimentos verticais (coluna "visual" preservada);
> - a Fase 2 já traz `I A O` (listados na seção 9) e uma linha de comando
>   mínima (`:w`, `:q`, `:q!`, `:wq`, `:x`), para sair e salvar do jeito Vim;
> - uma sessão de INSERT inteira é um passo de undo (grupos no core:
>   `BeginGroup`/`EndGroup`);
> - a Fase 3 inclui `^` (primeiro não-branco), útil em código com tabs; as
>   motions ficam em `internal/vim/motion.go` (um pacote só, sem
>   `motions/`), e cada uma devolve um `Target` (exclusivo, inclusivo ou
>   linewise) que os operadores aplicam;
> - contadores também valem para `x`, `u` e `Ctrl-r`; `y` entra com os
>   registers (Fase 5), o parser já aceita qualquer operador;
> - Fase 4: `v`/`V` (e arrastar o mouse) selecionam, `o` troca a ponta e
>   os operadores agem na seleção. Diferente do Vim, `$` no Visual não
>   inclui a quebra de linha (vale o último caractere, como `g_`); terminar
>   a seleção numa linha vazia inclui a quebra, como no Vim. O `u` volta o
>   cursor para o início do trecho alterado;
> - Fase 5: o register sem nome (`"`) guarda o que `d`, `c`, `x` e `y`
>   tiram, e `p`/`P` colam (com contador). Junto vieram o operador `c`
>   (adiantado da seção 13: `cw` vai só até o fim da palavra e `cc` mantém
>   a indentação), os atalhos `C`, `D` e `Y`, e `y`/`c`/`p` no Visual (o
>   texto trocado pelo `p` vai para o register, como no Vim). Os registers
>   são um mapa por nome no Controller: named registers (`"a`) só precisam
>   do parser. Ainda sem o clipboard do sistema;
> - Fase 6: Tree-sitter **oficial**, via cgo (decisão de 2026-10-07; havia
>   um Tree-sitter em Go puro e o Chroma como alternativas sem CGO). É a
>   única parte em C do projeto: o release roda num runner macOS (clang
>   para darwin, Zig para Linux estático/musl e Windows) e o `go install`
>   precisa de `cc`. Gramáticas: C, C++, Go, Python e Rust. As queries de
>   highlight vêm dos repositórios das gramáticas (`internal/syntax/
>   queries`, MIT); quando dois padrões capturam o mesmo nó, vence o de
>   baixo (a do Go foi reordenada para isso). O core avisa cada mudança
>   (`Editor.OnChange`, que o LSP também vai usar) e o `syntax.Highlighter`
>   aplica a edição na árvore e reparseia de forma incremental só quando a
>   tela pede as cores, consultando só as linhas visíveis;
> - Fase 7, primeira entrega: cliente LSP próprio (`internal/lsp`, só
>   stdlib: JSON-RPC com Content-Length), um servidor por arquivo aberto,
>   sync completo (`didChange` com o texto inteiro, como a seção 18 prevê
>   para o início) e diagnostics: sublinhados no código, `●` colorido na
>   margem, contagem e a mensagem da linha do cursor na statusline. O
>   cliente negocia UTF-32 (`positionEncoding`) e converte de UTF-16 quando
>   o servidor não aceita. O clangd recebe `-xc`/`-xc++ -std=c++98` além de
>   `-Wall -Wextra -Werror` como `fallbackFlags`. No macOS, o primeiro
>   clangd que um binário novo do lightyear abre pode levar ~15s (o
>   sistema verifica); o servidor sobe em segundo plano e a statusline
>   mostra "clangd…" até ficar pronto;
> - Fase 7, segunda entrega: `K` (hover numa caixa abaixo do cursor;
>   qualquer tecla fecha), `gd` (no mesmo arquivo move o cursor; em outro,
>   avisa onde está — abrir outros arquivos vem com os buffers da Fase 8),
>   `]d`/`[d` (próximo/anterior diagnostic, dando a volta) e completion no
>   INSERT: abre sozinha na primeira letra de uma palavra e depois de `.`,
>   `->` e `::` (ou com `Ctrl-n`/`Ctrl-Space`), filtra pelo prefixo enquanto
>   se digita, `Ctrl-n`/`Ctrl-p` (ou setas) escolhem, `Tab`/`Enter` aceitam
>   e `Esc` fecha e sai do INSERT. Respostas atrasadas são descartadas. O
>   Vim Controller continua sem conhecer LSP: o host registra comandos
>   (`Controller.Commands`) e os prefixos de duas teclas (`g`, `[`, `]`)
>   passaram a ser genéricos;
> - Fase 8, em partes. Primeira: text objects (`iw aw`, `i" a"` e as outras
>   aspas, `i( a( ib`, `i{ a{ iB`, `i[ a[`, `i< a<`) depois de operador
>   ou no Visual; os de colchetes cruzam linhas e, quando o bloco interno é
>   de linhas inteiras, viram linewise (`di{` mantém `{` e `}`, `ci{` mantém
>   a indentação). `f F t T ; ,` (com contador; `;` depois de `t` não fica
>   preso). Busca `/ ? n N * #`: o padrão é **texto literal** (não regex),
>   `*`/`#` buscam a palavra inteira, as ocorrências ficam destacadas até
>   `:noh`. Ainda não: busca como motion de operador (`d/x`) e contadores
>   em text objects (`2i(`);
> - Fase 8, segunda parte: `.` repete as teclas do último comando que mudou
>   o texto (uma sessão de INSERT inteira, colagens incluídas; um contador
>   novo substitui o do comando); `u`, `Ctrl-r`, `.` e macros não viram o
>   `.`. Macros: `q{a-z}` grava, `q` para (a statusline mostra "gravando
>   @a"), `@{a-z}` e `@@` tocam, com contador; uma macro que chama a si
>   mesma para em 20 níveis. Ficam num mapa próprio (ainda não nos
>   registers de texto). A completion aceita entra pelo controller, então
>   `.` e macros a repetem (só o que completa a palavra, como no Vim).
>   Depois: buffers, splits e os extras de LSP;
> - Fase 8, terceira parte: buffers. `lightyear edit a.c b.c` abre vários
>   arquivos; `:e arquivo` abre outro (ou volta a ele), `:bn`/`:bp`/`:b N`
>   (ou parte do nome) trocam, `:ls` lista, `:bd`/`:bd!` fecham. O `gd` com
>   a definição em outro arquivo abre um buffer e `Ctrl-o` volta (jump list
>   dos saltos do `gd`). `:q`, `:wq`, `:x` e `:qa` olham todos os buffers
>   (`E162` quando outro tem alterações), `:wa` salva todos e `:q!`/`:qa!`
>   saem. Registers, macros, busca e `.` valem entre buffers
>   (`Controller.SetEditor`); os comandos de buffer são do host
>   (`Controller.ExCommands`). O cliente LSP virou multi-documento: um
>   servidor por projeto (nome + raiz), compartilhado pelos buffers, com
>   `didOpen`/`didClose` e diagnostics por arquivo;
> - Fase 8, quarta parte: splits. `:sp`/`:vsp` (`Ctrl-w s`/`v`), com
>   arquivo opcional, abrem uma janela acima/à esquerda e entram nela;
>   `Ctrl-w w`/`W`/`h`/`j`/`k`/`l` trocam, `:close`/`Ctrl-w c` e
>   `:only`/`Ctrl-w o` fecham, e `:q` fecha a janela enquanto houver mais de
>   uma (o buffer continua aberto). Cada janela tem uma linha de título
>   (ativa em destaque); o clique foca a janela e a roda rola a que está
>   sob o mouse. Limites desta versão: todas as janelas na mesma orientação
>   (misturar `:sp` e `:vsp` avisa) e duas janelas no mesmo arquivo
>   compartilham o cursor e a rolagem, que moram no editor;
> - Fase 8, quinta parte: extras de LSP. `grn` abre `:Rename ` (rename no
>   projeto inteiro), `grr` lista as referências (uma só pula direto;
>   duplicatas do mesmo arquivo por caminhos diferentes, como
>   `/var` e `/private/var` no macOS, aparecem uma vez), `gra` lista as code
>   actions (os fix-its do clangd entram pelos diagnostics da linha; ações
>   com comando rodam no servidor, que devolve as edições por
>   `workspace/applyEdit`), e `:Format` formata. As edições abrem os outros
>   arquivos em buffers modificados (um passo de undo por arquivo; `:wa`
>   salva) e o buffer e o cursor atuais ficam onde estavam. **Formatação de
>   C/C++ só com `.clang-format` no projeto**: o estilo padrão do clangd
>   quebra a norminette, então sem ele o editor avisa e não formata (Go,
>   Python e Rust usam o formatador do servidor). Os prefixos de comandos
>   do host podem ter mais de duas teclas (`gr` espera o `n`/`r`/`a`);
> - Diagnostics sob demanda: com o mouse parado sobre o trecho sublinhado
>   (inclusive a célula depois do fim da linha, onde fica o `;` que falta)
>   aparece uma caixa com as mensagens inteiras, notas incluídas; sobre a
>   margem, todos os da linha. O `K` mostra os da linha do cursor acima do
>   hover do servidor. O `lightyear edit` passou a pedir todo movimento do
>   mouse ao terminal (o app já pedia);
> - Janelas redimensionáveis: cada janela tem uma fração da tela (peso),
>   então o tamanho acompanha o terminal. Arrastar a borda (`│` entre as
>   lado a lado, a linha de título nas empilhadas), `Ctrl-w > < + - = | _`
>   (com contador) e `:resize N` / `:vertical resize N` (`+N`/`-N`); o
>   espaço vem da janela vizinha, com mínimo de 12 colunas ou 2 linhas. O
>   `:vsp` divide a janela atual ao meio e fechar uma janela devolve o
>   espaço à vizinha. Na aba Exam o app guarda os pesos (`Shares`) e reabre
>   o editor igual (`WithShares`);
> - Correção com um clique: a caixa do mouse sobre um diagnostic com
>   correção ("fix available") mostra "▸ clique aqui para corrigir" e fica
>   enquanto o mouse estiver nela; o clique pede as code actions com os
>   diagnostics da caixa (a partir da linha deles) e aplica direto quando há
>   uma só — várias abrem a lista, como o `gra`.

## 1. Visão Geral

O projeto consiste no desenvolvimento de um **editor de código embutido em uma aplicação TUI escrita em Go**, oferecendo uma experiência de edição inspirada no Vim, porém sem o objetivo de implementar compatibilidade completa com Vim, Vimscript ou seu ecossistema de plugins.

O editor será desenvolvido como um componente independente dentro da aplicação existente, permitindo edição eficiente por teclado e funcionalidades modernas de desenvolvimento através da integração com **Language Server Protocol (LSP)**.

O objetivo principal é combinar três características:

- edição modal inspirada no Vim;
- integração nativa com a TUI existente;
- funcionalidades de IDE através de LSP.

O editor deverá permanecer desacoplado da interface da aplicação, do protocolo LSP e da interpretação dos comandos Vim-like.

## 2. Objetivos

O editor deverá permitir que o usuário edite arquivos diretamente dentro da aplicação TUI sem precisar abrir um editor externo. A experiência de edição deverá priorizar uso através do teclado.

Modos iniciais: **Normal**, **Insert** e **Visual**.

Comandos comuns do Vim deverão estar disponíveis, incluindo movimentos, operações de edição, repetição de comandos e combinações como:

```text
h j k l      w b e      0 $      gg G
i a o        I A O      x
dd dw d$ yy  p P        u Ctrl-r
```

Também deverão ser suportados contadores (`3j`, `5k`, `3dd`, `2dw`, `5yy`).

A arquitetura deverá permitir futuramente comandos mais avançados como `ciw`, `diw`, `daw`, `ci"`, `ci(`, `f`, `t` e `.`.

## 3. Não Objetivos

A primeira versão não pretende implementar um clone completo do Vim. Inicialmente não são objetivos: compatibilidade completa com Vim, interpretação de `.vimrc`, Vimscript, plugins do Vim, emulação de todas as opções internas, todos os text objects, nem reprodução exata de todos os edge cases.

O objetivo é oferecer uma **experiência Vim-like**, e não implementar o Vim novamente.

## 4. Arquitetura Geral

```text
                   Aplicação TUI
                        │
                ┌───────▼───────┐
                │  Editor View  │
                └───────┬───────┘
                  eventos/teclas
                ┌───────▼────────┐
                │ Vim Controller │
                └───────┬────────┘
                     Commands
                ┌───────▼───────┐
                │  Editor Core  │
                └───────┬───────┘
                     Buffer
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
     History         Tree-sitter       LSP
   Undo / Redo        Syntax         Language
                                   Intelligence
```

## 5. Editor Core

Responsável pelas operações fundamentais de edição. **Não conhece** Vim, LSP, Tree-sitter, framework TUI nem atalhos de teclado.

```go
type Editor struct {
    Buffer    *Buffer
    Cursor    Cursor
    Selection *Selection
    Viewport  Viewport
}
```

Operações: `Insert(text)`, `Delete(r Range)`, `Replace(r Range, text)`, `MoveCursor(position)`, `Undo()`, `Redo()`.

## 6. Buffer

Conteúdo textual do documento, com suporte eficiente a inserções, remoções, leitura, substituições, navegação, ranges e arquivos grandes. A implementação inicial pode ser simples, desde que a interface permita migrar para Piece Table, Rope ou Gap Buffer sem que o restante do editor dependa da estrutura.

## 7. Sistema de Posições

```go
type Position struct { Line, Column int }
type Range struct { Start, End Position }
```

Usado por cursor, seleções, edições, motions e syntax highlighting. Integrações externas (LSP) convertem para suas representações.

## 8. Cursor

Conhece linha e coluna. Não interpreta comandos Vim: `j`, `w`, `gg` são interpretados pela camada Vim e convertidos em movimentos do cursor.

## 9. Modos de Edição

- **Normal:** navegação e comandos (`h j k l`, `w b e`, `dd dw yy`, `p`, `gg G`).
- **Insert:** entradas `i a I A o O`; caracteres vão direto ao Editor Core; `Esc` volta ao Normal.
- **Visual:** `v` + motions expandem a seleção; operadores atuam sobre ela.

## 10. Vim Controller

Transforma sequências de teclas em comandos estruturados. `3dw` vira `Count=3, Operator=Delete, Motion=Word`:

```go
type Command struct {
    Count    int
    Operator Operator
    Motion   Motion
}
```

Operadores e motions combinam (`d`/`y` × `w`/`$`/`G`) em vez de cada combinação ser implementada à parte.

## 11. Comandos Pendentes

Ao receber `d`, o editor aguarda a próxima entrada (`dd`, `dw`, `d$`, `dG`). O controller mantém estado intermediário:

```go
type CommandState struct {
    Count    int
    Operator Operator
    Motion   Motion
    Pending  []Key
}
```

## 12. Motions

Iniciais: `h j k l`, `w b e`, `0 $`, `gg G`. Cada motion retorna um destino ou range, reutilizado na navegação e nos operadores (`w` move; `dw` = Delete + WordMotion).

```go
type Motion interface {
    Resolve(editor *Editor) Range
}
```

## 13. Operators

Iniciais: `d` (delete), `y` (yank). Depois: `c`, `>`, `<`, `=`. `operator + motion` produz a operação completa.

## 14. Registers

Um register padrão inicialmente (`dd`/`yy` guardam; `p`/`P` usam), com arquitetura para named registers.

## 15. Undo e Redo

Toda alteração produz uma operação reversível (Insert, Delete, Replace). `u` desfaz, `Ctrl-r` refaz. O histórico pertence ao Editor Core, não ao Vim Controller.

## 16–18. LSP

O editor atua como **cliente LSP** (JSON-RPC) de servidores externos.

**Linguagens-alvo da Fase 7** (decidido com o João):

| Linguagem | Servidor | Instalação (dica mostrada quando falta) |
|---|---|---|
| C / C++ | `clangd` | macOS: vem com o Xcode CLT (`xcrun clangd`); Linux: `apt install clangd` |
| Go | `gopls` | `go install golang.org/x/tools/gopls@latest` |
| Python | `pyright-langserver --stdio`, senão `pylsp` | `npm i -g pyright` ou `pip install python-lsp-server` |
| Rust | `rust-analyzer` | `rustup component add rust-analyzer` |

Notas de implementação:
- achar o binário não basta: o `rust-analyzer` do rustup é um *proxy* que
  falha se o componente não estiver instalado. O cliente precisa tratar
  servidor que morre na inicialização e mostrar a dica de instalação;
- para C (a linguagem das provas da 42), passar ao clangd
  `fallbackFlags: ["-Wall", "-Wextra", "-Werror"]` quando não houver
  `compile_commands.json`, para os diagnósticos baterem com o grader;
- posições LSP são UTF-16 por padrão; o editor usa colunas em runes
  (converter, ou negociar `positionEncoding`).

Também cabem: `typescript-language-server`. Primeiras funcionalidades: diagnostics, completion, hover, go-to-definition (references depois). Sincronização: `didOpen`, `didChange` (completa no início, incremental depois), `didClose`.

## 19. Syntax Highlighting

Não é responsabilidade do LSP: usa um parser incremental, preferencialmente **Tree-sitter** (keywords, strings, comments, functions, types, variables). Buffer → Tree-sitter (syntax) + LSP (intelligence) → Renderer.

## 20. Renderização

A camada visual é da aplicação TUI. O Editor Core fornece texto, cursor, seleção, syntax, diagnostics e viewport.

```text
  1 │ package main
  2 │
  3 │ import "fmt"
  5 │ func main() {
  6 │     fmt.Println("Hello")
  7 │ }
────┴────────────────────────────
 NORMAL │ main.go │ Go │ 6:5
```

## 21. Viewport

O editor não renderiza o documento inteiro: um `Viewport{Top, Left, Width, Height}` determina a região visível e acompanha o cursor.

## 22. Estrutura

`internal/editor` (core), `internal/vim` (controller, modes, parser, command, operator, register, motions), `internal/lsp`, `internal/syntax`, e o componente visual (aqui: `internal/tui/editorview`).

## 23. Fluxo de Entrada

Keyboard → TUI → Editor Component → Vim Controller → Command Parser → Editor Core → Buffer. Mudanças no buffer notificam History, Tree-sitter e LSP; depois Renderer → Terminal.

## 24. Configuração

Configuração própria (ex.: `editor.line_numbers`, `editor.relative_numbers`, `editor.tab_size`, `vim.enabled`, `lsp.go: gopls`). Compatibilidade com `.vimrc` pode ser estudada depois.

## 25. Roadmap

| Fase | Escopo | Status |
|---|---|---|
| 1 — Editor Core | Buffer, Cursor, Viewport, Insert, Delete, Save, Undo, Redo | **implementada** |
| 2 — Edição Modal | NORMAL/INSERT, `hjkl`, `i a o`, `x`, `dd`, `u`, `Ctrl-r` | **implementada** |
| 3 — Command Parser | count + operator + motion (`3j`, `3dd`, `dw`, `3dw`, `d$`) | **implementada** |
| 4 — Visual Mode | `v`, `V` e operações sobre seleções | **implementada** |
| 5 — Registers | `yy`, `dd`, `p`, `P` com register padrão | **implementada** |
| 6 — Syntax Highlighting | Tree-sitter, highlighting incremental | **implementada** |
| 7 — LSP | cliente JSON-RPC; diagnostics, hover, completion, go-to-definition | **implementada** |
| 8 — Avançados | `ciw diw daw`, busca, `f F t T`, `.`, macros, buffers, splits, code actions, rename, format, references | **implementada** |

## 26. Princípios

- **Editor independente do Vim:** o Editor Core não conhece comandos Vim.
- **Vim independente do LSP:** o Vim Controller não conhece LSP.
- **LSP independente da interface:** o cliente LSP não conhece a TUI.
- **Renderer independente da lógica de edição:** a interface só representa o estado.

## 27. Visão de Longo Prazo

Vim motions + Tree-sitter + LSP + custom commands + integração com a TUI: não uma implementação do Vim, mas um **editor de código moderno, modal, extensível e profundamente integrado à aplicação TUI em Go**.
