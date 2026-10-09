#include "helpers.h"

void	lt_suite_ft_memcmp(void)
{
	static const struct { const char *a; const char *b; size_t n; } cases[] = {
		{"abc", "abc", 3}, {"abc", "abd", 3}, {"abd", "abc", 3}, {"abc", "abd", 2},
		{"abc", "xyz", 0}, {"\0a", "\0b", 2}, {"\0b", "\0a", 2}, {"\xff", "\x00", 1},
		{"\x00", "\xff", 1}, {"\x80", "\x7f", 1}, {"ab\0c", "ab\0d", 4},
		{"t\200", "t\0", 2}, {"zyxbcdefgh", "abcdefgxyz", 0}, {"aaaa", "aaab", 4},
	};

	lt_group("ft_memcmp");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_memcmp(%s, %s, %zu)", lt_repr(cases[i].a, cases[i].n), lt_repr(cases[i].b, cases[i].n),
			cases[i].n))
		{
			int	got = ft_memcmp(cases[i].a, cases[i].b, cases[i].n);
			int	want = memcmp(cases[i].a, cases[i].b, cases[i].n);

			CHECK(sign(got) == sign(want), "esperado um valor %s, recebido %d",
				want < 0 ? "negativo" : want > 0 ? "positivo" : "igual a 0", got);
		}
	TEST("blocos grandes que só diferem no último byte")
	{
		char	*a = malloc(10000);
		char	*b = malloc(10000);

		memset(a, 'q', 10000);
		memset(b, 'q', 10000);
		b[9999] = 'r';
		CHECK(ft_memcmp(a, b, 10000) < 0, "esperado negativo");
		CHECK(ft_memcmp(a, b, 9999) == 0, "esperado 0 com n = 9999");
		free(a);
		free(b);
	}
}
