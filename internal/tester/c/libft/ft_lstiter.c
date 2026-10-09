#include "lists.h"

static int	g_seen[16];
static int	g_calls;

static void	visit(void *content)
{
	g_seen[g_calls++] = *(int *)content;
	*(int *)content *= 10;
}

void	lt_suite_ft_lstiter(void)
{
	lt_group("ft_lstiter");
	TEST("chama f em cada nó, na ordem")
	{
		t_list	*l = list(5);

		ft_lstiter(l, visit);
		CHECK(g_calls == 5, "f foi chamada %d vezes", g_calls);
		for (int i = 0; i < 5; i++)
			CHECK(g_seen[i] == i + 1, "a %dª chamada recebeu %d", i + 1, g_seen[i]);
		CHECK(*(int *)l->next->content == 20, "f não mudou o conteúdo de verdade");
		drop(l);
	}
	TEST("lista vazia: f não é chamada")
	{
		ft_lstiter(NULL, visit);
		CHECK(g_calls == 0, "f foi chamada %d vezes", g_calls);
	}
	TEST("lista com 1 nó")
	{
		t_list	*l = list(1);

		ft_lstiter(l, visit);
		CHECK(g_calls == 1 && g_seen[0] == 1, "f foi chamada %d vezes", g_calls);
		drop(l);
	}
}
