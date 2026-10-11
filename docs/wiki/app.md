---
title: App em tela cheia
order: 5
description: Rode só lightyear para ter perfil, avaliações, campus e o simulador numa tela clicável.
---

> **Canary:** o app chegou na `v1.3.0-canary.1`. Instale a canary para usar (veja [Instalação](instalacao.html#canais)).

```bash
lightyear
```

Sem subcomando, num terminal, o lightyear abre um app em tela cheia com abas. Cada aba usa os mesmos dados do comando equivalente: o que aparece no `lightyear evaluations` aparece na aba Avaliações, e assim por diante. Os subcomandos continuam funcionando igual.

Fora de um terminal (em pipes e scripts), o `lightyear` sozinho mostra o help.

## Abas

| # | Aba | O que mostra |
|---|-----|--------------|
| 1 | Início | seu campus e quantos estão online, seu nível, wallet e pontos de avaliação, a ocupação por cluster, as 3 próximas avaliações e os amigos online |
| 2 | Avaliações | as próximas avaliações, uma por linha ("hoje 14:30 · você avalia · em 1h18") |
| 3 | Projetos | seus projetos do cursus principal, com status e nota |
| 4 | Subjects | seus projetos no topo e os do catálogo embaixo; o clique baixa e abre o PDF |
| 5 | Campus | os postos do seu campus numa grade colorida |
| 6 | Slots | o calendário dos seus slots e a lista deles |
| 7 | Exam | o [simulador de provas](simulador.html): a prova em andamento ou a lista de exercícios |

Sem login, o app abre do mesmo jeito: as abas da API explicam como entrar (`lightyear setup` e `lightyear login`) e a aba Exam, que é offline, funciona normalmente.

## Teclado

| Tecla | Ação |
|-------|------|
| `1` a `7` | vai direto para a aba |
| `←` `→`, `h` `l`, `Tab` `Shift-Tab` | aba anterior / próxima |
| `↑` `↓`, `k` `j` | rola uma linha |
| `PgUp` `PgDn`, `Espaço` | rola uma página |
| `Home` | volta ao topo |
| `r` | atualiza a aba |
| `/` | busca (nas abas Campus, Subjects e Exam) |
| `esc` | limpa a busca |
| `q`, `Ctrl-c` | sai |

## Mouse

- Clique numa aba para abri-la, e nos botões do rodapé para as ações (`r atualizar`, `q sair`, `/ buscar`…).
- A roda do mouse rola o conteúdo.
- **Campus:** passe o mouse (ou clique) num posto para ver quem está lá.
- **Subjects:** clique num projeto para baixar e abrir o PDF do subject.
- **Exam:** passe o mouse sobre um exercício para ver o nível; clique para treinar só ele, sem tempo.

## Busca

Nas abas Campus, Subjects e Exam, `/` abre a busca, com sugestões enquanto você digita:

- **Campus:** busca uma pessoa online; o posto dela fica em destaque e a tela rola até ele.
- **Subjects:** busca um projeto; escolher a sugestão baixa e abre o PDF.
- **Exam:** busca um exercício pelo nome.

Dentro da busca, `↑` `↓` escolhem, `Enter` seleciona e `esc` cancela.

## Aba Exam {#exam}

A aba Exam é o [simulador de provas](simulador.html) dentro do app. Ela começa sempre uma prova do Exam Rank 02, com 3 horas.

| Tecla | Ação |
|-------|------|
| `s` | começa a prova |
| `e` | abre o subject e a sua entrega no [editor embutido](editor.html) |
| `E` | abre o subject e a sua entrega no seu `$EDITOR` |
| `g` | corrige (grademe), com spinner, checklist dos testes e o banner SUCCESS/FAILURE |
| `f` | encerra a prova (aperte `f` duas vezes para confirmar) |
| `r` | recarrega o estado da prova |

### e: o editor do lightyear

O `e` abre o editor embutido sobre o app: o subject (só leitura) à esquerda e o arquivo da entrega à direita, com o cursor já no código. O language server (clangd) aponta os erros com as mesmas flags do grader (`-Wall -Wextra -Werror`) e completa enquanto você digita.

- `Ctrl-w w` alterna entre o subject e o código.
- Arraste a borda `│` entre os dois (ou use `Ctrl-w <` e `Ctrl-w >`) para mudar a largura; o app lembra o tamanho na próxima vez.
- `:wq` salva e volta para o app.

### E: o seu editor

O `E` usa `$VISUAL` ou `$EDITOR` (com `vim` como padrão). Com vim, nvim ou vi, o subject abre à esquerda, travado como só leitura, e o código à direita, com o cursor nele. Saia do editor para voltar ao app.

## Notificações dentro do app {#notify}

Com o [`lightyear notify`](notificacoes.html) configurado, o app vigia a sua agenda enquanto estiver aberto, como o `notify watch`: checa a cada 2 minutos e manda as avaliações novas para o celular. A barra de abas mostra **notify ligado** e o rodapé diz o que foi avisado.
