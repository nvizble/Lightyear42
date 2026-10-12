---
title: Wiki do lightyear
order: 1
description: A 42 inteira, direto do terminal. Tudo o que o lightyear faz e como usar.
---

O **lightyear** é uma CLI open source para quem estuda na 42 Network. Ela conversa com a API oficial da Intra e traz para o terminal o que você faria no navegador: seu perfil, projetos, avaliações agendadas, slots, quem está no campus e o PDF dos subjects. Também faz coisas que a Intra não faz: avisa no celular quando surge uma avaliação nova, simula os Exam Ranks 02 a 05 offline e testa os seus projetos com centenas de casos.

É feita para estudantes da 42 (Common Core e piscine), funciona nas máquinas do campus sem `sudo` e roda em Linux, macOS e Windows (os testes de projeto precisam de Linux ou macOS).

```bash
curl -fsSL https://nvizble.github.io/Lightyear42/install.sh | sh
lightyear setup && lightyear login
lightyear
```

## Mapa da wiki

### Começar

- [Instalação](instalacao.html): um comando, `.deb`, binário, `go install`, atualizar e desinstalar.
- [Primeiros passos](primeiros-passos.html): criar a aplicação OAuth na Intra, login e autocomplete.
- [Solução de problemas](problemas.html): os erros mais comuns e como resolver.

### Usar

- [Referência de comandos](comandos.html): todos os comandos e flags.
- [App em tela cheia](app.html): o `lightyear` sem argumentos, com abas, mouse e atalhos.
- [Notificações no celular](notificacoes.html): push a cada avaliação nova, via ntfy.

### Estudar

- [Simulador de provas](simulador.html): os Exam Ranks 02 a 05 (C e Python) no seu computador, cronometrados e offline.
- [Testes de projeto](testes.html): `lightyear test` para libft, ft_printf, get_next_line, módulos de Python e born2beroot.
- [Editor embutido](editor.html): o editor estilo Vim, com erros do compilador e autocomplete.

### Participar

- [Contribuir](contribuir.html): arquitetura, ambiente de desenvolvimento, como adicionar exercícios e testes.

## Estável e canary

O canal **estável** (hoje a série 1.2.x) é o padrão e recebe só correções. O canal **canary** (`v1.3.0-canary.N`) traz as novidades antes, menos testadas: o app em tela cheia, o editor embutido e o `lightyear test` estão só nele por enquanto.

> **Canary:** as páginas marcadas assim descrevem recursos que ainda não chegaram à versão estável. Para usá-los, instale com `--canary` ou rode `lightyear update --canary` (veja [Instalação](instalacao.html#canais)).

O código está em [github.com/nvizble/Lightyear42](https://github.com/nvizble/Lightyear42), sob licença MIT.
