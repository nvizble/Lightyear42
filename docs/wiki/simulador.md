---
title: Simulador de provas
order: 8
description: Treine o Exam Rank 02 no seu computador, cronometrado e offline, com correção local.
---

O `lightyear exam` simula as provas da 42 no seu computador. Os **56 exercícios do Exam Rank 02** (níveis 1 a 4) vêm embutidos no binário, com enunciado em inglês e em português. Funciona offline e não precisa de login: só de um compilador C (`cc`).

```bash
lightyear exam start      # começa a prova: nível 1, exercício sorteado, 3 horas
lightyear exam grademe    # corrige o exercício atual
```

> **Dica:** no [app em tela cheia](app.html#exam) (canary), a aba Exam faz o mesmo com teclas: `s` começa, `e` abre o subject e o código lado a lado no editor, `g` corrige e `f` encerra.

## O fluxo de uma prova

1. **Comece** a prova:

   ```bash
   lightyear exam start              # Exam Rank 02, 3 horas
   lightyear exam start --time 90m   # outra duração
   ```

2. **Leia o enunciado** em `~/lightyear-exam/subjects/<exercício>/subject.pt.txt` (ou `subject.en.txt`).

3. **Escreva a solução** em `~/lightyear-exam/rendu/<exercício>/`, com o nome de arquivo que o enunciado pede (por exemplo, `~/lightyear-exam/rendu/first_word/first_word.c`).

4. **Corrija**:

   ```bash
   lightyear exam grademe
   ```

   Passou: você ganha os pontos do nível e recebe um exercício sorteado do nível seguinte. Falhou: continua no mesmo exercício, e o motivo fica em `~/lightyear-exam/traces/<exercício>.trace`.

5. **Acompanhe** o exercício, a nota e o tempo restante:

   ```bash
   lightyear exam status
   ```

6. **Encerre** quando terminar (ou desistir):

   ```bash
   lightyear exam finish
   ```

   ```text
   Prova encerrada com 50/100. Seu código continua em /home/voce/lightyear-exam/rendu.
   ```

A sessão fica salva: fechar o terminal não perde a prova. Só pode haver uma prova (ou prática) por vez; para começar outra, encerre a atual com `lightyear exam finish`. Quando o tempo acaba, o `grademe` para de corrigir e pede para encerrar.

## Pontuação

Cada nível vale uma parte igual de 100 pontos. No Exam Rank 02, com 4 níveis, cada exercício aprovado vale **25 pontos**. Passar no nível 4 termina a prova com 100.

## Treinar um exercício

```bash
lightyear exam list               # todos os exercícios, por rank e nível
lightyear exam practice ft_split  # um exercício só, sem tempo
```

A prática usa as mesmas pastas e o mesmo `grademe`, sem cronômetro. Ao passar, a prática termina.

## As pastas {#pastas}

Tudo fica em `~/lightyear-exam/`, como na prova de verdade:

| Pasta | Conteúdo |
|-------|----------|
| `subjects/<exercício>/` | o enunciado (`subject.en.txt`, `subject.pt.txt`) e, em alguns exercícios, arquivos fornecidos pela prova (como um header) |
| `rendu/<exercício>/` | a sua entrega |
| `traces/` | o trace de cada exercício que falhou |
| `history/` | entregas de provas anteriores |

Seu código nunca é apagado. Ao começar uma prova nova, `subjects/` e `traces/` são limpos e uma `rendu/` com arquivos é movida para `history/rendu-<data>-<hora>/`.

## Como a correção funciona {#correcao}

O `grademe` faz o mesmo que o grader da prova:

1. Compila a sua entrega com `cc -Wall -Wextra -Werror`. Nos exercícios de função (sem `main`), o lightyear junta um `main` de teste. Os arquivos fornecidos pela prova entram na compilação.
2. Compila a **solução de referência** do exercício do mesmo jeito.
3. Roda os dois com cada conjunto de argumentos do teste e **compara a saída padrão**. O código de saída não conta, só a saída.
4. Para no primeiro teste que falhar.

O trace explica a falha:

| Situação | O que o trace mostra |
|----------|----------------------|
| arquivo faltando | `arquivo não entregue: <nome>` |
| não compila | `erro de compilação:` e a saída do compilador |
| saída diferente | o comando com os argumentos, `esperado:` e `recebido:` |
| crash ou demora | o comando e o erro (cada teste tem 2 segundos) |

Por exemplo, um trace de saída diferente (ilustrativo):

```text
first_word "FOR PONY"
esperado: "FOR\n"
recebido: "FOR PONY\n"
```

## Os exercícios do Exam Rank 02

| Nível | Exercícios |
|-------|------------|
| 1 (12) | first_word, fizzbuzz, ft_putstr, ft_strcpy, ft_strlen, ft_swap, repeat_alpha, rev_print, rot_13, rotone, search_and_replace, ulstr |
| 2 (19) | alpha_mirror, camel_to_snake, do_op, ft_atoi, ft_strcmp, ft_strcspn, ft_strdup, ft_strpbrk, ft_strrev, ft_strspn, is_power_of_2, last_word, max, print_bits, reverse_bits, snake_to_camel, swap_bits, union, wdmatch |
| 3 (15) | add_prime_sum, epur_str, expand_str, ft_atoi_base, ft_list_size, ft_range, ft_rrange, hidenp, lcm, paramsum, pgcd, print_hex, rstr_capitalizer, str_capitalizer, tab_mult |
| 4 (10) | flood_fill, fprime, ft_itoa, ft_list_foreach, ft_list_remove_if, ft_split, rev_wstr, rostring, sort_int_tab, sort_list |

Quer adicionar exercícios ou outros ranks? Veja [Contribuir](contribuir.html#exercicios).
