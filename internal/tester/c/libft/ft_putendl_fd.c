#include <stdio.h>
#include "helpers.h"

void	lt_suite_ft_putendl_fd(void)
{
	static const char	*cases[] = {"hello", "", "a", "hello world", "\t\x80\xff"};

	lt_group("ft_putendl_fd");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_putendl_fd(%s, fd)", lt_str(cases[i])))
		{
			char		want[64];
			size_t		n;
			const char	*out;

			snprintf(want, sizeof(want), "%s\n", cases[i]);
			ft_putendl_fd((char *)cases[i], lt_capture());
			out = lt_captured(&n);
			CHECK(n == strlen(want) && memcmp(out, want, n) == 0, "escreveu %s no fd, esperado %s",
				lt_repr(out, n), lt_str(want));
		}
}
