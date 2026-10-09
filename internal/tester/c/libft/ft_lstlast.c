#include "lists.h"

void	lt_suite_ft_lstlast(void)
{
	lt_group("ft_lstlast");
	TEST("lista vazia devolve NULL")
		CHECK(ft_lstlast(NULL) == NULL, "não devolveu NULL");
	TEST("lista com 1 nó devolve o próprio nó")
	{
		t_list	*l = list(1);

		CHECK(ft_lstlast(l) == l, "não devolveu o nó");
		drop(l);
	}
	TEST("lista com 5 nós devolve o último")
	{
		t_list	*l = list(5);
		t_list	*last = ft_lstlast(l);

		CHECK(last && last->next == NULL && *(int *)last->content == 5, "não devolveu o 5º nó");
		drop(l);
	}
}
