---
title: Instalação
order: 2
description: Instale o lightyear com um comando, sem sudo, e mantenha ele atualizado.
---

## Um comando (Linux e macOS) {#um-comando}

```bash
curl -fsSL https://nvizble.github.io/Lightyear42/install.sh | sh
```

Sem `curl` (comum em VMs Debian recém-instaladas), use o `wget`:

```bash
wget -qO- https://tinyurl.com/lightyear42 | sh
```

Funciona nas máquinas da 42: não pede `sudo` nem depende do Go instalado. Depois, **abra um terminal novo** e rode `lightyear version`.

### O que o instalador faz

1. Descobre o sistema (Linux ou macOS) e o processador (x86_64 ou arm64).
2. Baixa o release correspondente do [GitHub Releases](https://github.com/nvizble/Lightyear42/releases).
3. Confere o sha256 do download com o `checksums.txt` do release.
4. Instala o binário em `~/.local/bin/lightyear`.
5. Se essa pasta não estiver no `PATH`, adiciona uma linha (marcada com `# lightyear`) no rc do seu shell: `~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish` ou `~/.profile`.

### Opções

Passe as opções depois de `sh -s --`:

| Opção | O que faz |
|-------|-----------|
| `--canary` | instala a canary mais nova (pre-release) |
| `--version 1.2.1` | instala uma versão específica |
| `--dir ~/bin` | instala em outra pasta (padrão: `~/.local/bin`) |

```bash
curl -fsSL https://nvizble.github.io/Lightyear42/install.sh | sh -s -- --canary
wget -qO- https://tinyurl.com/lightyear42 | sh -s -- --version 1.2.1
```

A pasta também pode vir da variável `LIGHTYEAR_DIR`.

## Ubuntu / Debian (.deb) {#deb}

Se você tem `sudo`, o pacote `.deb` instala em `/usr/bin`:

```bash
VER=1.2.0   # ajuste para a versão desejada
curl -sLO "https://github.com/nvizble/Lightyear42/releases/download/v${VER}/lightyear_${VER}_linux_amd64.deb"
sudo apt install "./lightyear_${VER}_linux_amd64.deb"
lightyear version
```

Em ARM64, troque `linux_amd64.deb` por `linux_arm64.deb`.

## macOS ou Linux, manualmente {#manual}

Baixe o `.tar.gz` do seu sistema no [GitHub Releases](https://github.com/nvizble/Lightyear42/releases) e coloque o binário numa pasta do `PATH`:

```bash
VER=1.2.0
curl -sL "https://github.com/nvizble/Lightyear42/releases/download/v${VER}/lightyear_${VER}_Darwin_arm64.tar.gz" \
  | tar -xz lightyear
mkdir -p ~/.local/bin && mv lightyear ~/.local/bin/
lightyear version
```

Os nomes dos arquivos seguem o padrão `lightyear_<versão>_<SO>_<arch>.tar.gz`: `Darwin_arm64` (Apple Silicon), `Darwin_x86_64` (Mac Intel), `Linux_x86_64` e `Linux_arm64`.

## Windows {#windows}

Baixe o `lightyear_<versão>_Windows_x86_64.zip` no [GitHub Releases](https://github.com/nvizble/Lightyear42/releases), extraia o `lightyear.exe` e coloque numa pasta do `PATH`.

> **Atenção:** o `lightyear test` precisa de Linux ou macOS (usa `fork`). No Windows, use o WSL para rodar os testes de projeto.

## Via Go {#go}

```bash
go install github.com/nvizble/Lightyear42/cmd/lightyear@latest
```

Exige **Go 1.25+** e um **compilador C** (`cc`, `gcc` ou `clang`): o realce de sintaxe do editor usa Tree-sitter, que é C. O binário vai para `$(go env GOPATH)/bin` (no Windows, `%USERPROFILE%\go\bin`), que precisa estar no `PATH`.

> **Atenção:** o Go das máquinas da 42 costuma ser mais antigo (1.23). Lá, prefira o [instalador de um comando](#um-comando): ele não depende do Go.

## Atualizar {#atualizar}

Quando sai uma versão nova do seu canal, o lightyear avisa no fim dos comandos:

```text
✦ Nova versão do lightyear disponível: 1.3.0-canary.35 → 1.3.0-canary.36
  Atualize com: lightyear update
```

No app em tela cheia, o aviso aparece na barra de abas. O lightyear confere no máximo uma vez por dia, em segundo plano, sem atrasar o comando e sem usar a API do GitHub. O aviso fica de fora de scripts e pipes, em CI e nos comandos `update` e `version`. Para desligar: `export LIGHTYEAR_NO_UPDATE_NOTIFIER=1`.


```bash
lightyear update --check   # só verifica se há versão nova
lightyear update           # baixa e substitui o binário atual
lightyear update -y        # sem pedir confirmação
```

O `update` também não usa Go. Ele precisa de permissão de escrita na pasta do binário: em `~/.local/bin`, tudo certo; numa instalação via `.deb` (em `/usr/bin`), use `sudo` ou instale o `.deb` novo pelo `apt`.

Builds sem versão de release (compiladas localmente, versão `dev`) só atualizam com `--force`.

### Canais: estável e canary {#canais}

| Comando | O que faz |
|---------|-----------|
| `lightyear update --canary` | entra no canal canary: pre-releases `v1.3.0-canary.N`, com as novidades antes |
| `lightyear update --stable` | volta para o canal estável (instala a última estável, mesmo que seja mais antiga que a canary instalada) |

Quem está numa canary continua recebendo canaries no `lightyear update`, até voltar com `--stable`. Para ver o que mudou na versão instalada, rode `lightyear --version`; o histórico completo está em `lightyear version --changelog`.

## Desinstalar {#desinstalar}

1. Saia da conta, para remover o token do keyring do sistema:

   ```bash
   lightyear logout
   ```

2. Apague o binário (o caminho aparece em `command -v lightyear`):

   ```bash
   rm "$(command -v lightyear)"
   ```

   Se instalou pelo `.deb`, use `sudo apt remove lightyear`.

3. Se quiser, apague também os dados locais. O `lightyear config show` mostra os caminhos usados na sua máquina; os padrões são:

   | Caminho | Conteúdo |
   |---------|----------|
   | `~/.config/42cli/` | `config.yaml` (credenciais OAuth, ntfy, amigos, tema do editor) |
   | `~/.local/share/42cli/` | PDFs e índice de subjects, estado das notificações, sessão do simulador |
   | `~/.cache/42cli/` | cache das respostas da API; no Linux, também o clangd e o ty baixados pelo editor |
   | `~/Library/Caches/42cli/` | só no macOS: o ty baixado pelo editor |
   | `~/lightyear-exam/` | suas soluções do simulador de provas |

   As variáveis `XDG_CONFIG_HOME`, `XDG_DATA_HOME` e `XDG_CACHE_HOME` mudam a pasta base das três primeiras, se estiverem definidas.

4. Remova a linha `# lightyear` que o instalador pôs no rc do seu shell, se houver.
