#include "helpers.h"

void	lt_suite_ft_putchar_fd(void)
{
	static const char	cases[] = {'a', 'Z', '0', ' ', '\n', '\0', (char)0x80, (char)0xff};

	lt_group("ft_putchar_fd");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_putchar_fd(%s, fd)", lt_repr(&cases[i], 1)))
		{
			size_t		n;
			const char	*out;

			ft_putchar_fd(cases[i], lt_capture());
			out = lt_captured(&n);
			CHECK(n == 1 && out[0] == cases[i], "escreveu %s no fd, esperado %s", lt_repr(out, n),
				lt_repr(&cases[i], 1));
		}
	TEST("várias chamadas seguidas")
	{
		int			fd = lt_capture();
		size_t		n;
		const char	*out;

		for (const char *s = "hello"; *s; s++)
			ft_putchar_fd(*s, fd);
		out = lt_captured(&n);
		CHECK(n == 5 && memcmp(out, "hello", 5) == 0, "escreveu %s", lt_repr(out, n));
	}
}
