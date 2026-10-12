---
title: Simulador de provas
order: 8
description: Treine os Exam Ranks 02 a 05 no seu computador, cronometrado e offline, com correção local.
---

O `lightyear exam` simula as provas da 42 no seu computador. Vêm embutidos no binário, com enunciado em inglês e em português, os **56 exercícios em C do Exam Rank 02** (níveis 1 a 4) e os **28 exercícios em Python dos Exam Ranks 03, 04 e 05**. Funciona offline e não precisa de login: só de um compilador C (`cc`) para o Rank 02 e de `python3` para os outros.

```bash
lightyear exam start      # começa a prova: nível 1, exercício sorteado, 3 horas
lightyear exam grademe    # corrige o exercício atual
```

> **Dica:** no [app em tela cheia](app.html#exam) (canary), a aba Exam faz o mesmo com teclas: `s` começa (`p` troca o rank), `e` abre o subject e o código lado a lado no editor, `g` corrige e `f` encerra.

## O fluxo de uma prova

1. **Comece** a prova:

   ```bash
   lightyear exam start              # Exam Rank 02, 3 horas
   lightyear exam start --rank 03    # Exam Rank 03 (Python)
   lightyear exam start --time 90m   # outra duração
   ```

2. **Leia o enunciado** em `~/lightyear-exam/subjects/<exercício>/subject.pt.txt` (ou `subject.en.txt`).

3. **Escreva a solução** em `~/lightyear-exam/rendu/<exercício>/`, com o nome de arquivo que o enunciado pede (por exemplo, `~/lightyear-exam/rendu/first_word/first_word.c` ou `~/lightyear-exam/rendu/py_anagram/py_anagram.py`).

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

Cada nível vale uma parte igual de 100 pontos. No Exam Rank 02, com 4 níveis, cada exercício aprovado vale **25 pontos**; passar no nível 4 termina a prova com 100. No Rank 03 (6 níveis) cada um vale 16, no 04 (4 níveis) 25 e no 05 (3 níveis) 33.

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

### Exercícios em Python {#python}

Nos Ranks 03 a 05 a entrega é **uma função** num arquivo `.py`. O lightyear importa a sua função, chama com os argumentos de cada teste e compara o `repr()` do que ela retorna com o da solução de referência (uma exceção conta como resposta: se a referência retorna e a sua levanta, falha). Antes de rodar, ele lê o seu código e reprova o que o enunciado proíbe (por exemplo, `sorted()` e `list.sort()` no `py_cryptic_sorter`).

```text
bracket_validator("(]")
esperado: False
recebido: True
```

## Os exercícios do Exam Rank 02

| Nível | Exercícios |
|-------|------------|
| 1 (12) | first_word, fizzbuzz, ft_putstr, ft_strcpy, ft_strlen, ft_swap, repeat_alpha, rev_print, rot_13, rotone, search_and_replace, ulstr |
| 2 (19) | alpha_mirror, camel_to_snake, do_op, ft_atoi, ft_strcmp, ft_strcspn, ft_strdup, ft_strpbrk, ft_strrev, ft_strspn, is_power_of_2, last_word, max, print_bits, reverse_bits, snake_to_camel, swap_bits, union, wdmatch |
| 3 (15) | add_prime_sum, epur_str, expand_str, ft_atoi_base, ft_list_size, ft_range, ft_rrange, hidenp, lcm, paramsum, pgcd, print_hex, rstr_capitalizer, str_capitalizer, tab_mult |
| 4 (10) | flood_fill, fprime, ft_itoa, ft_list_foreach, ft_list_remove_if, ft_split, rev_wstr, rostring, sort_int_tab, sort_list |

## Os exercícios em Python (Ranks 03 a 05)

| Rank | Nível | Exercícios |
|------|-------|------------|
| 03 | 1 | py_bracket_validator, py_cryptic_sorter |
| 03 | 2 | py_echo_validator, py_mirror_matrix |
| 03 | 3 | py_hidenp, py_inter, py_number_base_converter, py_pattern_tracker |
| 03 | 4 | py_anagram, py_shadow_merge, py_string_permutation_checker |
| 03 | 5 | py_string_sculptor, py_twist_sequence |
| 03 | 6 | py_whisper_cipher |
| 04 | 1 | py_array_rotation_detector, py_constellation_mapper |
| 04 | 2 | py_list_intersection_finder, py_merge_sorted_lists |
| 04 | 3 | py_package_dependency_resolver, py_palindrome_partitioner |
| 04 | 4 | py_sliding_window_maximum |
| 05 | 1 | py_compress_decompress, py_spiral_matrix |
| 05 | 2 | py_graph_cycle_detector, py_island_matrix_counter, py_room_scheduler |
| 05 | 3 | py_prism_detector, py_word_ladder |

A lista vem do [42exam.net](https://42exam.net/); os enunciados, os testes e as soluções de referência foram escritos para o lightyear.

Quer adicionar exercícios ou outros ranks? Veja [Contribuir](contribuir.html#exercicios).
