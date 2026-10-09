#include "helpers.h"

void	lt_suite_ft_strncmp(void)
{
	static const struct { const char *a; const char *b; size_t n; } cases[] = {
		{"abc", "abc", 3}, {"abc", "abd", 3}, {"abd", "abc", 3}, {"abc", "abd", 2},
		{"abc", "ab", 3}, {"ab", "abc", 3}, {"abc", "ab", 2}, {"", "", 0}, {"", "", 1},
		{"", "a", 1}, {"a", "", 1}, {"abc", "xyz", 0}, {"hello", "hellO", 10},
		{"same", "same", SIZE_MAX}, {"same", "samf", SIZE_MAX}, {"\200", "\0", 1},
		{"\0", "\200", 1}, {"test\200", "test\0", 6}, {"a\xff", "a\x01", 2},
		{"abc\0def", "abc\0xyz", 7}, {"1234", "1235", 3}, {"1234", "1235", 4},
		{"zyxbcdefgh", "abcdefgxyz", 0}, {"abcdefgh", "abcdwxyz", 4},
		{"abcdefgh", "abcdwxyz", 5}, {"test", "testss", 7}, {"testss", "test", 7},
	};

	lt_group("ft_strncmp");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_strncmp(%s, %s, %zu)", lt_str(cases[i].a), lt_str(cases[i].b), cases[i].n))
		{
			int	got = ft_strncmp(cases[i].a, cases[i].b, cases[i].n);
			int	want = strncmp(cases[i].a, cases[i].b, cases[i].n);

			CHECK(sign(got) == sign(want), "esperado um valor %s, recebido %d",
				want < 0 ? "negativo" : want > 0 ? "positivo" : "igual a 0", got);
		}
}
