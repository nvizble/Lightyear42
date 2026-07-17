# lightyear

CLI moderna, open source, para a [42 Network](https://www.42network.org/), inspirada em ferramentas como `gh`, `docker` e `kubectl`.

> Binário/comando: **`lightyear`** (antes `42`).

> Status: **Milestone 6 (release)** — GoReleaser + GitHub Releases. Binário/comando: **`lightyear`**.
>
> Nota: `lightyear exams` foi cortado — todos os endpoints de exames da API retornam 403 para tokens com scope `public`.

## Requisitos

- Go 1.25+ (só para instalar via `go install` ou desenvolver)
- Uma aplicação OAuth registrada na Intra: [profile.intra.42.fr/oauth/applications](https://profile.intra.42.fr/oauth/applications/new) com Redirect URI `http://127.0.0.1:53682/callback`
- Para `lightyear slots open/close`, ative o scope **projects** na app e rode `lightyear logout && lightyear login`

## Instalação

### Binário (recomendado)

Baixe o release em [GitHub Releases](https://github.com/nvizble/Lightyear42/releases) (macOS / Linux / Windows) ou:

```bash
# exemplo macOS Apple Silicon
curl -sL "https://github.com/nvizble/Lightyear42/releases/latest/download/lightyear_Darwin_arm64.tar.gz" \
  | tar -xz lightyear
sudo mv lightyear /usr/local/bin/
lightyear version
```

(Ajuste `Darwin_arm64` conforme o SO: `Darwin_x86_64`, `Linux_x86_64`, `Linux_arm64`, `Windows_x86_64`.)

### Via Go

```bash
go install github.com/nvizble/Lightyear42@latest
```

### Desenvolvimento

```bash
git clone https://github.com/nvizble/Lightyear42.git
cd Lightyear42
make install   # ~/go/bin/lightyear
lightyear --help
```

Para só compilar no diretório do projeto: `make build` → `./lightyear`.

## Uso

```bash
lightyear setup            # guia OAuth na Intra + grava UID/Secret
lightyear login            # autentica via OAuth2 (abre o navegador)
lightyear logout           # remove o token do keyring
lightyear me               # seu perfil: nível, wallet, pontos, campus
lightyear profile <login>  # perfil de qualquer usuário da 42
lightyear search <termo>   # busca usuários por prefixo de login (-n limita)
lightyear projects [login] # projetos com status e nota (--all inclui piscine)
lightyear evaluations      # próximas avaliações agendadas (alias: evals)
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
lightyear cache clear      # limpa o cache local de respostas da API
lightyear version          # versão do binário
lightyear config path      # caminho do config.yaml
lightyear config show      # configuração efetiva (secret mascarado)
```

Primeiro uso: `lightyear setup` → criar app na Intra → colar UID/Secret → `lightyear login`.

O token OAuth (access + refresh) é guardado no keyring do sistema — Keychain (macOS), Secret Service (Linux) ou Credential Manager (Windows) — e renovado automaticamente.

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

Exemplo de `config.yaml`:

```yaml
client_id: "seu-client-id"
client_secret: "seu-client-secret"
api_base_url: "https://api.intra.42.fr/v2"
redirect_uri: "http://127.0.0.1:53682/callback"

# Opcional: planta física dos clusters, usada no mapa do `lightyear campus` e nas
# barras de ocupação do `lightyear dashboard`. A API não expõe o layout do campus;
# sem isso, a grade é inferida das sessões ativas. Exemplo (42 São Paulo):
campus_layout:
  "1": { rows: 10, posts: 4 }
  "2": { rows: 12, posts: 6 }
  "3": { rows: 13, posts: 6, seats: 64 } # seats: capacidade real de grades irregulares
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

## Licença

[MIT](LICENSE)
