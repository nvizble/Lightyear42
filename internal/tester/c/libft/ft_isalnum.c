#include "helpers.h"

void	lt_suite_ft_isalnum(void)
{
	lt_group("ft_isalnum");
	check_class(ft_isalnum, want_alnum);
}
