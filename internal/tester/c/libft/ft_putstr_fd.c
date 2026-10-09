#include "helpers.h"

void	lt_suite_ft_putstr_fd(void)
{
	static const char	*cases[] = {"hello", "", "a", "hello\nworld\n", "\t\x80\xff", "42"};

	lt_group("ft_putstr_fd");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_putstr_fd(%s, fd)", lt_str(cases[i])))
		{
			size_t		n;
			const char	*out;

			ft_putstr_fd((char *)cases[i], lt_capture());
			out = lt_captured(&n);
			CHECK(n == strlen(cases[i]) && memcmp(out, cases[i], n) == 0, "escreveu %s no fd, esperado %s",
				lt_repr(out, n), lt_str(cases[i]));
		}
	TEST("string longa (100000 caracteres)")
	{
		size_t	n;

		ft_putstr_fd(filled('p', 100000), lt_capture());
		lt_captured(&n);
		CHECK(n == 100000, "escreveu %zu bytes, esperado 100000", n);
	}
}
