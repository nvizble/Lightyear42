---
title: Solução de problemas
order: 10
description: Os erros mais comuns e como resolver cada um.
---

Não achou o seu problema aqui? Abra uma issue em [github.com/nvizble/Lightyear42/issues](https://github.com/nvizble/Lightyear42/issues), com a saída do `lightyear version` e o erro completo.

## Instalação

### `lightyear: command not found` depois de instalar {#path}

O instalador põe o binário em `~/.local/bin` e adiciona essa pasta ao `PATH` no rc do seu shell, mas isso só vale em terminais novos.

- Abra um **terminal novo**, ou rode `export PATH="$HOME/.local/bin:$PATH"` no atual.
- Ainda não achou? Confira se a linha `# lightyear` foi parar no rc do shell que você usa de verdade (`echo $SHELL`): `~/.zshrc`, `~/.bashrc`, `~/.config/fish/config.fish` ou `~/.profile`.

### `go install` falha no campus {#go-antigo}

O lightyear exige **Go 1.25+**, e o Go das máquinas da 42 costuma ser mais antigo (1.23). O `go install` também precisa de um compilador C, por causa do Tree-sitter do editor.

Use o [instalador de um comando](instalacao.html#um-comando): ele baixa o binário pronto e não depende do Go.

### `lightyear update` sem permissão

O `update` substitui o binário onde ele está. Em `~/.local/bin` funciona sem `sudo`. Se você instalou pelo `.deb` (em `/usr/bin`), rode com `sudo` ou instale o `.deb` novo pelo `apt`.

## Login e API

### `client_id/client_secret não configurados`

Falta registrar a aplicação OAuth. Rode `lightyear setup` (veja [Primeiros passos](primeiros-passos.html#setup)).

### `nenhum token salvo: execute lightyear login`

Você ainda não entrou, ou saiu. Rode `lightyear login`.

### O navegador não abre no login

O `login` imprime a URL de autorização ("Se o navegador não abrir, acesse:"). Copie e abra no navegador da mesma máquina: o retorno vai para `http://127.0.0.1:53682/callback`. Se a porta estiver ocupada, o erro diz `abrir porta local 127.0.0.1:53682 (já em uso?)`: feche o outro `lightyear login` que estiver esperando.

### Erro de keyring no Linux sem interface gráfica {#keyring}

O token fica no keyring do sistema, que no Linux é o **Secret Service** (gnome-keyring, KWallet), acessado pela sessão D-Bus do usuário. Por SSH, num servidor ou num container sem sessão gráfica, normalmente não há Secret Service rodando, e o login falha com `salvar token no keyring` (ou `ler token do keyring` nos outros comandos).

Rode o lightyear numa sessão gráfica, onde o keyring está disponível. O lightyear não grava o token em arquivo, de propósito.

### Erro 403 nos slots {#slots-403}

O `lightyear slots open` e `close` precisam do scope **projects**.

1. Na página da sua aplicação na Intra ([profile.intra.42.fr/oauth/applications](https://profile.intra.42.fr/oauth/applications)), marque o scope **projects**.
2. Faça login de novo, para o token incluir o scope:

   ```bash
   lightyear logout && lightyear login
   ```

### Dados desatualizados

As respostas da API ficam em cache por um tempo, para poupar o rate limit da Intra. Para buscar tudo de novo:

```bash
lightyear cache clear
```

### `lightyear subject` não acha o PDF

O id do PDF de alguns projetos não está no catálogo embutido. Abra o subject na Intra, pegue o número na URL `https://cdn.intra.42.fr/pdf/pdf/<id>/en.subject.pdf` e grave:

```bash
lightyear subject set-id <projeto> <id>
```

## Editor

### O editor demora para mostrar os erros na primeira vez {#editor-lento}

- **macOS:** o primeiro clangd aberto por um binário novo do lightyear pode levar uns 15 segundos, enquanto o sistema verifica o programa. A barra de status mostra "clangd…" até ficar pronto. Das próximas vezes é instantâneo.
- **Linux sem clangd** (como nas máquinas da 42): na primeira vez que você abre um arquivo C, o lightyear baixa o clangd oficial (118 MB, só uma vez). Em Python sem pyright nem pylsp, baixa o ty (14 MB). A barra de status avisa enquanto baixa.

### O editor não mostra erros nem autocomplete

Falta o language server da linguagem. O editor avisa como instalar:

| Linguagem | Como instalar |
|-----------|---------------|
| C, C++ (macOS) | `xcode-select --install` |
| C, C++ (Linux) | `sudo apt install clangd` (no Linux x86_64 o lightyear baixa sozinho) |
| Go | `go install golang.org/x/tools/gopls@latest` |
| Python | `npm i -g pyright` ou `pip install python-lsp-server` (sem nenhum, o lightyear baixa o ty) |
| Rust | `rustup component add rust-analyzer` |

### `:Format` não formata o meu C

Em C e C++, o `:Format` só roda com um `.clang-format` no projeto: o estilo padrão quebraria a norminette.

## Testes de projeto

### `compilador "cc" não encontrado` ou erro com `make`/`nm`

Os testes em C precisam de `cc`, `make` e `nm`. No macOS, instale as Command Line Tools (`xcode-select --install`); no Debian/Ubuntu, `sudo apt install build-essential`.

### `... não parece a raiz do libft`

Rode o `lightyear test` na raiz do projeto: a pasta com o `Makefile` e o `libft.h` (ou o equivalente do projeto; nos módulos de Python, a pasta com `ex0/`). A mensagem diz o que falta.

### norminette, flake8 ou mypy aparecem como pulados

Eles são opcionais: sem a ferramenta, a checagem vira `SKIP` e não conta como falha. Para rodar:

```bash
pip install --user norminette flake8 mypy
```

O mesmo vale para os pacotes que alguns módulos de Python usam (pydantic, pandas, numpy, python-dotenv).

### Um caso falha com vazamento, mas eu libero tudo

A política é estrita: memória ainda guardada numa variável `static` no fim do caso conta como vazamento. Veja [como funciona](testes.html#como-funciona).

### `os testes precisam de Linux ou macOS`

Os testes usam `fork`, que o Windows não tem. Use o WSL.

### born2beroot: checagens puladas, ou `sudo: lightyear: command not found` {#born2beroot}

O born2beroot precisa de root para ler a configuração do SSH, do sudo, do firewall e o crontab. Sem root, essas checagens aparecem como puladas (`rode como root`).

O `sudo` troca o `PATH`, então use o caminho completo:

```bash
sudo ~/.local/bin/lightyear test born2beroot
```

Ele também precisa rodar **dentro da VM avaliada** (Linux). Sem lightyear na VM, use o script avulso descrito em [Testes de projeto](testes.html#born2beroot).

## Simulador de provas

### `já existe uma prova em andamento`

Só pode haver uma prova ou prática por vez. Encerre a atual com `lightyear exam finish` (o seu código fica guardado).

### `tempo esgotado`

O tempo da prova acabou e o `grademe` não corrige mais. Encerre com `lightyear exam finish` e comece outra.
