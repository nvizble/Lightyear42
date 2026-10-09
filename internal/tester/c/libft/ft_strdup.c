#include "helpers.h"

static const char	*g_s;

static void	*call(void)
{
	return (ft_strdup(g_s));
}

void	lt_suite_ft_strdup(void)
{
	static const char	*cases[] = {"", "a", "hello", "hello world", "\t\n\x80\xff",
		"lorem ipsum dolor sit amet"};

	lt_group("ft_strdup");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_strdup(%s)", lt_str(cases[i])))
		{
			char	*d = ft_strdup(cases[i]);

			CHECK(d != NULL, "devolveu NULL");
			CHECK(d != cases[i], "devolveu o próprio ponteiro, sem copiar");
			CHECK_STR(d, cases[i]);
			CHECK(lt_block_size(d) >= strlen(cases[i]) + 1, "alocou %zu bytes, precisava de %zu",
				lt_block_size(d), strlen(cases[i]) + 1);
			free(d);
		}
	TEST("string longa (100000 caracteres)")
	{
		char	*d = ft_strdup(filled('z', 100000));

		CHECK(d && strlen(d) == 100000 && d[99999] == 'z', "cópia errada");
		free(d);
	}
	TEST("a cópia é independente da original")
	{
		char	s[] = "hello";
		char	*d = ft_strdup(s);

		s[0] = 'j';
		CHECK(d && d[0] == 'h', "mudar a original mudou a cópia");
		free(d);
	}
	TEST("malloc falhando devolve NULL")
	{
		g_s = "hello";
		lt_alloc_fails(call, free);
	}
}
