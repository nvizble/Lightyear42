#include "lists.h"

void	lt_suite_ft_lstsize(void)
{
	static const int	sizes[] = {0, 1, 2, 5, 1000};

	lt_group("ft_lstsize");
	for (size_t i = 0; i < LEN(sizes); i++)
		TEST(lt_fmt("lista com %d nós", sizes[i]))
		{
			t_list	*l = list(sizes[i]);

			CHECK(ft_lstsize(l) == sizes[i], "esperado %d, recebido %d", sizes[i], ft_lstsize(l));
			drop(l);
		}
}
