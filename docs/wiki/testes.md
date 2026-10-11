---
title: Testes de projeto (lightyear test)
order: 7
description: Teste o seu projeto da 42 com centenas de casos tirados do subject, cada um isolado.
---

> **Canary:** o `lightyear test` chegou na `v1.3.0-canary.32` (Python 05 a 10 e os bis na `canary.35`). Instale a canary para usar (veja [Instalação](instalacao.html#canais)).

O `lightyear test` confere o seu projeto contra o subject: as regras de entrega (README, norminette, Makefile, funções permitidas, variáveis globais) e o comportamento de cada função, com os casos de borda. Cada caso roda isolado, então um segfault, um loop infinito ou um vazamento derrubam só aquele caso, com o esperado e o recebido.

| Projeto | Comando |
|---------|---------|
| [libft](#libft) | `lightyear test libft` |
| [ft_printf](#ft_printf) | `lightyear test ft_printf` |
| [get_next_line](#get_next_line) | `lightyear test get_next_line` |
| [Módulos de Python](#python) | `lightyear test python-module-00` … `-07`, `-09`, `-10`, `-05-bis`, `-06-bis` |
| [born2beroot](#born2beroot) | `sudo lightyear test born2beroot` (dentro da VM) |

## Como rodar {#como-rodar}

Entre na **raiz do projeto** (a pasta com o `Makefile`, ou com `ex0/` nos módulos de Python) e rode:

```bash
cd ~/libft
lightyear test libft
```

Se a pasta não parecer a raiz do projeto, o lightyear avisa o que falta (por exemplo, `falta Makefile, libft.h`).

### No editor

Sem flags, o `lightyear test` abre o [editor do lightyear](editor.html) na pasta do projeto, com os arquivos numa árvore à esquerda e um botão **▶ Rodar testes** em cima.

| Tecla | Ação |
|-------|------|
| `Espaço` `e` | mostra / esconde a árvore de arquivos (`Enter` abre, `h`/`l` fecham e abrem pastas, `q` fecha) |
| `Espaço` `r` | roda os testes (salva os arquivos antes) |
| `Espaço` `t` | mostra / esconde os resultados (`Enter` num erro abre o código dele, `r` roda de novo) |
| `esc` | volta o foco para o código |
| `:q` | sai |

O mouse funciona em tudo: clique no botão, nos arquivos e nos erros. Clicar num erro abre o arquivo na linha dele. Você corrige, aperta `Espaço` `r` e vê o resultado de novo, sem sair do editor.

### No terminal

```bash
lightyear test libft --run                       # imprime o resultado e sai
lightyear test libft --run --all                 # lista também os casos que passaram
lightyear test libft --run --only ft_split       # só uma função
lightyear test libft --run --only ft_split,ft_itoa
```

Com `--run`, o comando sai com código de erro se algum caso falhar, o que serve para scripts. O `--only` escolhe funções da libft; as checagens do projeto (README, Makefile…) rodam sempre. Nos outros projetos, ele não muda nada.

O resultado mostra uma linha por grupo e, embaixo dos grupos que falharam, cada caso com o problema. Por exemplo, numa libft com um `ft_strlen` que conta um a mais (resumido):

```text
✓ README.md 5
– norminette (pulado)
    – norminette
        norminette não está instalada (pip install norminette)
✓ Makefile 14
✗ ft_strlen 17 de 17 falharam
    ✗ ft_strlen("")
        esperado 0, recebido 1
    ✗ ft_strlen("a")
        esperado 1, recebido 2
    …

libft: 690 passaram · 17 falharam · 1 pulados (14.2s)
```

## Como funciona {#como-funciona}

- **Uma cópia do projeto.** Nada é compilado na sua pasta: o lightyear copia o projeto (sem `.git` e sem `.o`, `.a` e `.d`) para uma pasta temporária e roda o `make` e os testes lá.
- **Cada caso num processo.** Um crash, um loop infinito ou um `exit()` no meio do teste falham só aquele caso.
- **Um `malloc` do lightyear.** Nos projetos em C, os testes trocam o `malloc` e o `free` por versões que:
  - acusam **vazamento**: o que o caso deixa alocado no fim é falha;
  - acusam **double free** e `free` de um ponteiro que não veio do `malloc`;
  - acusam **escrita além do bloco** (heap overflow, o clássico "faltou o +1 no malloc");
  - enchem cada bloco novo com lixo, para um `'\0'` esquecido aparecer;
  - **falham de propósito, um de cada vez**: nas funções que alocam, o 1º `malloc` devolve `NULL`, depois o 2º, e assim por diante. A função tem que devolver `NULL` e liberar o que já tinha alocado.
- **Tempo limite.** Cada caso tem 5 segundos. Se uma função travar em 3 casos, os casos seguintes dela são pulados. Cada `make` tem 3 minutos e a suíte inteira, 10.

> **Atenção:** a política de vazamento é estrita. Memória ainda guardada numa variável `static` quando o caso termina **conta como vazamento**: tudo tem que ser liberado. Isso pega, por exemplo, o get_next_line que não libera o buffer guardado ao chegar no fim do arquivo.

### Status dos casos

| Status | Significado |
|--------|-------------|
| `OK` | passou |
| `KO` | resultado errado, vazamento, double free, overflow ou malloc sem proteção (o detalhe diz qual) |
| `CRASH` | o programa morreu: segmentation fault, bus error, abort… |
| `TIMEOUT` | passou do tempo limite (loop infinito ou esperando entrada?) |
| `SKIP` | pulado: ferramenta não instalada, bônus não feito, função que já travou |

Casos pulados não contam como falha.

### O que precisa estar instalado

| Para | Precisa | Opcional (se faltar, aparece como pulado) |
|------|---------|-------------------------------------------|
| Projetos em C | `cc`, `make`, `nm` | `norminette` |
| Módulos de Python | `python3` | `flake8`, `mypy`, e os pacotes que o módulo usa: `pydantic`, `pandas`, `numpy`, `python-dotenv` |
| born2beroot | `bash`, rodar como root | — |

Os testes precisam de Linux ou macOS (usam `fork`). No Windows, use o WSL.

## libft {#libft}

```bash
cd ~/libft && lightyear test libft
```

Mais de 700 casos, seguindo a versão 19 do subject:

- **README.md** na raiz, com a primeira linha em itálico (`*This project has been created as part of the 42 curriculum by <login>.*`) e as seções Description, Instructions e Resources. A mesma checagem vale para ft_printf e get_next_line.
- **norminette**, um caso por arquivo.
- **Arquivos:** tudo na raiz, só `ft_*.c` e `libft.h`.
- **Makefile:** as regras `libft.a`, `all`, `clean`, `fclean` e `re` existem e fazem algo; compila com `-Wall -Wextra -Werror`; usa `ar` (`libtool` é proibido); `make` de novo não refaz nada (sem relink); `clean` apaga os `.o` e mantém a `libft.a`; `fclean` apaga a `libft.a`; `re` recompila.
- **Variáveis globais:** nenhuma.
- **Funções permitidas:** só `malloc`, `free` e `write`.
- **As 43 funções**, cada uma com seus casos de borda: overlap no `ft_memmove`, `INT_MIN` no `ft_itoa`, overflow no `ft_calloc`, `ft_split` só com separadores, e todos os `malloc` falhando um de cada vez.

As funções de lista (`ft_lst*`) são obrigatórias desde a versão 19. Se a sua libft for do subject antigo e as listas só compilarem com `make bonus`, o lightyear roda o `make bonus` antes de testar.

## ft_printf {#ft_printf}

```bash
cd ~/ft_printf && lightyear test ft_printf
```

- README, norminette, o Makefile (com a `libftprintf.a`), variáveis globais e funções permitidas (`malloc`, `free`, `write`).
- Cada conversão (`c s p d i u x X %`) comparada com o `printf` de verdade, a saída e o retorno, com os casos de borda: `INT_MIN`, `%c` com 0, `%s` com `NULL`, `%p` com `NULL`, `UINT_MAX`…
- Sem buffer: a saída tem que sair na hora, intercalada com chamadas a `write`.
- Devolve -1 quando o `write` falha.
- **Bônus**, se o Makefile tiver a regra `bonus`: em duas partes, `-0.` com largura, e `# +`, com milhares de combinações de flags, largura e precisão. Uma parte não feita (nenhum caso passou) aparece como pulada, porque o subject não pede todas.

## get_next_line {#get_next_line}

```bash
cd ~/get_next_line && lightyear test get_next_line
```

A pasta precisa ter `get_next_line.c`, `get_next_line_utils.c` e `get_next_line.h`.

- README e norminette.
- Compila com `-Wall -Wextra -Werror`, com `-D BUFFER_SIZE=42` e **sem** ele (o seu código precisa de um valor padrão).
- Funções permitidas: só `read`, `malloc` e `free` (sem `lseek`); nenhuma variável global.
- Cerca de 30 casos rodados com cada `BUFFER_SIZE`: 1, 2, 5, 42, 1000 e 10000000. Arquivo vazio, só `\n`, linhas do tamanho do buffer, uma linha de 100000 caracteres, entrada padrão, pipe que chega aos pedaços, fd inválido, diretório, erro de leitura no meio, nada alocado no fim e cada `malloc` falhando.
- Acusa quem lê o arquivo inteiro antes de devolver a primeira linha.
- **Bônus**, se existirem `get_next_line_bonus.c`, `get_next_line_utils_bonus.c` e `get_next_line_bonus.h`: os mesmos casos, mais vários fds ao mesmo tempo, e só uma variável `static`.

## Módulos de Python {#python}

```bash
cd ~/python00 && lightyear test python-module-00
```

Rode na pasta do módulo, a que tem `ex0/`, `ex1/`… O módulo 06 precisa da pasta `alchemy/`, e o 06 bis de `data_processor.py` e `ex3/`. O módulo 08 ainda não tem testes.

Todo módulo tem as mesmas checagens comuns:

- **entrega:** cada arquivo que o subject pede;
- **flake8** e **mypy** sem erros (pulados se não estiverem instalados);
- **type hints** em todas as funções, onde o subject pede;
- **sintaxe** válida;
- **funções autorizadas:** só as que o subject permite em cada exercício.

E o comportamento de cada exercício, com os exemplos do subject e casos de borda. Cada caso roda num processo `python3` próprio, em geral com 5 segundos de limite.

| Módulo | Tema | O que os casos cobrem |
|--------|------|-----------------------|
| `python-module-00` | Growing Code | as funções do jardim (ex0 a ex7), com `input` e as versões iterativa e recursiva |
| `python-module-01` | Code Cultivation | classes do jardim: herança, encapsulamento, `super`, métodos de classe e estáticos |
| `python-module-02` | Garden Guardian | exceções do tipo certo, exceções próprias, `finally` |
| `python-module-03` | Data Quest | listas, tuplas, sets, dicts, geradores e comprehensions |
| `python-module-04` | Data Archivist | arquivos, streams e context managers: arquivo que não existe, sem permissão |
| `python-module-05` | Code Nexus | classes abstratas que não dá para instanciar, polimorfismo, o pipeline de dados |
| `python-module-06` | The Codex | o pacote `alchemy`, a forma de import que cada script pede, imports circulares |
| `python-module-07` | DataDeck | fábricas abstratas, capacidades e estratégias |
| `python-module-09` | Cosmic Data | modelos do Pydantic v2, com cada restrição testada nas bordas |
| `python-module-10` | FuncMage Chronicles | lambdas, funções de ordem superior, closures, `functools` e decoradores |
| `python-module-05-bis` | The Codex | venv, `requirements.txt`, `pyproject.toml`, configuração por variáveis de ambiente e `.env` (sem segredo no código, `.env` fora do git), o pacote `alchemy` |
| `python-module-06-bis` | Code Nexus | classes abstratas, Protocols, fábricas abstratas e estratégias |

Os módulos 00 a 04 têm de 77 a 130 casos; os outros, de 150 a 225.

> **Dica:** pacotes que o módulo usa e que não estão instalados (pydantic, pandas, numpy, python-dotenv) fazem os casos que dependem deles aparecerem como pulados, não como erro. Para rodar tudo, instale-os, por exemplo com `pip install --user pydantic flake8 mypy`.

## born2beroot {#born2beroot}

O born2beroot é diferente: não testa uma pasta, testa a **máquina**. Rode dentro da VM avaliada, como root. O resultado sai direto no terminal, sem editor.

### Com o lightyear

A VM costuma não ter `curl`; instale com o `wget`, sem sudo:

```bash
wget -qO- https://tinyurl.com/lightyear42 | sh
sudo ~/.local/bin/lightyear test born2beroot
```

Use o caminho completo: o `sudo` troca o `PATH` e normalmente não acha o `~/.local/bin`.

### Sem instalar nada

O teste é um script bash que também roda sozinho:

```bash
wget -qO b2b.sh https://raw.githubusercontent.com/nvizble/Lightyear42/main/internal/tester/sh/born2beroot.sh && sudo bash b2b.sh
```

O login vem do hostname (sem o `42` do fim). Se for outro, passe como argumento: `sudo bash b2b.sh <login>`.

### O que é checado

Vale para Debian (AppArmor e UFW) e Rocky (SELinux e firewalld).

| Grupo | Checagens |
|-------|-----------|
| sistema | Linux; Debian estável (sem testing/unstable) ou Rocky; sem interface gráfica |
| partições | partição criptografada (LUKS), pelo menos 2 volumes LVM, os LVM dentro da partição criptografada |
| AppArmor / SELinux | AppArmor ativo, ligado na inicialização e com perfis carregados; ou SELinux em Enforcing e ligado na inicialização |
| SSH | serviço rodando e ligado na inicialização, escutando na 4242, nada na 22, `PermitRootLogin no`, `Port 4242` na configuração |
| UFW / firewalld | instalado, ativo, ligado na inicialização, só a porta 4242 aberta |
| hostname e usuários | hostname termina em 42 e é `<login>42`; o usuário existe e está nos grupos `sudo` e `user42` |
| política de senha | `login.defs` (30, 2 e 7 dias), `chage` de cada usuário, `pam_pwquality` com mínimo de 10 caracteres, maiúscula, minúscula, número, no máximo 3 iguais seguidos, sem o nome do usuário, 7 caracteres diferentes da anterior, valendo para o root |
| sudo | instalado; 3 tentativas; mensagem de senha errada própria; log em `/var/log/sudo/`; `log_input` e `log_output`; `iolog_dir`; `requiretty`; `secure_path` restrito |
| monitoring.sh | no cron a cada 10 minutos e ao ligar; cron ativo; escrito em bash; usa o `wall`; roda sem erros; e **cada valor que ele mostra** conferido com a máquina: arquitetura e kernel, CPUs físicas, vCPUs, memória (total e percentual, em MiB, MB ou GB), disco, CPU, último boot, LVM, conexões TCP, usuários, IPv4 e MAC, comandos sudo |

Sem root, as checagens que precisam dele aparecem como puladas, com o aviso `rode como root: sudo lightyear test born2beroot`.
