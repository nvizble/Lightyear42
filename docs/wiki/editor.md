---
title: Editor embutido
order: 6
description: Um editor estilo Vim dentro do lightyear, com cores, erros do compilador e autocomplete.
---

> **Canary:** o editor está na série `v1.3.0-canary`. Instale a canary para usar (veja [Instalação](instalacao.html#canais)).

O lightyear tem um editor de código próprio, com edição modal como no Vim, realce de sintaxe (Tree-sitter) e os erros e o autocomplete de um language server. Ele aparece em três lugares:

- na [aba Exam do app](app.html#exam), com `e`: o subject ao lado da sua entrega;
- no [`lightyear test`](testes.html): o projeto numa árvore, com o botão ▶ Rodar testes;
- sozinho, com o comando experimental `lightyear edit` (escondido do help):

```bash
lightyear edit main.c
lightyear edit ft_split.c libft.h    # vários arquivos, em buffers
lightyear edit --plain notas.txt     # editor sem modos
```

O mouse funciona: clique para posicionar o cursor, arraste para selecionar, role com a roda.

## Modos

| Modo | Como entrar | Para quê |
|------|-------------|----------|
| NORMAL | `esc` | mover, apagar, copiar, rodar comandos |
| INSERT | `i` `a` `o` `I` `A` `O` | digitar texto |
| VISUAL | `v` (caracteres), `V` (linhas), ou arrastando o mouse | selecionar |

Com `--plain`, não há modos: o editor funciona como um editor comum, com `Ctrl-S` salvar, `Ctrl-Z`/`Ctrl-Y` desfazer/refazer e `Ctrl-Q` sair.

## Teclas do modo NORMAL

### Movimentos

| Tecla | Movimento |
|-------|-----------|
| `h` `j` `k` `l` | esquerda, baixo, cima, direita |
| `w` `b` `e` | próxima palavra, palavra anterior, fim da palavra |
| `0` `^` `$` | início da linha, primeiro caractere não branco, fim da linha |
| `W` `B` `E` | o mesmo para WORDs (tudo que não é espaço: `a.b` é uma WORD só) |
| `ge` `gE` | fim da palavra (ou WORD) anterior |
| `0` `Home` | início da linha |
| `$` `End` | fim da linha |
| `g_` | último caractere não branco |
| `N\|` | coluna N (`4\|`) |
| `+` `Enter` `-` | primeiro não branco da linha de baixo / de cima |
| `gg` `G` | início e fim do arquivo; com número, vão à linha N (`42G`) |
| `Ctrl-Home` `Ctrl-End` | início e fim do arquivo |
| `{` `}` | parágrafo anterior / próximo (a linha vazia antes ou depois) |
| `%` | o par do `(` `[` `{` sob o cursor ou adiante na linha; `50%` vai à metade do arquivo |
| `H` `M` `L` | topo, meio e fim da tela |
| `f` `F` `t` `T` + caractere | até o caractere na linha; `;` repete, `,` repete ao contrário |
| `ma` … `` `a `` `'a` | marca a posição em `a` (de `a` a `z`); `` ` `` volta ao lugar exato, `'` à linha (também com operador: `d'a`) |

### Rolar a tela

| Tecla | Ação |
|---|---|
| `Ctrl-d` `Ctrl-u` | meia tela para baixo / para cima (o cursor vai junto) |
| `Ctrl-f` `Ctrl-b`, `PgDn` `PgUp` | uma tela para baixo / para cima |
| `Ctrl-e` `Ctrl-y` | uma linha, sem mover o cursor (só se ele sair da tela) |
| `zz` `zt` `zb` | põe a linha do cursor no meio, no topo ou embaixo da tela |

Contadores multiplicam: `3j`, `10l`, `2fa`.

### Editar

| Tecla | Ação |
|-------|------|
| `x` `X` | apaga o caractere sob o cursor / antes dele |
| `s` `S` | troca o caractere / a linha inteira (entra no INSERT) |
| `r` + caractere | substitui o caractere (`3rx` substitui três) |
| `~` | troca maiúscula e minúscula e anda (`5~`) |
| `J` `gJ` | junta a linha de baixo (`J` com um espaço, `gJ` como está; `3J` junta três) |
| `>>` `<<` | indenta / desindenta a linha com um tab (`>j`, `>}`, `3>>`) |
| `gU` `gu` `g~` + movimento | maiúsculas, minúsculas, troca (`gUw`, `guiw`); `gUU` `guu` `g~~` na linha |
| `Ctrl-a` `Ctrl-x` | soma / subtrai do número sob o cursor ou adiante (`5 Ctrl-a`) |
| `d` + movimento | apaga (`dw`, `d$`, `dG`, `dt;`); `dd` apaga a linha, `D` até o fim |
| `c` + movimento | muda: apaga e entra no INSERT (`cw`, `cb`); `cc` muda a linha mantendo a indentação, `C` até o fim |
| `y` + movimento | copia (`yw`, `y$`); `yy` e `Y` copiam a linha |
| `p` `P` | cola depois / antes; o que `d`, `x`, `c` e `D` apagam também fica para colar (`ddp` troca duas linhas) |
| `u` `Ctrl-r` | desfaz / refaz; uma sessão inteira de INSERT desfaz com um `u` |
| `.` | repete a última mudança (`dw.`, `3.`) |

Contadores valem com operadores: `3dd`, `2d3w`, `3x`, `2u`.

### Text objects

Depois de `d`, `c`, `y` ou no VISUAL: `iw` `aw` (palavra), `i"` `a"` (e as outras aspas), `i(` `a(` `ib`, `i{` `a{` `iB`, `i[` `a[`, `i<` `a<`. Exemplos: `diw`, `ci"`, `da(`, `yi{`, `viw`. O `di{` mantém as chaves nas linhas delas.

### Busca

| Tecla | Ação |
|-------|------|
| `/texto` `?texto` | busca para frente / para trás |
| `n` `N` | próxima / anterior |
| `*` `#` | a palavra sob o cursor, para frente / para trás |
| `:noh` | apaga os destaques |

A busca é por **texto literal**, não por expressão regular.

### Macros

`qa` começa a gravar na macro `a` (a barra mostra "gravando @a"), `q` para, `@a` toca, `@@` repete e `5@a` toca cinco vezes. Vale qualquer letra de `a` a `z`.

### VISUAL

Com a seleção feita: `d` ou `x` apagam, `y` copia, `c` ou `s` mudam, `p` troca pelo que foi copiado e `o` troca a ponta da seleção. `U`, `u` e `~` passam para maiúsculas, minúsculas ou trocam; `>` e `<` indentam as linhas; `J` junta as linhas selecionadas. Movimentos e contadores estendem a seleção (`vjd`, `V3jd`, `v}d`).

### INSERT

Além de digitar: setas, `Home` `End`, `Ctrl-Home` `Ctrl-End` e `PgUp` `PgDn` movem o cursor; `Ctrl-←` `Ctrl-→` andam uma palavra; `Ctrl-w` apaga a palavra antes do cursor e `Ctrl-u` apaga o que foi digitado na linha (até a indentação).

## Comandos

| Comando | Ação |
|---------|------|
| `:w` | salva |
| `:q` | fecha (avisa se há alterações não salvas) |
| `:q!` | sai sem salvar |
| `:wq`, `:x`, `ZZ` | salva e sai |
| `ZQ` | sai sem salvar |
| `:wa` | salva todos os arquivos |
| `:wqa`, `:xa` | salva todos e sai |
| `Ctrl-S`, `Ctrl-Q` | também salvam e saem |

Arquivos só leitura (como o subject na aba Exam) recusam edição com `E21: arquivo só leitura`.

## Buffers e janelas {#buffers}

Cada arquivo aberto é um **buffer**:

| Comando | Ação |
|---------|------|
| `:e arquivo` | abre outro arquivo |
| `:bn` `:bp` | próximo / anterior |
| `:b 2` | vai ao buffer 2 |
| `:ls` | lista os buffers |
| `:bd` | fecha o buffer |

Copiar, colar, macros e busca valem entre os arquivos.

**Janelas** mostram buffers lado a lado:

| Comando | Ação |
|---------|------|
| `:vsp arquivo`, `Ctrl-w v` | divide na vertical (lado a lado) |
| `:sp arquivo` | divide na horizontal (empilhadas) |
| `Ctrl-w w`, `Ctrl-w h/j/k/l` | troca de janela (o clique também) |
| `:close`, `:q` | fecha a janela |
| `:only` | fecha as outras |
| `Ctrl-w >` `<` `+` `-` | aumenta / diminui (aceita contador: `10 Ctrl-w >`) |
| `Ctrl-w =` | iguala os tamanhos |
| `Ctrl-w \|` | maximiza a largura |
| `:vertical resize 50`, `:resize N` | tamanho exato |

Também dá para arrastar a borda `│` entre janelas lado a lado, ou a linha de título nas empilhadas.

## Terminal {#terminal}

`:term` abre um terminal embaixo do código, rodando o seu shell na pasta do arquivo. Dá para compilar, rodar a norminette ou os testes sem sair do editor. `:term make` (ou qualquer comando) roda só aquele comando.

| Tecla | Ação |
|---|---|
| `:term`, `:terminal` | abre o terminal (com ele aberto, só volta para ele) |
| `:term comando` | roda um comando no terminal (`:term make re`, `:term norminette`) |
| `Ctrl-\` `Ctrl-n` | volta ao código (como no Vim) |
| `Ctrl-w k` | também volta ao código; `Ctrl-w j` desce para o terminal |
| clique | no terminal ou no código, troca o foco |

Com o terminal em foco, as teclas vão para o programa, até `Ctrl-c`, as setas e o `Tab`. A barra de status mostra **TERMINAL**. Quando o shell (ou o comando) termina, o título avisa o código de saída e qualquer tecla fecha o painel. Fechar o editor encerra o que estiver rodando.

> **Atenção:** o terminal precisa de Linux ou macOS; no Windows, use o WSL.

## Language server {#lsp}

Com um language server, o editor mostra os erros enquanto você digita, completa nomes e navega pelo código.

| Linguagem | Servidor |
|-----------|----------|
| C, C++ | clangd |
| Go | gopls |
| Python | pyright, pylsp ou ty (o primeiro que encontrar) |
| Rust | rust-analyzer |

Sem um `compile_commands.json`, o clangd usa as flags dos graders da 42: `-Wall -Wextra -Werror` (e `-std=c++98` em C++). A barra de status mostra qual servidor está rodando. Se nenhum estiver instalado, o editor avisa como instalar.

### Downloads automáticos

Nas máquinas da 42 não dá para usar `sudo`, então o lightyear baixa sozinho o que falta, só uma vez, conferindo o sha256:

- **clangd** (Linux x86_64 sem clangd): o clangd oficial, 118 MB, na primeira vez que você abre um arquivo C ou C++. Fica em `~/.cache/42cli` (cerca de 160 MB, com os headers que ele usa). A barra de status avisa enquanto baixa.
- **ty** (Python sem pyright nem pylsp, Linux e macOS): o type checker da Astral, um binário de 14 MB, sem Node nem pip. Aponta erros de tipo e nomes não definidos; hover, `gd` e autocomplete funcionam.

Com o servidor instalado no sistema, nada é baixado.

### O que o language server faz

| Tecla | Ação |
|-------|------|
| mouse sobre um erro | mostra a mensagem inteira do compilador, com as notas |
| `K` | assinatura e documentação do que está sob o cursor, e os erros da linha |
| `gd` | vai à definição (em outro arquivo, abre um buffer; `Ctrl-o` volta) |
| `]d` `[d` | próximo / anterior erro |
| `grn`, `:Rename nome` | renomeia no projeto inteiro |
| `grr` | lista as referências |
| `gra` | mostra as correções sugeridas (o `;` que falta, o `#include`…) |
| `:Format` | formata o arquivo |

Os erros aparecem sublinhados, com um `●` na margem e a contagem na barra de status (`✖ 2 ⚠ 1`). Quando o erro tem correção ("fix available"), passe o mouse e clique na caixa **▸ clique aqui para corrigir**: com uma correção só, ela é aplicada na hora; com várias, aparece a lista.

Os arquivos alterados por `grn` ou `gra` abrem em buffers, para você conferir e salvar com `:wa`.

> **Atenção:** em C e C++, o `:Format` só funciona com um `.clang-format` no projeto. O estilo padrão do clang-format quebraria a norminette.

### Autocomplete

No INSERT, a lista abre sozinha enquanto você digita (e depois de `.`, `->` e `::`), filtrando pelo que já foi digitado. `Ctrl-n` abre na hora; `Ctrl-n` e `Ctrl-p` (ou as setas) escolhem; `Tab` ou `Enter` aceitam; `esc` fecha.

## Temas {#temas}

```text
:colorscheme            lista os temas
:colorscheme dracula    troca o tema
```

Temas: `lightyear` (padrão), `gruvbox`, `dracula`, `nord`, `tokyonight`, `monokai`, `catppuccin`, `onedark`, `solarized`, `github-light` (para terminal claro) e `off` (sem cores).

`Tab` completa o nome (`:colorscheme d` + `Tab`); `Tab` e `Shift-Tab` andam pelas opções. A escolha fica salva no `config.yaml` e vale em todos os lugares onde o editor aparece.

## Limites conhecidos

- A busca é por texto literal (sem regex) e ainda não funciona como movimento de operador (`d/x`).
- Text objects não aceitam contador (`2i(`).
- Não há integração com a área de transferência do sistema.
- No macOS, o primeiro clangd aberto por um binário novo do lightyear pode levar uns 15 segundos (o sistema verifica o programa). A barra mostra "clangd…" até ficar pronto; das próximas vezes é instantâneo.
