# 42 CLI

CLI moderna, open source, para a [42 Network](https://www.42network.org/), inspirada em ferramentas como `gh`, `docker` e `kubectl`.

> Status: **Milestone 4 concluído** — `me`, `profile`, `search`, `projects` e `campus` funcionais. Próximo marco: dashboard Bubble Tea.
>
> Nota: `42 exams` foi cortado — todos os endpoints de exames da API retornam 403 para tokens com scope `public`.

## Requisitos

- Go 1.25+
- Uma aplicação OAuth registrada na Intra: [profile.intra.42.fr/oauth/applications](https://profile.intra.42.fr/oauth/applications/new) com Redirect URI `http://127.0.0.1:53682/callback`

## Instalação (desenvolvimento)

```bash
git clone https://github.com/joaodiniz/42cli.git
cd 42cli
make build
./42 --help
```

## Uso atual

```bash
./42 login            # autentica via OAuth2 (abre o navegador)
./42 logout           # remove o token do keyring
./42 me               # seu perfil: nível, wallet, pontos, campus
./42 profile <login>  # perfil de qualquer usuário da 42
./42 search <termo>   # busca usuários por prefixo de login (-n limita)
./42 projects [login] # projetos com status e nota (--all inclui piscine)
./42 campus           # mapa de quem está online no campus (--id p/ outro)
./42 campus --friends # mapa filtrado pela sua lista de amigos
./42 friends add <l>  # gerencia a lista local de amigos (add/remove/list)
./42 friends online   # quais amigos estão online e em qual posto
./42 cache clear      # limpa o cache local de respostas da API
./42 version          # versão do binário
./42 config path      # caminho do config.yaml
./42 config show      # configuração efetiva (secret mascarado)
```

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

# Opcional: planta física dos clusters para o `42 campus`.
# A API não expõe o layout do campus; sem isso, a grade é inferida
# a partir das sessões ativas. Exemplo (42 São Paulo):
campus_layout:
  "1": { rows: 10, posts: 4 }
  "2": { rows: 12, posts: 6 }
  "3": { rows: 13, posts: 6 }
```

## Arquitetura

Clean Architecture: comandos Cobra apenas delegam; a lógica fica em `internal/services`.

Veja [AGENTS.md](AGENTS.md) para a constituição técnica e o roadmap por milestones.

## Desenvolvimento

```bash
make test    # testes
make lint    # golangci-lint
make fmt     # gofmt
make build   # binário ./42
```

## Roadmap

1. **Bootstrap** (concluído) — Cobra, config, CI
2. **OAuth2** (concluído) — `login` / `logout`, keyring
3. **Cliente API** (concluído) — retries, erros tipados, cache
4. **Comandos** (atual) — `me`, `profile`, `search`, …
5. **Dashboard** — Bubble Tea
6. **Release** — docs, GoReleaser, multi-OS

## Licença

[MIT](LICENSE)
