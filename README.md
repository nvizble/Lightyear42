# lightyear

CLI moderna, open source, para a [42 Network](https://www.42network.org/), inspirada em ferramentas como `gh`, `docker` e `kubectl`.

> Binário/comando: **`lightyear`** (antes `42`).

**Site:** [nvizble.github.io/Lightyear42](https://nvizble.github.io/Lightyear42/)

> Status: **v1.2.0**: simulador de provas (`lightyear exam`), notificações no celular (`lightyear notify`) e `evaluations --open`.
>
> Nota: dados de exames da Intra seguem fora (a API retorna 403 com scope `public`); o `lightyear exam` é um simulador offline, sem API.

## Requisitos

- Go 1.25+ e um compilador C (`gcc` ou `clang`) — só para instalar via `go install` ou desenvolver (o editor usa Tree-sitter, que é C)
- Uma aplicação OAuth registrada na Intra: [profile.intra.42.fr/oauth/applications](https://profile.intra.42.fr/oauth/applications/new) com Redirect URI `http://127.0.0.1:53682/callback`
- Para `lightyear slots open/close`, ative o scope **projects** na app e rode `lightyear logout && lightyear login`

## Instalação

> **Campus 42:** o Go do campus costuma ser **1.23**, enquanto o lightyear exige **1.25+**.
> Prefira o binário do Release ou o `.deb` — não depende do Go instalado.

### Ubuntu / Debian (`.deb`)

```bash
# amd64 (x86_64) — ajuste a versão/arch se necessário
VER=1.2.0
curl -sLO "https://github.com/nvizble/Lightyear42/releases/download/v${VER}/lightyear_${VER}_linux_amd64.deb"
sudo apt install "./lightyear_${VER}_linux_amd64.deb"
lightyear version
```
(Para ARM64: `linux_arm64.deb`. Sem `sudo`, use o tarball em `~/.local/bin` — veja abaixo.)

Sem sudo
```bash
# amd64 (x86_64) — ajuste a versão/arch se necessário
VER=1.2.0
mkdir -p ~/.local/bin
curl -sL "https://github.com/nvizble/Lightyear42/releases/download/v${VER}/lightyear_${VER}_Linux_x86_64.tar.gz" \
  | tar -xz -C ~/.local/bin lightyear
# garanta que ~/.local/bin está no PATH
lightyear version
```

### Binário (macOS / Linux / Windows)

Baixe o release em [GitHub Releases](https://github.com/nvizble/Lightyear42/releases) ou:

```bash
# exemplo macOS Apple Silicon
VER=1.2.0
curl -sL "https://github.com/nvizble/Lightyear42/releases/download/v${VER}/lightyear_${VER}_Darwin_arm64.tar.gz" \
  | tar -xz lightyear
sudo mv lightyear /usr/local/bin/
lightyear version
```

(Ajuste `Darwin_arm64` conforme o SO: `Darwin_x86_64`, `Linux_x86_64`, `Linux_arm64`, `Windows_x86_64`.)

### Via Go (requer Go 1.25+ e um compilador C)

```bash
go install github.com/nvizble/Lightyear42/cmd/lightyear@latest
```

Isso instala o binário `lightyear` em `$(go env GOPATH)/bin` (no Windows: `%USERPROFILE%\go\bin`). Esse diretório precisa estar no `PATH`.

### Desenvolvimento

```bash
git clone https://github.com/nvizble/Lightyear42.git
cd Lightyear42
make install   # ~/go/bin/lightyear
lightyear --help
```

Para só compilar no diretório do projeto: `make build` → `./lightyear`.

### Atualizar

```bash
lightyear update --check   # só verifica
lightyear update           # baixa e substitui o binário atual
lightyear update -y        # sem confirmação
lightyear update --canary  # entra no canal canary (pre-releases: novidades antes)
lightyear update --stable  # volta para a última versão estável
```

**Canais:** o estável é o padrão. As canaries (`v1.3.0-canary.N`) trazem as
novidades antes, menos testadas. Quem está numa canary continua recebendo
canaries no `lightyear update`; o `--stable` volta para a estável, mesmo que
ela seja mais antiga que a canary instalada.

Requer permissão de escrita no caminho do executável (ex.: `~/.local/bin`).
Instalações via `.deb` em `/usr/bin` pedem `sudo` ou reinstale o `.deb` novo.

## Uso

Rode só `lightyear` para abrir o **app em tela cheia** (canary, a partir da v1.3.0-canary.1):
abas clicáveis (Início, Avaliações, Projetos, Subjects, Campus, Slots, Exam),
rolagem com o mouse e ações no rodapé. Teclado: `1`–`7`/`←→` trocam de aba, `↑↓`/`PgUp`/`PgDn`
rolam, `r` atualiza, `q` sai; na aba Exam, `s` começa, `e` abre o subject e a
sua entrega lado a lado no editor do lightyear (estilo Vim, com os erros do
compilador e autocomplete), `E` abre no seu `$EDITOR`, `g` corrige e `f`
encerra. O cursor já começa no código e o subject fica só leitura; `Ctrl-w w`
alterna entre as duas janelas e `:wq` salva e volta para o app.
Na aba Campus, passe o mouse num posto para ver quem está lá; `/` busca uma
pessoa (com sugestões) e destaca o posto. Na aba Subjects, `/` busca um projeto
e o clique (ou a sugestão escolhida) baixa e abre o PDF do subject.
Fora de um terminal (pipes, scripts) o `lightyear` sozinho continua mostrando o help.

```bash
lightyear                  # app em tela cheia (TUI clicável)
lightyear setup            # guia OAuth na Intra + grava UID/Secret
lightyear login            # autentica via OAuth2 (abre o navegador)
lightyear logout           # remove o token do keyring
lightyear me               # seu perfil: nível, wallet, pontos, campus
lightyear profile <login>  # perfil de qualquer usuário da 42
lightyear search <termo>   # busca usuários por prefixo de login (-n limita)
lightyear projects [login] # projetos com status e nota (--all inclui piscine)
lightyear subject <proj>   # baixa e abre o PDF do subject (CDN + catálogo embutido)
lightyear subject set-id <proj> <id>  # atualiza o pdf-id no índice local
lightyear subject import <f.json>     # merge de um JSON externo no índice local
lightyear evaluations      # próximas avaliações agendadas (alias: evals)
lightyear evaluations --open # abre a avaliação atual na Intra (como avaliador)
lightyear notify setup     # configura o push no celular (ntfy) e mostra como assinar
lightyear notify test      # envia uma notificação de teste
lightyear notify watch     # monitora e avisa a cada avaliação nova (--interval)
lightyear notify check     # checa uma vez e sai (para cron / systemd timer)
lightyear slots            # lista slots futuros de disponibilidade
lightyear slots open --duration 1h   # abre a partir do momento mais cedo (~30min)
lightyear slots open --from "..." --to "..."  # ou --from + --duration
lightyear slots close <id> # fecha um slot livre
lightyear slots close --all # fecha todos os slots livres
lightyear campus           # mapa de quem está online no campus (--id p/ outro)
lightyear campus --friends # mapa filtrado pela sua lista de amigos
lightyear friends add <l>  # gerencia a lista local de amigos (add/remove/list)
lightyear friends online   # quais amigos estão online e em qual posto
lightyear dashboard        # TUI: perfil, ocupação, avaliações, calendário de slots, amigos
lightyear exam start       # simulador de prova (Exam Rank 02), cronometrado, offline
lightyear exam practice <ex>  # treina um exercício específico, sem tempo
lightyear exam grademe     # corrige o exercício atual (status / finish / list)
lightyear cache clear      # limpa o cache local de respostas da API
lightyear update           # atualiza o binário pelo GitHub Releases (--check / -y)
lightyear version          # versão e novidades dela (ou lightyear --version)
lightyear version --changelog  # histórico completo (CHANGELOG.md)
lightyear config path      # caminho do config.yaml
lightyear config show      # configuração efetiva (secret mascarado)
```

Primeiro uso: `lightyear setup` → criar app na Intra → colar UID/Secret → `lightyear login`.

O `setup` também instala o autocomplete do shell (`$SHELL`: zsh/bash/fish).
Para reinstalar: `lightyear completion install` (depois `exec zsh` / novo terminal).
Em `lightyear subject <TAB>` aparecem os projetos do catálogo (ex.: `push_swap`).

O token OAuth (access + refresh) é guardado no keyring do sistema — Keychain (macOS), Secret Service (Linux) ou Credential Manager (Windows) — e renovado automaticamente.

### Notificações no celular

Avisa por push quando uma avaliação nova entra na sua agenda — como avaliador
ou como avaliado. A entrega usa o [ntfy](https://ntfy.sh): grátis, open source,
sem conta.

```bash
lightyear notify setup   # gera um tópico aleatório e grava no config.yaml
lightyear notify test    # confirma que a notificação chega no celular
lightyear notify watch   # deixa rodando (Ctrl+C para sair)
```

No celular, instale o app **ntfy** (App Store / Play Store / F-Droid) e assine o
tópico mostrado pelo `setup`.

> O tópico é o segredo: em servidores públicos, quem souber o nome recebe as
> suas notificações. Por isso o `setup` gera um nome aleatório — evite trocar
> por algo fácil de adivinhar. Para um servidor próprio ou tópico protegido,
> use `--server` e `--token`.

Na primeira checagem nada é enviado: a agenda atual vira a linha de base, para
você não receber uma rajada de avisos sobre avaliações que já conhecia.

Para rodar sem deixar um terminal aberto, agende o `check` (que checa uma vez e
sai):

```cron
*/5 * * * * /usr/local/bin/lightyear notify check >/dev/null 2>&1
```

O intervalo mínimo do `watch` é 1 minuto — a agenda é cacheada por 1 minuto e o
rate limit da Intra é compartilhado com os outros comandos.

O estado do que já foi avisado fica em `$XDG_DATA_HOME/42cli/notifications.json`.

### Subjects (PDF)

Requer `lightyear login`. Sem sessão autenticada o comando recusa o acesso.

A API pública não expõe attachments de subject (HTTP 403 para alunos). O PDF
é servido na CDN (`cdn.intra.42.fr/pdf/pdf/<id>/…`). O CLI resolve o id assim:

1. `--pdf-id` / `subject set-id` (grava no índice local)
2. índice local (`$XDG_DATA_HOME/42cli/subjects/index.json`) — na 1ª utilização
   é preenchido automaticamente com o catálogo embutido (~240 projetos)
3. catálogo embutido (`internal/subjects/catalog.json`)
4. página HTML do projeto na Intra (quando acessível)

```bash
lightyear subject push_swap
lightyear subject set-id push_swap 193464   # corrigir/atualizar um id
```

Para regenerar o catálogo partilhado: use um scraper Playwright local
(não versionado neste repo), depois abra um PR atualizando
`internal/subjects/catalog.json` ou rode `lightyear subject import ./catalog.json`.

### Simulador de provas

Funciona offline e não precisa de login; só de um compilador C (`cc`).
Os 56 exercícios do Exam Rank 02 (níveis 1–4) vêm embutidos no binário, com
enunciado em inglês e português.

```bash
lightyear exam start              # nível 1, exercício sorteado, 3h (--time 90m)
# leia ~/lightyear-exam/subjects/<ex>/subject.pt.txt
# escreva em ~/lightyear-exam/rendu/<ex>/<ex>.c
lightyear exam grademe            # compila com -Wall -Wextra -Werror e compara
                                  # com a solução de referência
lightyear exam status             # exercício, nota e tempo restante
lightyear exam finish             # encerra (o código fica em rendu/ e history/)
lightyear exam practice ft_split  # um exercício só, sem tempo
lightyear exam list               # todos os exercícios por nível
```

Passou → próximo nível (25 pontos cada). Falhou → o trace com esperado ×
recebido fica em `~/lightyear-exam/traces/`.

Quer adicionar exercícios (outros ranks, outros níveis)? Cada um é uma pasta em
`internal/exam/exercises/rank<NN>/level<N>/<nome>/` com `subject.en.txt`,
`subject.pt.txt`, `meta.json` (arquivos + testes) e `ref/` (solução de
referência) — veja os existentes e abra um PR. Créditos em
[internal/exam/CREDITS.md](internal/exam/CREDITS.md).

## Configuração

Arquivo padrão (XDG):

```text
$XDG_CONFIG_HOME/42cli/config.yaml   # fallback: ~/.config/42cli/config.yaml
```

Variáveis de ambiente (prefixo `FORTYTWO_`):

| Variável | Descrição |
|----------|-----------|
| `FORTYTWO_CLIENT_ID` | OAuth Client ID |
| `FORTYTWO_CLIENT_SECRET` | OAuth Client Secret |
| `FORTYTWO_API_BASE_URL` | Base da API (default: `https://api.intra.42.fr/v2`) |
| `FORTYTWO_REDIRECT_URI` | Redirect URI do login local |
| `FORTYTWO_NTFY_SERVER` | Servidor ntfy (default: `https://ntfy.sh`) |
| `FORTYTWO_NTFY_TOPIC` | Tópico ntfy que recebe as notificações |
| `FORTYTWO_NTFY_TOKEN` | Token ntfy, para tópicos protegidos |

Exemplo de `config.yaml`:

```yaml
client_id: "seu-client-id"
client_secret: "seu-client-secret"
api_base_url: "https://api.intra.42.fr/v2"
redirect_uri: "http://127.0.0.1:53682/callback"

# Escrito por `lightyear notify setup`.
notifications:
  ntfy_server: "https://ntfy.sh"
  ntfy_topic: "lightyear-3f9a2c...."   # gerado aleatoriamente; funciona como segredo
  ntfy_token: ""                       # só para tópicos protegidos

# Opcional: planta física dos clusters, usada no mapa do `lightyear campus` e nas
# barras de ocupação do `lightyear dashboard`. A API não expõe o layout do campus;
# sem isso, a grade é inferida das sessões ativas. Exemplo (42 São Paulo):
campus_layout:
  "1": { rows: 10, posts: 4 }
  "2": { rows: 12, posts: 6 }
  "3": { rows: 13, posts: 6, seats: 64 }
  # O mapa desenha posts espelhados (pN…p1) por padrão.
  # Use natural_posts: true só se o teu campus for esquerda→direita (p1…pN).
```

## Arquitetura

Clean Architecture: comandos Cobra apenas delegam; a lógica fica em `internal/services`.

Veja [AGENTS.md](AGENTS.md) para a constituição técnica e o roadmap por milestones.

## Desenvolvimento

```bash
make test    # testes
make lint    # golangci-lint
make fmt     # gofmt
make build   # binário ./lightyear
make install # instala em ~/go/bin
```

## Publicar um release

```bash
git tag v0.1.0
git push origin v0.1.0
```

O workflow [Release](.github/workflows/release.yml) roda o GoReleaser e publica
os binários em [Releases](https://github.com/nvizble/Lightyear42/releases).

## Roadmap

1. **Bootstrap** (concluído) — Cobra, config, CI
2. **OAuth2** (concluído) — `login` / `logout`, keyring
3. **Cliente API** (concluído) — retries, erros tipados, cache
4. **Comandos** (concluído) — `me`, `profile`, `search`, …
5. **Dashboard** (concluído) — Bubble Tea em tempo real
6. **Release** (concluído) — docs, GoReleaser, GitHub Releases
7. **Self-update** (concluído) — `lightyear update` via GitHub Releases
8. **Notificações** (concluído) — push no celular via ntfy (`lightyear notify`)
9. **Simulador de provas** — `lightyear exam` (Exam Rank 02); próximo: TUI examshell

Chat/DM no terminal: parked (API sem DMs públicos; fórum ≠ chat).

## Licença

[MIT](LICENSE)
