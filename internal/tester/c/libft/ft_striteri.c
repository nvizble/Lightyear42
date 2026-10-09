#include "helpers.h"

static char		*g_base;
static size_t	g_calls;

static void	upper_odd(unsigned int i, char *c)
{
	if (i % 2 && *c >= 'a' && *c <= 'z')
		*c -= 32;
}

static void	record(unsigned int i, char *c)
{
	CHECK(i == g_calls, "f recebeu o índice %u na chamada %zu", i, g_calls);
	CHECK(c == g_base + i, "f recebeu um ponteiro para uma cópia, não para s[%u]", i);
	g_calls++;
}

void	lt_suite_ft_striteri(void)
{
	lt_group("ft_striteri");
	TEST("muda a string no lugar (índices ímpares em maiúscula)")
	{
		char	s[] = "hello world";

		ft_striteri(s, upper_odd);
		CHECK_STR(s, "hElLo wOrLd");
	}
	TEST("f recebe o índice e o endereço de cada caractere, na ordem")
	{
		char	s[] = "lorem ipsum";

		g_base = s;
		ft_striteri(s, record);
		CHECK(g_calls == strlen(s), "f foi chamada %zu vezes, esperado %zu", g_calls, strlen(s));
	}
	TEST("string vazia: f não é chamada")
	{
		char	s[] = "";

		g_base = s;
		ft_striteri(s, record);
		CHECK(g_calls == 0, "f foi chamada %zu vezes", g_calls);
	}
	TEST("um caractere")
	{
		char	s[] = "a";

		g_base = s;
		ft_striteri(s, record);
		CHECK(g_calls == 1, "f foi chamada %zu vezes", g_calls);
	}
}
