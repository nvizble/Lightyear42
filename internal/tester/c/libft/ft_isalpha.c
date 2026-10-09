#include "helpers.h"

void	lt_suite_ft_isalpha(void)
{
	lt_group("ft_isalpha");
	check_class(ft_isalpha, want_alpha);
}
