#include "lists.h"

void	lt_suite_ft_lstadd_back(void)
{
	lt_group("ft_lstadd_back");
	TEST("lista vazia: o novo nó vira a lista")
	{
		t_list	*lst = NULL;
		t_list	*n = node("a", NULL);

		ft_lstadd_back(&lst, n);
		CHECK(lst == n, "*lst não aponta para o novo nó");
		drop(lst);
	}
	TEST("lista com 1 nó")
	{
		t_list	*lst = list(1);
		t_list	*n = node("x", NULL);

		ft_lstadd_back(&lst, n);
		CHECK(lst->next == n && n->next == NULL, "o nó não foi para o fim");
		drop(lst);
	}
	TEST("lista com 3 nós: o começo não muda")
	{
		t_list	*lst = list(3);
		t_list	*head = lst;
		t_list	*n = node("x", NULL);

		ft_lstadd_back(&lst, n);
		CHECK(lst == head, "*lst mudou");
		CHECK(count(lst) == 4 && lst->next->next->next == n, "o nó não foi para o fim");
		drop(lst);
	}
	TEST("adicionar vários seguidos mantém a ordem")
	{
		t_list	*lst = NULL;
		int		v[4] = {1, 2, 3, 4};

		for (int i = 0; i < 4; i++)
			ft_lstadd_back(&lst, node(&v[i], NULL));
		for (int i = 0; i < 4; i++)
		{
			t_list	*l = lst;

			for (int k = 0; k < i; k++)
				l = l->next;
			CHECK(l && *(int *)l->content == i + 1, "posição %d: ordem errada", i);
		}
		CHECK(*(int *)lst->content == 1 && *(int *)lst->next->next->next->content == 4, "ordem errada");
		drop(lst);
	}
}
