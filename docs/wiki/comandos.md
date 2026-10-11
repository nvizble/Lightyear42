---
title: Referência de comandos
order: 4
description: Todos os comandos do lightyear, com as flags de cada um.
---

Esta página resume o `--help` de cada comando. A ajuda completa está sempre no terminal: `lightyear <comando> --help`.

| Grupo | Comandos |
|-------|----------|
| [Conta](#conta) | [setup](#setup), [login](#login), [logout](#logout) |
| [Perfil e busca](#perfil-e-busca) | [me](#me), [profile](#profile), [search](#search), [projects](#projects) |
| [Projetos e subjects](#projetos-e-subjects) | [subject](#subject) |
| [Avaliações e slots](#avaliacoes-e-slots) | [evaluations](#evaluations), [slots](#slots) |
| [Campus e amigos](#campus-e-amigos) | [campus](#campus), [friends](#friends), [dashboard](#dashboard) |
| [Notificações](#notificacoes) | [notify](#notify) |
| [Simulador](#simulador) | [exam](#exam) |
| [Testes](#testes) | [test](#test) |
| [Editor](#editor) | [edit](#edit) |
| [Manutenção](#manutencao) | [update](#update), [version](#version), [config](#config), [cache](#cache), [completion](#completion) |

## lightyear {#lightyear}

```bash
lightyear              # abre o app em tela cheia (canary)
lightyear --version    # versão e novidades dela (igual a lightyear version)
lightyear help <cmd>   # ajuda de um comando
```

> **Canary:** o `lightyear` sem argumentos abre o [app em tela cheia](app.html). Na estável, e fora de um terminal, ele mostra o help.

## Conta {#conta}

### setup {#setup}

Assistente para criar a aplicação OAuth na Intra e gravar o UID e o Secret no `config.yaml`. No fim, instala o autocomplete do shell. Detalhes em [Primeiros passos](primeiros-passos.html#setup).

| Flag | Descrição |
|------|-----------|
| `--client-id string` | UID da aplicação OAuth (pula o prompt) |
| `--client-secret string` | Secret da aplicação OAuth (pula o prompt) |
| `--force` | sobrescreve credenciais existentes sem perguntar |
| `--no-browser` | não abre o navegador automaticamente |
| `--no-completion` | não instala o autocomplete do shell |
| `--shell string` | shell do autocomplete (zsh, bash, fish); padrão: `$SHELL` |

### login {#login}

Abre o navegador na autorização da Intra e espera o retorno local. O token fica no keyring do sistema (Keychain, Secret Service ou Credential Manager).

| Flag | Descrição |
|------|-----------|
| `--timeout duration` | tempo máximo de espera pela autorização (padrão `3m0s`) |

### logout {#logout}

Remove o token do keyring. Não revoga a autorização na Intra.

## Perfil e busca {#perfil-e-busca}

### me {#me}

Seu perfil: nível, wallet, pontos de avaliação, campus e localização.

```bash
lightyear me
```

### profile {#profile}

O perfil público de qualquer pessoa da 42 Network, pelo login da Intra.

```bash
lightyear profile <login>
```

### search {#search}

Lista usuários cujo login começa com o termo.

```bash
lightyear search jdi        # logins que começam com "jdi"
lightyear search jdi -n 30  # até 30 resultados
```

| Flag | Descrição |
|------|-----------|
| `-n, --limit int` | número máximo de resultados, até 100 (padrão 10) |

### projects {#projects}

Seus projetos (ou os de outro login), com status e nota. Mostra só o cursus principal por padrão.

```bash
lightyear projects
lightyear projects <login> --all
```

| Flag | Descrição |
|------|-----------|
| `-a, --all` | inclui todos os cursus (piscine, eventos…) |

## Projetos e subjects {#projetos-e-subjects}

### subject {#subject}

Baixa o PDF do subject de um projeto e abre no visualizador padrão. Exige login.

```bash
lightyear subject push_swap
lightyear subject push_swap --lang fr
lightyear subject push_swap --pdf-id 193464
```

A API pública não entrega o PDF (responde 403), então o lightyear descobre o id do PDF no CDN da Intra nesta ordem:

1. `--pdf-id <n>`, que também grava o id no índice local;
2. o índice local (`~/.local/share/42cli/subjects/index.json`), preenchido na primeira vez com o catálogo embutido (cerca de 240 projetos);
3. o catálogo embutido no binário;
4. a página do projeto na Intra, quando acessível.

| Flag | Descrição |
|------|-----------|
| `--lang string` | idioma do subject: en, fr… (padrão `en`) |
| `--pdf-id int` | id numérico do PDF no CDN; grava no índice local |
| `--force` | baixa de novo, mesmo com o PDF em cache |
| `--no-open` | só baixa e imprime o caminho, sem abrir |
| `--http-debug` | imprime método, URL e status de cada pedido HTTP (stderr) |

#### subject set-id {#subject-set-id}

Grava ou atualiza o id do PDF de um projeto no índice local, sem baixar. O id é o número na URL `https://cdn.intra.42.fr/pdf/pdf/<id>/en.subject.pdf`.

```bash
lightyear subject set-id push_swap 193464
```

Aceita `--http-debug`.

#### subject import {#subject-import}

Faz merge de um JSON `{"slug": id, ...}` no índice local. Só é preciso para atualizar com um catálogo novo.

```bash
lightyear subject import ./catalog.json
```

## Avaliações e slots {#avaliacoes-e-slots}

### evaluations {#evaluations}

Suas próximas avaliações, como avaliador ou avaliado, da mais próxima para a mais distante. Alias: `evals`.

| Flag | Descrição |
|------|-----------|
| `--open` | abre no navegador a próxima avaliação em que você avalia, para iniciar ou preencher (prioriza as que já começaram) |

### slots {#slots}

Lista, abre e fecha os horários em que você se declara disponível para avaliar. Sem subcomando, lista.

> **Atenção:** exige o scope **projects** na sua aplicação da Intra e um login feito depois de ativá-lo: `lightyear logout && lightyear login`.

#### slots list {#slots-list}

Lista seus slots futuros.

#### slots open {#slots-open}

Cria slots de 15 minutos cobrindo o intervalo pedido.

```bash
lightyear slots open --duration 1h                                   # o mais cedo possível (~30 min, grade de 15 min)
lightyear slots open --from "2026-07-18 14:00" --duration 1h
lightyear slots open --from "2026-07-18 14:00" --to "2026-07-18 15:00"
```

| Flag | Descrição |
|------|-----------|
| `--duration string` | duração (ex.: `30m`, `1h`); sozinha, começa o mais cedo possível |
| `--from string` | início, na hora local (`YYYY-MM-DD HH:MM`) |
| `--to string` | fim, na hora local; exige `--from`; use `--to` ou `--duration` |

#### slots close {#slots-close}

Remove slots futuros livres (sem avaliação marcada).

```bash
lightyear slots close 12345
lightyear slots close --all
```

| Flag | Descrição |
|------|-----------|
| `--all` | fecha todos os slots futuros livres |

## Campus e amigos {#campus-e-amigos}

### campus {#campus}

Quem está logado no campus, num mapa por cluster, fileira e posto (ex.: `c1r2p3`). Usa o seu campus primário por padrão. A planta dos clusters pode ser configurada (veja [Primeiros passos](primeiros-passos.html#configuracao)).

| Flag | Descrição |
|------|-----------|
| `--id int` | ID de outro campus |
| `-f, --friends` | mostra só os logins da sua lista de amigos |

### friends {#friends}

Uma lista local de logins favoritos, guardada no `config.yaml` (a API da 42 não tem amizades). Alimenta o `campus --friends` e o `friends online`.

```bash
lightyear friends add <login>
lightyear friends remove <login>
lightyear friends list
lightyear friends online          # quais amigos estão online e em que posto
lightyear friends online --id 20  # em outro campus
```

| Subcomando | Flags |
|------------|-------|
| `add <login>` | — |
| `remove <login>` | — |
| `list` | — |
| `online` | `--id int`: ID do campus (padrão: o seu campus primário) |

### dashboard {#dashboard}

Uma TUI em tempo real com seu perfil, a ocupação dos clusters, as próximas avaliações, o calendário de slots e os amigos online. Teclas: `r` atualiza na hora, `q` ou `esc` sai.

| Flag | Descrição |
|------|-----------|
| `--id int` | ID do campus (padrão: o seu campus primário) |
| `--interval duration` | intervalo da atualização automática, mínimo 15s (padrão `1m0s`) |

## Notificações {#notificacoes}

### notify {#notify}

Push no celular quando aparece uma avaliação nova na sua agenda, via [ntfy](https://ntfy.sh). Guia completo em [Notificações no celular](notificacoes.html).

| Subcomando | O que faz | Flags |
|------------|-----------|-------|
| `setup [tópico]` | grava servidor e tópico no `config.yaml`; sem argumento, mantém o tópico atual ou gera um aleatório | `--server string` (padrão `https://ntfy.sh`), `--token string` (tópicos protegidos) |
| `test` | envia uma notificação de teste | — |
| `check` | checa uma vez, notifica o que é novo e sai (para cron) | — |
| `watch` | checa a cada intervalo até `Ctrl+C` | `--interval duration` (mínimo `1m0s`, padrão `2m0s`) |

## Simulador {#simulador}

### exam {#exam}

Simula as provas da 42 offline, com correção local. Guia completo em [Simulador de provas](simulador.html).

| Subcomando | O que faz | Flags |
|------------|-----------|-------|
| `start` | começa uma prova cronometrada no nível 1, com exercício sorteado | `--rank string` (padrão `02`), `--time duration` (padrão `3h0m0s`) |
| `practice <exercício>` | treina um exercício, sem tempo | — |
| `grademe` | corrige o exercício atual | — |
| `status` | mostra exercício, nota e tempo restante | — |
| `finish` | encerra a prova (seu código fica guardado) | — |
| `list` | lista os exercícios por rank e nível | — |

## Testes {#testes}

### test {#test}

> **Canary:** disponível a partir da `v1.3.0-canary.32`.

Testa o seu projeto com centenas de casos tirados do subject. Rode na raiz do projeto. Guia completo em [Testes de projeto](testes.html).

```bash
cd ~/libft && lightyear test libft              # editor com árvore e ▶ Rodar testes
lightyear test libft --run                      # sem editor, só o resultado
lightyear test libft --run --only ft_split      # só uma função
sudo lightyear test born2beroot                 # dentro da VM avaliada
```

Projetos: `libft`, `ft_printf`, `get_next_line`, `python-module-00` a `python-module-07`, `python-module-09`, `python-module-10`, `python-module-05-bis`, `python-module-06-bis` e `born2beroot`.

| Flag | Descrição |
|------|-----------|
| `--run` | roda direto no terminal, sem abrir o editor; sai com erro se algum caso falhar |
| `--all` | com `--run`, lista também os casos que passaram |
| `--only strings` | testa só essas funções da libft (ex.: `--only ft_split,ft_itoa`) |

## Editor {#editor}

### edit {#edit}

> **Canary:** comando experimental, escondido do `lightyear --help`.

Abre o [editor embutido](editor.html), estilo Vim, com um ou mais arquivos.

```bash
lightyear edit main.c
lightyear edit ft_split.c libft.h
lightyear edit --plain notas.txt
```

| Flag | Descrição |
|------|-----------|
| `--plain` | editor sem modos (sem os comandos estilo Vim) |

## Manutenção {#manutencao}

### update {#update}

Baixa a versão mais nova do GitHub Releases e substitui o binário atual. Não usa Go. Veja [Instalação](instalacao.html#atualizar).

| Flag | Descrição |
|------|-----------|
| `--check` | só verifica se há versão nova |
| `-y, --yes` | não pede confirmação |
| `--canary` | entra no canal canary (pre-releases) |
| `--stable` | volta para o canal estável (instala a última estável, mesmo que mais antiga) |
| `--force` | permite atualizar builds sem versão de release (ex.: `dev`) |

### version {#version}

Versão, commit, data de build e o que mudou nesta versão. `lightyear --version` faz o mesmo.

| Flag | Descrição |
|------|-----------|
| `--changelog` | mostra o changelog inteiro |

### config {#config}

| Subcomando | O que faz |
|------------|-----------|
| `config path` | imprime o caminho do `config.yaml` |
| `config show` | mostra a configuração efetiva (arquivo + ambiente + padrões), com o `client_secret` mascarado, e as pastas de cache e dados |

### cache {#cache}

O lightyear guarda as respostas da API num cache SQLite com validade, para economizar o rate limit da Intra.

```bash
lightyear cache clear   # apaga o cache; a próxima consulta busca dados frescos
```

### completion {#completion}

| Subcomando | O que faz |
|------------|-----------|
| `completion install [--shell zsh\|bash\|fish]` | gera o script e configura o rc do shell (padrão: `$SHELL`) |
| `completion bash` | imprime o script para bash (precisa do pacote `bash-completion`) |
| `completion zsh` | imprime o script para zsh |
| `completion fish` | imprime o script para fish |
| `completion powershell` | imprime o script para PowerShell |

Os geradores aceitam `--no-descriptions`, que tira as descrições das sugestões.
