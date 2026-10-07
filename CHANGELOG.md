# Changelog

Mudanças visíveis do lightyear, da mais nova para a mais antiga. O
`lightyear --version` mostra a seção da versão instalada (e
`lightyear version --changelog`, este arquivo inteiro).

Versões `-canary.N` são pre-releases: `lightyear update --canary` entra no
canal canary e `lightyear update --stable` volta para a estável.

## [1.3.0-canary.26] — 2026-10-07

- Editor: temas de cores para o código — `:colorscheme` lista, e
  `:colorscheme dracula` troca na hora: lightyear (padrão), gruvbox,
  dracula, nord, tokyonight, monokai, catppuccin, onedark, solarized,
  github-light (terminal claro) e off (sem cores). A escolha fica salva no
  `config.yaml` e vale também no editor da aba Exam.

## [1.3.0-canary.25] — 2026-10-07

- Editor: aceite o "fix available" com o mouse — passe o mouse no erro e
  clique na caixa (▸ clique aqui para corrigir); com uma correção só, ela é
  aplicada na hora, com várias aparece a lista. Pelo teclado, `gra` na
  linha do erro.

## [1.3.0-canary.24] — 2026-10-07

- Editor: janelas redimensionáveis — arraste a borda `│` entre o subject e
  o código (ou a linha de título, nas empilhadas), ou use `Ctrl-w >`/`<`
  (com contador: `10 Ctrl-w >`), `Ctrl-w +`/`-`, `Ctrl-w =` para igualar,
  `Ctrl-w |` para maximizar e `:vertical resize 50`. Na aba Exam, o editor
  reabre com o tamanho que você deixou.
- Editor: quando o erro tem correção ("fix available"), a caixa do mouse e
  o `K` dizem como aplicar: com o cursor na linha, `gra` e Enter.

## [1.3.0-canary.23] — 2026-10-07

- Editor: passe o mouse sobre um erro sublinhado (ou sobre o `●` na
  margem) para ler a mensagem inteira do compilador, com as notas. O `K`
  também mostra os erros da linha, junto com a documentação.

## [1.3.0-canary.22] — 2026-10-07

- App, aba Exam: `e` abre o editor do lightyear dentro do app — o subject
  (só leitura) à esquerda e a sua entrega à direita, com os erros do
  compilador (as flags do grader) e autocomplete. `Ctrl-w w` alterna,
  `:wq` salva e volta para o app; `E` continua abrindo o seu `$EDITOR`.
- Editor: arquivos só leitura (`E21` ao tentar editar), `:wqa`/`:xa`.

## [1.3.0-canary.21] — 2026-10-07

- Editor: `grn` renomeia no projeto inteiro (`:Rename nome`), `grr` lista
  as referências, `gra` mostra as correções do language server (o `;` que
  falta, o `#include`…) e `:Format` formata. Arquivos alterados abrem em
  buffers para conferir e salvar com `:wa`.
- Formatar C/C++ exige um `.clang-format` no projeto: o estilo padrão
  quebraria a norminette.

## [1.3.0-canary.20] — 2026-10-07

- Editor: janelas — `:vsp ft.h` (ou `Ctrl-w v`) põe dois arquivos lado a
  lado, `:sp` empilha; `Ctrl-w w`/`h`/`j`/`k`/`l` trocam de janela (o
  clique também), `:close` e `:only` fecham, e `:q` fecha a janela.

## [1.3.0-canary.19] — 2026-10-07

- Editor: vários arquivos — `lightyear edit a.c b.c`, `:e arquivo`,
  `:bn`/`:bp`/`:b 2`, `:ls`, `:bd`, `:wa`; o `:q` avisa se outro arquivo
  tem alterações. O `gd` abre a definição em outro arquivo e `Ctrl-o`
  volta. Copiar, colar, macros e busca valem entre os arquivos.
- Editor: um language server por projeto, compartilhado pelos arquivos
  abertos (antes era um por arquivo).

## [1.3.0-canary.18] — 2026-10-07

- Editor: `.` repete a última mudança — `dw.`, `ciwfoo<esc>` e `.` em outra
  palavra, `A;<esc>j.`; com contador (`3.`).
- Editor: macros — `qa` começa a gravar (a barra mostra "gravando @a"), `q`
  para, `@a` toca, `@@` repete e `5@a` toca cinco vezes.

## [1.3.0-canary.17] — 2026-10-07

- Editor: text objects — `diw`, `daw`, `ci"`, `da(`, `di{` (mantém as
  chaves nas linhas delas), `yi[`, também no Visual (`viw`, `vi(`).
- Editor: `f`, `F`, `t`, `T` com `;` e `,` (`dt;`, `cf)`, `2fa`).
- Editor: busca com `/` e `?`, `n`/`N`, `*`/`#` (a palavra sob o cursor);
  as ocorrências ficam destacadas até `:noh`.

## [1.3.0-canary.16] — 2026-10-07

- Editor: autocomplete do language server — a lista abre enquanto você
  digita (e depois de `.`, `->`, `::`, ou com `Ctrl-n`), filtra pelo que já
  foi digitado; `Ctrl-n`/`Ctrl-p` escolhem e `Tab`/`Enter` aceitam.
- Editor: `K` mostra a assinatura/documentação do que está sob o cursor,
  `gd` vai à definição e `]d`/`[d` pulam entre os erros.

## [1.3.0-canary.15] — 2026-10-07

- Editor: erros e avisos do language server — clangd (C/C++, com
  `-Wall -Wextra -Werror` como no grader), gopls, pyright e rust-analyzer.
  O trecho fica sublinhado, a margem ganha um `●` e a barra de status mostra
  a mensagem da linha do cursor e a contagem (`✖ 2 ⚠ 1`). Sem o servidor
  instalado, o editor avisa como instalar.

## [1.3.0-canary.14] — 2026-10-07

- Editor: syntax highlighting com Tree-sitter para C, C++, Go, Python e
  Rust — palavras-chave, strings, comentários, números, funções e tipos.
  As cores acompanham cada tecla, sem reprocessar o arquivo inteiro.
- Instalar via `go install` (canary) agora pede um compilador C (`cc`);
  os binários dos releases continuam prontos para usar.

## [1.3.0-canary.13] — 2026-10-07

- Editor: copiar e colar como no Vim — `y` copia (`yy`, `yw`, `y$`, `Y`),
  `p`/`P` colam depois/antes (`3p`), e o que `d`, `x` e `D` apagam também
  fica para colar (`ddp` troca duas linhas, `xp` dois caracteres).
- Editor: `c` muda o texto — `cw`, `cc` (mantém a indentação), `c$`/`C`,
  `cb`, `2cw` — e um `u` desfaz a mudança inteira.
- Editor, modo Visual: `y` copia, `c` muda e `p` troca a seleção pelo que
  foi copiado.

## [1.3.0-canary.12] — 2026-10-07

- Editor: modo Visual — `v` seleciona caracteres e `V` linhas inteiras
  (com motions e contadores: `vjd`, `V3jd`, `vwd`), `o` troca a ponta e
  `d`/`x` apagam a seleção. Arrastar o mouse também seleciona.
- Editor: o `u` volta o cursor para o início do trecho desfeito, como no Vim.

## [1.3.0-canary.11] — 2026-10-07

- `lightyear --version` (e `lightyear version`) mostra as novidades da
  versão instalada; `lightyear version --changelog` mostra o histórico
  inteiro. O changelog vai embutido no binário, funciona offline.

## [1.3.0-canary.10] — 2026-10-07

- Editor (`lightyear edit`, experimental): contadores e operador × motion,
  como no Vim — `3j`, `5k`, `10l`, `dw`, `3dw`, `2d3w`, `de`, `db`, `d$`,
  `d0`, `d^`, `dj`, `dG`, `dgg`, `3dd`, `3x`, `2u`.
- Novos movimentos: `w b e` (palavras), `0 ^ $` (linha), `gg G` (com
  contador, vão à linha N).

## [1.3.0-canary.9] — 2026-10-07

- Editor: edição modal estilo Vim — NORMAL e INSERT, `hjkl`, `i a o I A O`,
  `x`, `dd`, `u`, `Ctrl-r` e `:w`, `:q`, `:q!`, `:wq`, `:x`. Uma sessão
  inteira de INSERT desfaz com um `u`. `--plain` abre o editor sem modos.

## [1.3.0-canary.8] — 2026-10-07

- Editor embutido (fase 1), experimental: `lightyear edit <arquivo>`, com
  números de linha, Ctrl-S/Ctrl-Z/Ctrl-Y, clique para posicionar o cursor
  e Ctrl-Q com confirmação quando há alterações não salvas.

## [1.3.0-canary.7] — 2026-10-07

- App: nova aba **Subjects** — seus projetos no topo e os 241 do catálogo;
  `/` busca, e o clique (ou a sugestão escolhida) baixa e abre o PDF.

## [1.3.0-canary.6] — 2026-10-07

- App, aba Campus: `/` busca uma pessoa, com sugestões enquanto você digita;
  o posto dela fica em destaque e a tela rola até ele.
- Corrigido: em terminais estreitos, todas as linhas das abas ganhavam
  "…" no fim.

## [1.3.0-canary.5] — 2026-10-07

- App com o visual do showreel: Início enxuto, avaliações em linhas
  ("hoje 14:30 · você avalia · em 1h18"), campus em grade de postos
  coloridos e grademe com spinner, checklist dos testes, banner
  SUCCESS/FAILURE e os níveis.
- Campus: passe o mouse (ou clique) num posto para ver quem está lá.
- Corrigido: o cartão da prova aparecia duplicado depois do grademe.

## [1.3.0-canary.4] — 2026-10-06

- App, aba Exam: o vim abre com o cursor no código e o subject só leitura
  (`Ctrl-w w` alterna entre os dois).

## [1.2.1] — 2026-10-06

- `lightyear update --canary` e `--stable`: escolha o canal de versões.
- Corrigido: no macOS, o primeiro teste do `lightyear exam grademe` podia
  reprovar com "timeout" falso.

## [1.3.0-canary.3] — 2026-10-06

- `lightyear update --canary` e `--stable`; quem está na canary continua
  recebendo canaries.

## [1.3.0-canary.2] — 2026-10-06

- App, aba Exam: `e` abre o subject e a sua entrega lado a lado no vim.

## [1.3.0-canary.1] — 2026-10-06

- `lightyear` sozinho abre o app em tela cheia: abas clicáveis (Início,
  Avaliações, Projetos, Campus, Slots, Exam), mouse e atalhos.
- Corrigido: no macOS, o primeiro teste do grademe podia dar timeout falso.

## [1.2.0] — 2026-10-06

- `lightyear exam`: simulador de provas offline com os 56 exercícios do
  Exam Rank 02, enunciados em português e inglês, correção com
  `cc -Wall -Wextra -Werror` e modo prática.
- `lightyear notify`: push no celular quando surge uma avaliação nova.
- `lightyear evaluations --open`: abre a avaliação na Intra.

## [1.1.3] — 2026-07-20

- Campus: o mapa desenha os postos espelhados (pN … p1) em todos os
  clusters; `natural_posts: true` volta para p1 … pN.

## [1.1.2] — 2026-07-20

- Corrigido: mapa do campus em clusters com numeração física invertida.

## [1.1.1] — 2026-07-18

- O `lightyear setup` instala o autocomplete do shell, e o
  `lightyear subject <TAB>` completa os projetos.

## [1.1.0] — 2026-07-18

- `lightyear update`: atualiza o binário pelo GitHub Releases.
- `lightyear subject`: baixa e abre o PDF do subject (catálogo embutido,
  `set-id` e `import`).

## [1.0.2] — 2026-07-17

- Pacote `.deb` para Ubuntu/Debian.

## [1.0.1] — 2026-07-17

- `go install` gera o binário `lightyear`.

## [1.0.0] — 2026-07-17

- `lightyear setup`: guia para criar a aplicação OAuth na Intra.

## [0.1.0] — 2026-07-17

- Primeira versão: login/logout (OAuth2, token no keyring), `me`,
  `profile`, `search`, `projects`, `evaluations`, `slots`, `campus`,
  `friends` e o `dashboard` em tempo real.
