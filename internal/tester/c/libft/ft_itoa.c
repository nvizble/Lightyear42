#include <stdio.h>
#include "helpers.h"

static void	*call(void)
{
	return (ft_itoa(-2147483647 - 1));
}

void	lt_suite_ft_itoa(void)
{
	static const int	cases[] = {0, 1, -1, 9, -9, 10, -10, 42, -42, 99, 100, -100, 101,
		1000000, -1000000, 123456789, -123456789, 2147483647, -2147483647 - 1, 2147483646,
		-2147483647, 1073741824, 7, 1001};

	lt_group("ft_itoa");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_itoa(%d)", cases[i]))
		{
			char	*got = ft_itoa(cases[i]);
			char	want[16];

			snprintf(want, sizeof(want), "%d", cases[i]);
			CHECK(got != NULL, "devolveu NULL");
			CHECK_STR(got, want);
			free(got);
		}
	TEST("malloc falhando devolve NULL")
		lt_alloc_fails(call, free);
}
