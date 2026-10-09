#include "helpers.h"

static void	*call(void)
{
	return (ft_substr("hello world", 3, 5));
}

void	lt_suite_ft_substr(void)
{
	static const struct { const char *s; unsigned int start; size_t len; const char *want; } cases[] = {
		{"hello", 0, 5, "hello"}, {"hello", 0, 10, "hello"}, {"hello", 1, 3, "ell"},
		{"hello", 4, 1, "o"}, {"hello", 5, 1, ""}, {"hello", 10, 3, ""}, {"hello", 0, 0, ""},
		{"", 0, 0, ""}, {"", 0, 5, ""}, {"", 1, 1, ""}, {"hello", 2, SIZE_MAX, "llo"},
		{"hello", UINT_MAX, 3, ""}, {"hello", 0, SIZE_MAX, "hello"}, {"hello", 4, 100, "o"},
		{"lorem ipsum dolor sit amet", 6, 5, "ipsum"}, {"lorem ipsum dolor sit amet", 0, 1, "l"},
		{"lorem ipsum dolor sit amet", 22, 4, "amet"}, {"lorem ipsum dolor sit amet", 26, 4, ""},
		{"tripouille", 1, 1, "r"}, {"tripouille", 100, 1, ""},
	};

	lt_group("ft_substr");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_substr(%s, %u, %zu)", lt_str(cases[i].s), cases[i].start, cases[i].len))
		{
			char	*got = ft_substr(cases[i].s, cases[i].start, cases[i].len);

			CHECK(got != NULL, "devolveu NULL; esperado %s (uma string alocada, mesmo vazia)",
				lt_str(cases[i].want));
			CHECK_STR(got, cases[i].want);
			free(got);
		}
	TEST("len maior que o resto da string: aloca só o necessário")
	{
		char	*got = ft_substr("hello", 1, 1000000);

		CHECK_STR(got, "ello");
		CHECK(lt_block_size(got) <= 64, "alocou %zu bytes para uma substring de 4 caracteres",
			lt_block_size(got));
		free(got);
	}
	TEST("start depois do fim: aloca só 1 byte")
	{
		char	*got = ft_substr("hello", 400, 20);

		CHECK_STR(got, "");
		CHECK(lt_block_size(got) <= 16, "alocou %zu bytes para uma string vazia", lt_block_size(got));
		free(got);
	}
	TEST("o resultado não é o próprio ponteiro de s")
	{
		const char	*s = "hello";
		char		*got = ft_substr(s, 0, 5);

		CHECK(got != s, "devolveu s, sem alocar");
		free(got);
	}
	TEST("malloc falhando devolve NULL")
		lt_alloc_fails(call, free);
}
