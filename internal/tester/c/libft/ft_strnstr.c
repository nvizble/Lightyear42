#include "helpers.h"

/* ref is the BSD strnstr, the behavior the subject asks for. */
static char	*ref(const char *big, const char *little, size_t len)
{
	size_t	n = strlen(little);

	if (n == 0)
		return ((char *)big);
	for (size_t i = 0; big[i] && i + n <= len; i++)
		if (strncmp(big + i, little, n) == 0)
			return ((char *)big + i);
	return (NULL);
}

void	lt_suite_ft_strnstr(void)
{
	static const struct { const char *big; const char *little; size_t len; } cases[] = {
		{"hello world", "world", 11}, {"hello world", "world", 10}, {"hello world", "world", 100},
		{"hello", "", 0}, {"", "", 0}, {"", "", 5}, {"", "a", 1}, {"aaab", "aab", 4},
		{"aaab", "aab", 3}, {"abc", "abcd", 10}, {"abcabc", "cab", SIZE_MAX},
		{"lorem ipsum dolor sit amet", "ipsum", 15}, {"lorem ipsum dolor sit amet", "dolor", 15},
		{"lorem ipsum dolor sit amet", "dolor", 17}, {"lorem ipsum dolor sit amet", "dolor", 16},
		{"hello", "lo", 5}, {"hello", "lo", 4}, {"hello", "h", 1}, {"hello", "h", 0},
		{"hello", "hello", 5}, {"hello", "hello!", 6}, {"abc\0def", "def", 7},
		{"mississippi", "issip", 11}, {"mississippi", "issip", 8}, {"aaaaab", "aab", 6},
		{"MZIRIBMZIRIBMZE123", "MZIRIBMZE", 18}, {"a", "a", SIZE_MAX},
	};

	lt_group("ft_strnstr");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_strnstr(%s, %s, %zu)", lt_str(cases[i].big), lt_str(cases[i].little), cases[i].len))
		{
			char	*got = ft_strnstr(cases[i].big, cases[i].little, cases[i].len);
			char	*want = ref(cases[i].big, cases[i].little, cases[i].len);

			CHECK(got == want, "esperado %s, recebido %s",
				want ? lt_fmt("big + %td", want - cases[i].big) : "NULL",
				got ? lt_fmt("big + %td", got - cases[i].big) : "NULL");
		}
}
