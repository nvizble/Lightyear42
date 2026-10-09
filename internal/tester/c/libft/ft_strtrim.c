#include "helpers.h"

static void	*call(void)
{
	return (ft_strtrim("  hello  ", " "));
}

void	lt_suite_ft_strtrim(void)
{
	static const struct { const char *s; const char *set; const char *want; } cases[] = {
		{"  hello  ", " ", "hello"}, {"xxhixx", "x", "hi"}, {"hello", "", "hello"},
		{"", "abc", ""}, {"", "", ""}, {"aaaa", "a", ""}, {"abcHELLOcba", "abc", "HELLO"},
		{"hello world", " ", "hello world"}, {" \n\t hi \t\n", " \n\t", "hi"}, {"ab", "b", "a"},
		{"ba", "b", "a"}, {"b", "b", ""}, {"lorem ipsum", "lmr", "orem ipsu"},
		{"   xxx   xxx", " x", ""}, {"abcdba", "acb", "d"}, {"hello", "xyz", "hello"},
		{"  a  ", " ", "a"}, {"\x80hi\x80", "\x80", "hi"}, {"ttripouillett", "t", "ripouille"},
		{"hello  ", " ", "hello"}, {"  hello", " ", "hello"},
	};

	lt_group("ft_strtrim");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_strtrim(%s, %s)", lt_str(cases[i].s), lt_str(cases[i].set)))
		{
			char	*got = ft_strtrim(cases[i].s, cases[i].set);

			CHECK(got != NULL, "devolveu NULL; esperado %s", lt_str(cases[i].want));
			CHECK_STR(got, cases[i].want);
			free(got);
		}
	TEST("tudo cortado: aloca só 1 byte")
	{
		char	*got = ft_strtrim("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "a");

		CHECK_STR(got, "");
		CHECK(lt_block_size(got) <= 16, "alocou %zu bytes para uma string vazia", lt_block_size(got));
		free(got);
	}
	TEST("malloc falhando devolve NULL")
		lt_alloc_fails(call, free);
}
