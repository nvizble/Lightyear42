#include "helpers.h"

static void	*call(void)
{
	return (ft_strjoin("hello", " world"));
}

void	lt_suite_ft_strjoin(void)
{
	static const struct { const char *a; const char *b; } cases[] = {
		{"hello", " world"}, {"", ""}, {"", "abc"}, {"abc", ""}, {"a", "b"},
		{"lorem ipsum", " dolor sit amet"}, {"\t\n", "\x80\xff"}, {"42", "42"},
	};

	lt_group("ft_strjoin");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_strjoin(%s, %s)", lt_str(cases[i].a), lt_str(cases[i].b)))
		{
			char	want[128];
			char	*got;

			strcpy(want, cases[i].a);
			strcat(want, cases[i].b);
			got = ft_strjoin(cases[i].a, cases[i].b);
			CHECK(got != NULL, "devolveu NULL");
			CHECK_STR(got, want);
			free(got);
		}
	TEST("strings longas (50000 + 50000)")
	{
		char	*a = malloc(50001);
		char	*got;

		memcpy(a, filled('a', 50000), 50001);
		got = ft_strjoin(a, filled('b', 50000));
		CHECK(got && strlen(got) == 100000 && got[49999] == 'a' && got[50000] == 'b',
			"resultado errado");
		free(a);
		free(got);
	}
	TEST("não muda as strings de entrada")
	{
		char	a[] = "abc";
		char	b[] = "def";

		free(ft_strjoin(a, b));
		CHECK(strcmp(a, "abc") == 0 && strcmp(b, "def") == 0, "a ficou %s, b ficou %s", lt_str(a), lt_str(b));
	}
	TEST("malloc falhando devolve NULL")
		lt_alloc_fails(call, free);
}
