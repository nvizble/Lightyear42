#include <stdio.h>
#include "helpers.h"

void	lt_suite_ft_putnbr_fd(void)
{
	static const int	cases[] = {0, 1, -1, 9, -9, 10, -10, 42, -42, 100, 1000000, -1000000,
		123456789, 2147483647, -2147483647 - 1, -2147483647, 2147483646, 101};

	lt_group("ft_putnbr_fd");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_putnbr_fd(%d, fd)", cases[i]))
		{
			char		want[16];
			size_t		n;
			const char	*out;

			snprintf(want, sizeof(want), "%d", cases[i]);
			ft_putnbr_fd(cases[i], lt_capture());
			out = lt_captured(&n);
			CHECK(n == strlen(want) && memcmp(out, want, n) == 0, "escreveu %s no fd, esperado %s",
				lt_repr(out, n), lt_str(want));
		}
}
