---
title: Notificações no celular
order: 9
description: Receba um push no celular sempre que uma avaliação nova entrar na sua agenda.
---

O `lightyear notify` avisa no celular quando aparece uma avaliação nova na sua agenda, como avaliador ou como avaliado. A entrega usa o [ntfy](https://ntfy.sh): grátis, open source e sem conta.

Exige [login](primeiros-passos.html#login).

## Configurar em 3 passos

### 1. Gere o tópico

```bash
lightyear notify setup
```

```text
Notificações configuradas.
  servidor: https://ntfy.sh
  tópico:   lightyear-…

No celular:
  1. instale o app ntfy (App Store / Play Store / F-Droid)
  2. assine o tópico "lightyear-…"
     ou abra https://ntfy.sh/lightyear-…

Depois rode: lightyear notify test
```

O `setup` gera um tópico aleatório e grava no `config.yaml`. Rodar de novo mantém o tópico que já existe.

> **Atenção:** o tópico é o segredo. Em servidores públicos como o ntfy.sh, quem souber o nome recebe as suas notificações. Por isso o nome é aleatório: evite trocar por algo fácil de adivinhar.

### 2. Assine no celular

Instale o app **ntfy** (App Store, Play Store ou F-Droid) e assine o tópico mostrado pelo `setup`.

### 3. Teste

```bash
lightyear notify test
```

Chega no celular uma notificação "lightyear": *Notificações configuradas. Você será avisado quando uma nova avaliação for agendada.*

## Como as notificações chegam

| Situação | Título | Texto |
|----------|--------|-------|
| você vai avaliar | "Você vai avaliar <projeto>" | quando, e quem você avalia; tocar abre a avaliação na Intra |
| você será avaliado | "<login> vai te avaliar" | quando, e o projeto |

Na **primeira checagem nada é enviado**: a agenda atual vira a linha de base, para você não receber uma rajada de avisos sobre avaliações que já conhecia. Depois, cada avaliação é avisada uma vez só. O que já foi avisado fica em `~/.local/share/42cli/notifications.json`.

## Deixar rodando: watch

```bash
lightyear notify watch                 # checa a cada 2 minutos
lightyear notify watch --interval 5m
```

Roda até você apertar `Ctrl+C`. O intervalo mínimo é 1 minuto: a agenda fica em cache por 1 minuto e o rate limit da Intra é dividido com os outros comandos. Falhas pontuais de rede ou da API aparecem como aviso, e a checagem seguinte continua normalmente.

## Sem terminal aberto: check no cron {#cron}

O `notify check` checa uma vez, notifica o que for novo e sai. Feito para agendadores (cron, systemd timer, launchd):

```bash
crontab -e
```

```text
*/5 * * * * $HOME/.local/bin/lightyear notify check >/dev/null 2>&1
```

Ajuste o caminho para onde o lightyear está instalado (`command -v lightyear` mostra).

> **Atenção:** o token de login fica no keyring do sistema. Se o `check` funciona no terminal mas falha no cron com um erro de keyring (`ler token do keyring`), o cron não está alcançando o keyring da sua sessão. Nesse caso, deixe o `watch` rodando ou o app aberto.

## Dentro do app {#app}

> **Canary:** disponível a partir da `v1.3.0-canary.28`.

Com o notify configurado, o [app em tela cheia](app.html) também vigia a agenda enquanto está aberto, como o `watch`: checa a cada 2 minutos e manda as avaliações novas para o celular. A barra de abas mostra **notify ligado** e o rodapé diz o que foi avisado.

## Servidor próprio ou tópico protegido

```bash
lightyear notify setup meu-topico --server https://ntfy.exemplo.com --token tk_xxxxxxxx
```

| Flag | Descrição |
|------|-----------|
| `--server string` | servidor ntfy (padrão: `https://ntfy.sh`) |
| `--token string` | token de acesso, para tópicos protegidos |

O mesmo vale por variáveis de ambiente: `FORTYTWO_NTFY_SERVER`, `FORTYTWO_NTFY_TOPIC` e `FORTYTWO_NTFY_TOKEN`.
