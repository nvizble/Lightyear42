#include "helpers.h"

static void	run(const char *s, int c)
{
	char	*got = ft_strchr(s, c);
	char	*want = strchr(s, c);

	CHECK(got == want, "esperado %s, recebido %s",
		want ? lt_fmt("s + %td", want - s) : "NULL",
		got ? lt_fmt("s + %td", got - s) : "NULL");
}

void	lt_suite_ft_strchr(void)
{
	static const struct { const char *s; int c; const char *name; } cases[] = {
		{"hello world", 'h', "primeiro caractere"},
		{"hello world", 'o', "primeira de várias ocorrências"},
		{"hello world", 'd', "último caractere"},
		{"hello world", ' ', "espaço"},
		{"hello world", 'z', "não encontrado"},
		{"hello world", '\0', "'\\0' devolve o fim da string"},
		{"hello world", 'o' + 256, "c = 'o' + 256 (convertido para char)"},
		{"hello world", 256, "c = 256 (vira '\\0')"},
		{"hello world", 'h' - 256, "c = 'h' - 256"},
		{"", 'a', "string vazia"},
		{"", '\0', "string vazia procurando '\\0'"},
		{"\xe9t\xe9", (char)0xe9, "byte não ASCII com c negativo"},
		{"\xe9t\xe9", 0xe9, "byte não ASCII com c positivo"},
		{"abc\0abc", 'c', "não passa do '\\0'"},
		{"tripouille", 't' + 256, "Tripouille: 't' + 256"},
	};

	lt_group("ft_strchr");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("%s: ft_strchr(%s, %d)", cases[i].name, lt_str(cases[i].s), cases[i].c))
			run(cases[i].s, cases[i].c);
}
