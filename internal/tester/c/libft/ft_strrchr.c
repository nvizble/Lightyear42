#include "helpers.h"

static void	run(const char *s, int c)
{
	char	*got = ft_strrchr(s, c);
	char	*want = strrchr(s, c);

	CHECK(got == want, "esperado %s, recebido %s",
		want ? lt_fmt("s + %td", want - s) : "NULL",
		got ? lt_fmt("s + %td", got - s) : "NULL");
}

void	lt_suite_ft_strrchr(void)
{
	static const struct { const char *s; int c; const char *name; } cases[] = {
		{"hello world", 'o', "última de várias ocorrências"},
		{"hello world", 'h', "só no primeiro caractere"},
		{"hello world", 'd', "último caractere"},
		{"hello world", 'z', "não encontrado"},
		{"hello world", '\0', "'\\0' devolve o fim da string"},
		{"hello world", 'l' + 256, "c = 'l' + 256 (convertido para char)"},
		{"hello world", 256, "c = 256 (vira '\\0')"},
		{"", 'a', "string vazia"},
		{"", '\0', "string vazia procurando '\\0'"},
		{"aaaa", 'a', "todos iguais"},
		{"a", 'a', "um caractere só"},
		{"\xe9t\xe9", (char)0xe9, "byte não ASCII com c negativo"},
		{"abc\0abc", 'a', "não olha depois do '\\0'"},
		{"tripouille", 't' + 256, "Tripouille: 't' + 256"},
	};

	lt_group("ft_strrchr");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("%s: ft_strrchr(%s, %d)", cases[i].name, lt_str(cases[i].s), cases[i].c))
			run(cases[i].s, cases[i].c);
	TEST("string longa")
	{
		char	*s = filled('x', 5000);

		s[1234] = 'y';
		CHECK(ft_strrchr(s, 'y') == s + 1234, "não achou o 'y' na posição 1234");
		CHECK(ft_strrchr(s, 'x') == s + 4999, "não achou o último 'x'");
	}
}
