#include "helpers.h"

void	lt_suite_ft_isdigit(void)
{
	lt_group("ft_isdigit");
	check_class(ft_isdigit, want_digit);
}
