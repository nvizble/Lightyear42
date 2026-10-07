# Editor de Código Modal Embutido para TUI em Go

> **Status:** Fase 1 (Editor Core) implementada em `internal/editor` +
> componente em `internal/tui/editorview` (comando experimental escondido
> `lightyear edit <arquivo>`). Próximas: Fase 2 (edição modal) e Fase 3
> (command parser), em `internal/vim`.
>
> **Desvios desta implementação em relação ao texto abaixo:**
> - o componente visual mora em `internal/tui/editorview` (a TUI do projeto
>   fica em `internal/tui`), não em `internal/ui/editor`;
> - `Selection` entra só na Fase 4 (Visual Mode);
> - colunas do `Position` contam runes; a expansão de tabs (tab = 4) é usada
>   no viewport e nos movimentos verticais (coluna "visual" preservada).

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

O editor atua como **cliente LSP** (JSON-RPC) de servidores externos (`gopls`, `rust-analyzer`, `pyright`, `typescript-language-server`). Primeiras funcionalidades: diagnostics, completion, hover, go-to-definition (references depois). Sincronização: `didOpen`, `didChange` (completa no início, incremental depois), `didClose`.

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
| 2 — Edição Modal | NORMAL/INSERT, `hjkl`, `i a o`, `x`, `dd`, `u`, `Ctrl-r` | próxima |
| 3 — Command Parser | count + operator + motion (`3j`, `3dd`, `dw`, `3dw`, `d$`) | |
| 4 — Visual Mode | `v`, `V` e operações sobre seleções | |
| 5 — Registers | `yy`, `dd`, `p`, `P` com register padrão | |
| 6 — Syntax Highlighting | Tree-sitter, highlighting incremental | |
| 7 — LSP | cliente JSON-RPC; diagnostics, hover, completion, go-to-definition | |
| 8 — Avançados | `ciw diw daw`, busca, `f F t T`, `.`, macros, buffers, splits, code actions, rename, format, references | |

## 26. Princípios

- **Editor independente do Vim:** o Editor Core não conhece comandos Vim.
- **Vim independente do LSP:** o Vim Controller não conhece LSP.
- **LSP independente da interface:** o cliente LSP não conhece a TUI.
- **Renderer independente da lógica de edição:** a interface só representa o estado.

## 27. Visão de Longo Prazo

Vim motions + Tree-sitter + LSP + custom commands + integração com a TUI: não uma implementação do Vim, mas um **editor de código moderno, modal, extensível e profundamente integrado à aplicação TUI em Go**.
