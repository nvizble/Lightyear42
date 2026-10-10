/* get_next_line suite, compiled with the student's files and
** -D BUFFER_SIZE=n (the group names carry n). LT_PREFIX names the bonus run. */
#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/wait.h>
#include <unistd.h>
#include "lt.h"

#ifndef LT_PREFIX
# define LT_PREFIX ""
#endif

char	*get_next_line(int fd);

/* check_lines reads fd to the end and compares each line with content:
** every line ends in '\n' except a last one without it, then NULL (twice). */
static void	check_lines(int fd, const char *content, size_t len)
{
	size_t	off = 0;
	int		n = 0;
	char	*line;

	while (off < len)
	{
		size_t	end = off;
		size_t	got;

		while (end < len && content[end] != '\n')
			end++;
		if (end < len)
			end++;
		line = get_next_line(fd);
		n++;
		CHECK(line != NULL, "linha %d: esperado %s, recebido NULL", n, lt_repr(content + off, end - off));
		got = strlen(line);
		CHECK(got == end - off && memcmp(line, content + off, got) == 0,
			"linha %d: esperado %s, recebido %s", n, lt_repr(content + off, end - off), lt_repr(line, got));
		free(line);
		off = end;
	}
	line = get_next_line(fd);
	CHECK(line == NULL, "depois da última linha, esperado NULL, recebido %s", lt_str(line));
	line = get_next_line(fd);
	CHECK(line == NULL, "chamar de novo depois do fim deveria devolver NULL, recebido %s", lt_str(line));
}

static void	file_case(const char *name, const char *content, size_t len)
{
	TEST(name)
	{
		int	fd = lt_tmpfile(content, len);

		check_lines(fd, content, len);
		close(fd);
	}
}

/* repeat builds n copies of c followed by tail (static buffer, grows). */
static char	*repeat(char c, size_t n, const char *tail)
{
	static char		*buf;
	static size_t	cap;
	size_t			t = strlen(tail);

	if (n + t + 1 > cap)
	{
		free(buf);
		cap = n + t + 1;
		buf = malloc(cap);
	}
	memset(buf, c, n);
	memcpy(buf + n, tail, t + 1);
	return (buf);
}

/* LIT tests a literal (its length from sizeof, so it may hold bytes like \xff). */
#define LIT(name, s) file_case(name, s, sizeof(s) - 1)

static void	null_case(const char *name, int fd)
{
	TEST(name)
	{
		char	*line = get_next_line(fd);

		CHECK(line == NULL, "esperado NULL, recebido %s", lt_str(line));
	}
}


void	lt_suite_gnl(void)
{
	static char	group[64];
	char		*s;
	size_t		bs = BUFFER_SIZE;

	snprintf(group, sizeof(group), "%sBUFFER_SIZE=%zu", LT_PREFIX, bs);
	lt_group(group);

	LIT("arquivo vazio", "");
	LIT("só um \\n", "\n");
	LIT("várias linhas vazias", "\n\n\n\n");
	LIT("um caractere", "x");
	LIT("uma linha sem \\n no fim", "abc");
	LIT("uma linha com \\n", "abc\n");
	LIT("várias linhas, a última sem \\n", "first line\nsecond\n\nfourth after an empty one\nlast");
	LIT("várias linhas terminando em \\n", "a\nb\nc\n");
	LIT("linha vazia no começo", "\nabc\ndef");
	LIT("espaços e tabs", " \t \n\t\n  x  \n");
	LIT("bytes não ASCII", "\xc3\xa9t\xc3\xa9\n\xff\xfe\x80\n");
	LIT("linhas longas e curtas alternadas", "a\nlorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor\nb\n\nc");

	{
		/* Joining one byte at a time is quadratic: a correct get_next_line
		** with BUFFER_SIZE=1 takes seconds on 100000 bytes. */
		size_t	big = bs < 32 ? 10000 : 100000;

		s = repeat('x', big, "\nshort\n");
		file_case(lt_fmt("linha de %zu caracteres", big), s, strlen(s));
		s = repeat('y', big, "");
		file_case(lt_fmt("linha de %zu caracteres sem \\n", big), s, strlen(s));
	}
	{
		static char	many[3000];
		size_t		n = 0;

		for (int i = 0; i < 1000; i++)
		{
			many[n++] = (char)('a' + i % 26);
			many[n++] = (char)('A' + i % 26);
			many[n++] = '\n';
		}
		file_case("1000 linhas curtas", many, n);
	}
	if (bs <= 100000)
	{
		static const int	deltas[] = {-1, 0, 1};
		static const char	*names[] = {"BUFFER_SIZE - 1", "BUFFER_SIZE", "BUFFER_SIZE + 1"};

		for (int d = 0; d < 3; d++)
		{
			size_t	n = bs + (size_t)deltas[d];

			if (n == 0)
				continue ;
			s = repeat('z', n, "\nnext\n");
			file_case(lt_fmt("linha de %s caracteres (%zu) com \\n", names[d], n), s, strlen(s));
			s = repeat('w', n, "");
			file_case(lt_fmt("linha de %s caracteres (%zu) sem \\n", names[d], n), s, strlen(s));
			{
				char	*two = malloc(2 * n + 3);

				memset(two, 'q', 2 * n + 2);
				two[n] = '\n';
				two[2 * n + 1] = '\n';
				two[2 * n + 2] = '\0';
				file_case(lt_fmt("duas linhas de %s caracteres", names[d]), two, 2 * n + 2);
				free(two);
			}
		}
	}

	null_case("fd inválido (-1) devolve NULL", -1);
	null_case("fd que não está aberto (4242) devolve NULL", 4242);
	TEST("fd já fechado devolve NULL")
	{
		int		fd = lt_tmpfile("abc\n", 4);
		char	*line;

		close(fd);
		line = get_next_line(fd);
		CHECK(line == NULL, "esperado NULL, recebido %s", lt_str(line));
	}
	TEST("diretório (o read falha) devolve NULL")
	{
		int		fd = open("/", O_RDONLY);
		char	*line = get_next_line(fd);

		CHECK(line == NULL, "esperado NULL, recebido %s", lt_str(line));
		close(fd);
	}
	TEST("erro de leitura no meio do arquivo: devolve NULL sem vazar")
	{
		int		fd = lt_tmpfile("first\nsecond\nthird\n", 19);
		int		dir = open("/", O_RDONLY);
		char	*line = get_next_line(fd);

		CHECK(line && strcmp(line, "first\n") == 0, "1ª linha: esperado \"first\\n\", recebido %s", lt_str(line));
		free(line);
		dup2(dir, fd);
		close(dir);
		lt_note("o fd passou a dar erro no read depois da 1ª linha");
		for (int i = 0; i < 5 && (line = get_next_line(fd)) != NULL; i++)
			free(line);
		lt_note("%s", "");
		close(fd);
	}
	TEST("lê da entrada padrão (fd 0)")
	{
		const char	*content = "from stdin\nsecond line\nno newline";
		int			fd = lt_tmpfile(content, strlen(content));

		dup2(fd, 0);
		close(fd);
		check_lines(0, content, strlen(content));
	}
	TEST("lê de um pipe que chega aos pedaços")
	{
		int		p[2];
		pid_t	pid;

		CHECK(pipe(p) == 0, "pipe falhou");
		pid = fork();
		if (pid == 0)
		{
			close(p[0]);
			(void)!write(p[1], "abc", 3);
			usleep(50000);
			(void)!write(p[1], "def\ngh", 6);
			usleep(50000);
			(void)!write(p[1], "i\nlast", 6);
			_exit(0);
		}
		close(p[1]);
		check_lines(p[0], "abcdef\nghi\nlast", 15);
		close(p[0]);
		waitpid(pid, NULL, 0);
	}
	TEST("devolve a linha sem esperar o fim do arquivo")
	{
		int		p[2];
		char	*line;

		CHECK(pipe(p) == 0, "pipe falhou");
		(void)!write(p[1], "line one\n", 9);
		lt_note("travou: o get_next_line esperou o fim do arquivo em vez de devolver a linha que já chegou");
		line = get_next_line(p[0]);
		lt_note("%s", "");
		CHECK(line && strcmp(line, "line one\n") == 0, "esperado \"line one\\n\", recebido %s", lt_str(line));
		free(line);
		close(p[1]);
		line = get_next_line(p[0]);
		CHECK(line == NULL, "depois do fim, esperado NULL, recebido %s", lt_str(line));
		close(p[0]);
	}
	TEST("cada linha é um bloco novo, que dá para liberar")
	{
		int		fd = lt_tmpfile("one\ntwo\n", 8);
		char	*a = get_next_line(fd);
		char	*b = get_next_line(fd);

		CHECK(a && b && a != b && strcmp(a, "one\n") == 0 && strcmp(b, "two\n") == 0, "linhas erradas: %s, %s",
			lt_str(a), lt_str(b));
		free(a);
		free(b);
		free(get_next_line(fd));
		close(fd);
	}
	TEST("malloc falhando em cada alocação: devolve NULL, sem crash e sem vazar")
	{
		const char	*content = "first line\nsecond\nthird line here\n";

		for (long n = 0; n < 200; n++)
		{
			int		fd = lt_tmpfile(content, strlen(content));
			char	*line;
			int		failed;

			lt_note("com o %ldº malloc devolvendo NULL", n + 1);
			lt_fail_alloc(n);
			for (int i = 0; i < 6 && (line = get_next_line(fd)) != NULL; i++)
				free(line);
			failed = lt_alloc_failed();
			lt_fail_alloc(-1);
			/* What is left to read goes, so a stash kept on purpose isn't a leak. */
			for (int i = 0; i < 6 && (line = get_next_line(fd)) != NULL; i++)
				free(line);
			close(fd);
			CHECK(lt_live() == 0, "o %ldº malloc devolveu NULL e ficaram %zu bloco(s) sem liberar", n + 1, lt_live());
			if (!failed)
				break ;
		}
		lt_note("%s", "");
	}
}
