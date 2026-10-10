/* get_next_line bonus: several file descriptors at the same time. */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include "lt.h"

char	*get_next_line(int fd);

/* round_robin reads files[i] one line per turn, each fd in sequence,
** checking every line against its own file. */
static void	round_robin(const char **files, int n)
{
	int		fds[16];
	size_t	off[16] = {0};
	int		done[16] = {0};
	int		left = n;

	for (int i = 0; i < n; i++)
		fds[i] = lt_tmpfile(files[i], strlen(files[i]));
	for (int turn = 0; left > 0 && turn < 10000; turn++)
	{
		int		i = turn % n;
		char	*line;

		if (done[i])
			continue ;
		line = get_next_line(fds[i]);
		if (files[i][off[i]] == '\0')
		{
			CHECK(line == NULL, "fd %d (arquivo %d): esperado NULL no fim, recebido %s", fds[i], i + 1, lt_str(line));
			done[i] = 1;
			left--;
			continue ;
		}
		{
			const char	*want = files[i] + off[i];
			size_t		len = strcspn(want, "\n");

			if (want[len] == '\n')
				len++;
			CHECK(line != NULL, "fd %d (arquivo %d): esperado %s, recebido NULL", fds[i], i + 1, lt_repr(want, len));
			CHECK(strlen(line) == len && memcmp(line, want, len) == 0,
				"fd %d (arquivo %d): esperado %s, recebido %s (linha de outro fd?)", fds[i], i + 1,
				lt_repr(want, len), lt_str(line));
			off[i] += len;
			free(line);
		}
	}
	for (int i = 0; i < n; i++)
		close(fds[i]);
}

void	lt_suite_gnl_bonus(void)
{
	static char	group[64];

	snprintf(group, sizeof(group), "bônus: vários fds (BUFFER_SIZE=%d)", BUFFER_SIZE);
	lt_group(group);
	TEST("3 arquivos alternando (3, 4, 5, 3, 4, 5…)")
	{
		const char	*files[] = {"a1\na2\na3\n", "b1\nb2\n", "c1\nc2\nc3\nc4"};

		round_robin(files, 3);
	}
	TEST("arquivos de tamanhos bem diferentes")
	{
		const char	*files[] = {"only one line", "", "x\ny\nz\nw\nv\nu\nt\n",
			"a much longer line than the others, to cross the buffer more than once\nend\n"};

		round_robin(files, 4);
	}
	TEST("10 arquivos ao mesmo tempo")
	{
		const char	*files[10];
		char		bufs[10][64];

		for (int i = 0; i < 10; i++)
		{
			snprintf(bufs[i], sizeof(bufs[i]), "file %d line 1\nfile %d line 2\nfile %d end", i, i, i);
			files[i] = bufs[i];
		}
		round_robin(files, 10);
	}
	TEST("um fd inválido no meio não atrapalha os outros")
	{
		int		a = lt_tmpfile("a1\na2\n", 6);
		int		b = lt_tmpfile("b1\nb2\n", 6);
		char	*l1 = get_next_line(a);
		char	*bad = get_next_line(-1);
		char	*l2 = get_next_line(b);
		char	*l3 = get_next_line(a);
		char	*l4 = get_next_line(b);

		CHECK(bad == NULL, "fd -1 deveria devolver NULL, recebido %s", lt_str(bad));
		CHECK(l1 && l2 && l3 && l4 && !strcmp(l1, "a1\n") && !strcmp(l2, "b1\n") && !strcmp(l3, "a2\n")
			&& !strcmp(l4, "b2\n"), "linhas trocadas: %s %s %s %s", lt_str(l1), lt_str(l2), lt_str(l3), lt_str(l4));
		free(l1);
		free(l2);
		free(l3);
		free(l4);
		free(get_next_line(a));
		free(get_next_line(b));
		close(a);
		close(b);
	}
}
