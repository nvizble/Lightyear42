#include "helpers.h"

static const char	*g_s;
static char			g_c;

static void	*call(void)
{
	return (ft_split(g_s, g_c));
}

static void	release(void *words)
{
	free_split(words);
}

/* run splits s on c and compares with want (NULL-terminated). */
static void	run(const char *s, char c, const char **want)
{
	char	**got = ft_split(s, c);
	size_t	n = 0;

	CHECK(got != NULL, "devolveu NULL");
	while (want[n])
		n++;
	CHECK(lt_block_size(got) == 0 || lt_block_size(got) >= (n + 1) * sizeof(char *),
		"o array tem espaço para %zu ponteiros, precisa de %zu (as palavras e o NULL do fim)",
		lt_block_size(got) / sizeof(char *), n + 1);
	for (size_t i = 0; i <= n; i++)
	{
		if (want[i] == NULL)
			CHECK(got[i] == NULL, "esperado NULL na posição %zu (fim do array), recebido %s", i, lt_str(got[i]));
		else
		{
			CHECK(got[i] != NULL, "esperado %zu palavras, o array termina na posição %zu", n, i);
			CHECK(strcmp(got[i], want[i]) == 0, "palavra %zu: esperado %s, recebido %s", i, lt_str(want[i]),
				lt_str(got[i]));
		}
	}
	free_split(got);
}

void	lt_suite_ft_split(void)
{
	static const struct { const char *s; char c; const char *want[8]; } cases[] = {
		{"hello world", ' ', {"hello", "world", NULL}},
		{"  hello   world  ", ' ', {"hello", "world", NULL}},
		{"hello", ' ', {"hello", NULL}},
		{"", ' ', {NULL}},
		{"     ", ' ', {NULL}},
		{" ", ' ', {NULL}},
		{"hello", '\0', {"hello", NULL}},
		{"", '\0', {NULL}},
		{"a b c", ' ', {"a", "b", "c", NULL}},
		{",,a,,b,,", ',', {"a", "b", NULL}},
		{"lorem ipsum dolor sit amet", 'i', {"lorem ", "psum dolor s", "t amet", NULL}},
		{"xxxxhixxxx", 'x', {"hi", NULL}},
		{"\t\thello\t", '\t', {"hello", NULL}},
		{"a", 'a', {NULL}},
		{"ab", 'a', {"b", NULL}},
		{"ba", 'a', {"b", NULL}},
		{"one,two,,three,", ',', {"one", "two", "three", NULL}},
		{"  tripouille  42  ", ' ', {"tripouille", "42", NULL}},
		{"hello world", 'z', {"hello world", NULL}},
		{"\x80" "a" "\x80" "b", (char)0x80, {"a", "b", NULL}},
	};

	lt_group("ft_split");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_split(%s, %s)", lt_str(cases[i].s), lt_repr(&cases[i].c, 1)))
			run(cases[i].s, cases[i].c, cases[i].want);
	TEST("1000 palavras")
	{
		char	*s = malloc(2001);
		char	**got;
		size_t	n = 0;

		for (size_t i = 0; i < 1000; i++)
		{
			s[2 * i] = (char)('a' + i % 26);
			s[2 * i + 1] = ' ';
		}
		s[2000] = '\0';
		got = ft_split(s, ' ');
		CHECK(got != NULL, "devolveu NULL");
		while (got[n])
		{
			CHECK(strlen(got[n]) == 1 && got[n][0] == 'a' + (char)(n % 26), "palavra %zu errada: %s", n,
				lt_str(got[n]));
			n++;
		}
		CHECK(n == 1000, "esperado 1000 palavras, recebido %zu", n);
		free_split(got);
		free(s);
	}
	TEST("uma palavra de 100000 caracteres")
	{
		char	**got = ft_split(filled('w', 100000), ' ');

		CHECK(got && got[0] && strlen(got[0]) == 100000 && got[1] == NULL, "resultado errado");
		free_split(got);
	}
	TEST("malloc falhando em cada alocação: devolve NULL e libera as palavras já alocadas")
	{
		g_s = "  lorem ipsum   dolor sit ";
		g_c = ' ';
		lt_alloc_fails(call, release);
	}
	TEST("malloc falhando com string vazia")
	{
		g_s = "";
		g_c = ' ';
		lt_alloc_fails(call, release);
	}
}
