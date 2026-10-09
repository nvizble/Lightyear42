#include "helpers.h"

static size_t	g_calls;
static char		g_seen[64];

static char	add_index(unsigned int i, char c)
{
	return ((char)(c + (char)i));
}

static char	record(unsigned int i, char c)
{
	CHECK(i == g_calls, "f recebeu o índice %u na chamada %zu", i, g_calls);
	g_seen[g_calls++] = c;
	return ((char)(c >= 'a' && c <= 'z' ? c - 32 : c));
}

static void	*call(void)
{
	return (ft_strmapi("hello", add_index));
}

void	lt_suite_ft_strmapi(void)
{
	static const struct { const char *s; const char *want; } cases[] = {
		{"", ""}, {"a", "a"}, {"aaaa", "abcd"}, {"0000000000", "0123456789"}, {"abc", "ace"},
	};

	lt_group("ft_strmapi");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_strmapi(%s, c + i)", lt_str(cases[i].s)))
		{
			char	*got = ft_strmapi(cases[i].s, add_index);

			CHECK(got != NULL, "devolveu NULL");
			CHECK_STR(got, cases[i].want);
			free(got);
		}
	TEST("f é chamada uma vez por caractere, na ordem, com o índice certo")
	{
		const char	*s = "hello world";
		char		*got = ft_strmapi(s, record);

		CHECK(g_calls == strlen(s), "f foi chamada %zu vezes, esperado %zu", g_calls, strlen(s));
		CHECK(memcmp(g_seen, s, strlen(s)) == 0, "f recebeu %s", lt_repr(g_seen, g_calls));
		CHECK_STR(got, "HELLO WORLD");
		free(got);
	}
	TEST("string vazia: f não é chamada")
	{
		free(ft_strmapi("", record));
		CHECK(g_calls == 0, "f foi chamada %zu vezes", g_calls);
	}
	TEST("não muda a string original")
	{
		char	s[] = "abc";

		free(ft_strmapi(s, record));
		CHECK(strcmp(s, "abc") == 0, "s ficou %s", lt_str(s));
	}
	TEST("malloc falhando devolve NULL")
		lt_alloc_fails(call, free);
}
