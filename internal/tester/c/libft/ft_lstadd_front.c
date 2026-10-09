#include "lists.h"

void	lt_suite_ft_lstadd_front(void)
{
	lt_group("ft_lstadd_front");
	TEST("lista vazia: o novo nó vira a lista")
	{
		t_list	*lst = NULL;
		t_list	*n = node("a", NULL);

		ft_lstadd_front(&lst, n);
		CHECK(lst == n, "*lst não aponta para o novo nó");
		CHECK(n->next == NULL, "next do nó não é NULL");
		drop(lst);
	}
	TEST("lista com 1 nó")
	{
		t_list	*first = node("b", NULL);
		t_list	*lst = first;
		t_list	*n = node("a", NULL);

		ft_lstadd_front(&lst, n);
		CHECK(lst == n && n->next == first && first->next == NULL, "a lista ficou errada");
		drop(lst);
	}
	TEST("lista com 3 nós")
	{
		t_list	*lst = list(3);
		t_list	*second = lst;
		t_list	*n = node("x", NULL);

		ft_lstadd_front(&lst, n);
		CHECK(lst == n && n->next == second && count(lst) == 4, "a lista ficou errada");
		drop(lst);
	}
}
