#include "helpers.h"

void	lt_suite_ft_isascii(void)
{
	lt_group("ft_isascii");
	check_class(ft_isascii, want_ascii);
}
