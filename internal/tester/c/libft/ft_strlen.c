#include "helpers.h"

void	lt_suite_ft_strlen(void)
{
	static const char	*cases[] = {"", "a", "ab", "hello", "hello world",
		"\t\n\v\f\r ", "\x80\xff\xe9", "abc\0def", "1234567", "12345678", "123456789",
		"lorem ipsum dolor sit amet, consectetur adipiscing elit"};
	static const size_t	big[] = {100, 4096, 100000, 1000000};

	lt_group("ft_strlen");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_strlen(%s)", lt_str(cases[i])))
			CHECK(ft_strlen(cases[i]) == strlen(cases[i]), "esperado %zu, recebido %zu",
				strlen(cases[i]), ft_strlen(cases[i]));
	for (size_t i = 0; i < LEN(big); i++)
		TEST(lt_fmt("string de %zu caracteres", big[i]))
			CHECK(ft_strlen(filled('x', big[i])) == big[i], "esperado %zu, recebido %zu",
				big[i], ft_strlen(filled('x', big[i])));
	TEST("string que não começa alinhada na memória")
	{
		char	buf[64] = "_hello world";

		for (size_t off = 0; off < 8; off++)
			CHECK(ft_strlen(buf + off) == strlen(buf + off), "offset %zu: esperado %zu, recebido %zu",
				off, strlen(buf + off), ft_strlen(buf + off));
	}
}
