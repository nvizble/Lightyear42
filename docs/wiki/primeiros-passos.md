---
title: Primeiros passos
order: 3
description: Conecte o lightyear à sua conta da Intra em dois comandos.
---

O lightyear usa a API oficial da 42, que pede uma **aplicação OAuth** registrada na Intra. Cada estudante cria a sua (leva um minuto) e o lightyear guarda o UID e o Secret dela.

```bash
lightyear setup   # cria a ligação com a Intra
lightyear login   # entra na sua conta pelo navegador
```

> **Dica:** o [simulador de provas](simulador.html) e o [`lightyear test`](testes.html) funcionam offline e não precisam de login. Pode pular esta página se só quer usá-los.

## lightyear setup {#setup}

O `setup` é um assistente: mostra o passo a passo, abre a página de criação da aplicação no navegador e grava as credenciais no `config.yaml`.

Na página [profile.intra.42.fr/oauth/applications/new](https://profile.intra.42.fr/oauth/applications/new), preencha:

| Campo | Valor |
|-------|-------|
| Name | `lightyear` (ou outro nome) |
| Redirect URI | `http://127.0.0.1:53682/callback` |
| Scopes | marque **public** e **projects** |

Salve, copie o **UID** e o **Secret** e cole no terminal quando o `setup` pedir. O Secret não aparece enquanto você digita.

O scope **projects** é o que permite abrir e fechar [slots de avaliação](comandos.html#slots). Sem ele, o resto funciona, mas `lightyear slots open/close` dá erro 403.

No fim, o `setup` também instala o autocomplete do seu shell (veja [abaixo](#autocomplete)).

### Sem interação

Para scripts, ou se preferir não usar o assistente:

```bash
lightyear setup --client-id SEU_UID --client-secret SEU_SECRET
```

Se já existirem credenciais, o `setup` pergunta antes de sobrescrever; com `--force`, sobrescreve direto. As outras flags estão na [referência](comandos.html#setup).

Também dá para pular o `setup` e usar variáveis de ambiente: `FORTYTWO_CLIENT_ID` e `FORTYTWO_CLIENT_SECRET`.

## lightyear login {#login}

```bash
lightyear login
```

Abre o navegador na página de autorização da Intra e espera o retorno em `http://127.0.0.1:53682/callback` (por até 3 minutos; mude com `--timeout 5m`). Autorize e volte ao terminal.

O token (de acesso e de renovação) fica no **keyring do sistema**, nunca num arquivo:

| Sistema | Onde fica |
|---------|-----------|
| macOS | Keychain |
| Linux | Secret Service (gnome-keyring, KWallet) |
| Windows | Credential Manager |

O token se renova sozinho; você só precisa fazer login de novo se sair ou se mudar os scopes da aplicação.

> **Atenção:** ativou o scope **projects** depois do primeiro login? Faça login de novo para o token novo incluir o scope: `lightyear logout && lightyear login`.

## lightyear logout {#logout}

```bash
lightyear logout
```

Remove o token do keyring. Não revoga a autorização na Intra: para isso, apague a aplicação na sua página de aplicações OAuth.

## Testando

```bash
lightyear me           # seu perfil: nível, wallet, pontos de avaliação, campus
lightyear evaluations  # suas próximas avaliações
```

## O app em tela cheia {#app}

> **Canary:** disponível a partir da `v1.3.0-canary.1`.

Rode só `lightyear`, sem subcomando, para abrir o app em tela cheia: abas clicáveis com o início, avaliações, projetos, subjects, campus, slots e o simulador de provas. Veja [App em tela cheia](app.html).

Sem login, o app abre do mesmo jeito: as abas da API explicam o que falta e a aba Exam funciona offline. Fora de um terminal (em pipes e scripts), o `lightyear` sozinho continua mostrando o help.

## Autocomplete {#autocomplete}

O `setup` já configura o autocomplete do shell em `$SHELL` (zsh, bash ou fish). Para reinstalar, ou para outro shell:

```bash
lightyear completion install            # detecta o $SHELL
lightyear completion install --shell zsh
```

Depois abra um terminal novo (ou rode `exec zsh`). Com o autocomplete, `lightyear subject <TAB>` sugere os projetos do catálogo e `lightyear test <TAB>` os projetos testáveis.

Para gerar o script na mão (PowerShell incluído), use `lightyear completion bash|zsh|fish|powershell`; o `--help` de cada um explica onde salvar.

## Configuração {#configuracao}

A configuração vem de três lugares, nesta ordem de prioridade:

1. variáveis de ambiente com prefixo `FORTYTWO_`;
2. o arquivo `~/.config/42cli/config.yaml` (ou `$XDG_CONFIG_HOME/42cli/config.yaml`);
3. os valores padrão.

```bash
lightyear config path   # caminho do config.yaml
lightyear config show   # configuração efetiva, com o secret mascarado
```

| Variável | Descrição |
|----------|-----------|
| `FORTYTWO_CLIENT_ID` | UID da aplicação OAuth |
| `FORTYTWO_CLIENT_SECRET` | Secret da aplicação OAuth |
| `FORTYTWO_API_BASE_URL` | base da API (padrão: `https://api.intra.42.fr/v2`) |
| `FORTYTWO_REDIRECT_URI` | Redirect URI do login local |
| `FORTYTWO_NTFY_SERVER` | servidor ntfy (padrão: `https://ntfy.sh`) |
| `FORTYTWO_NTFY_TOPIC` | tópico ntfy das notificações |
| `FORTYTWO_NTFY_TOKEN` | token ntfy, para tópicos protegidos |

### Planta do campus {#campus-layout}

A API não informa o layout físico dos clusters. Sem configuração, o mapa do `lightyear campus` é inferido das sessões ativas. Para desenhar a planta real, adicione ao `config.yaml`:

```yaml
campus_layout:
  "1": { rows: 10, posts: 4 }
  "2": { rows: 12, posts: 6 }
  "3": { rows: 13, posts: 6, seats: 64 }
```

O mapa desenha os postos espelhados (pN … p1). Se no seu campus a numeração vai da esquerda para a direita (p1 … pN), adicione `natural_posts: true` ao cluster.
