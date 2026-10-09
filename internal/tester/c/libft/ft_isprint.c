#include "helpers.h"

void	lt_suite_ft_isprint(void)
{
	lt_group("ft_isprint");
	check_class(ft_isprint, want_print);
}
