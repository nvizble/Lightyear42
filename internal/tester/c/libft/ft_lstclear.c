#include "lists.h"

static int	g_dels;

static void	del(void *content)
{
	g_dels++;
	free(content);
}

void	lt_suite_ft_lstclear(void)
{
	static const int	sizes[] = {1, 2, 5, 100};

	lt_group("ft_lstclear");
	for (size_t i = 0; i < LEN(sizes); i++)
		TEST(lt_fmt("lista com %d nós: del em todos, tudo liberado e *lst = NULL", sizes[i]))
		{
			t_list	*lst = NULL;

			for (int k = 0; k < sizes[i]; k++)
				lst = node(text("item"), lst);
			ft_lstclear(&lst, del);
			CHECK(g_dels == sizes[i], "del foi chamada %d vezes, esperado %d", g_dels, sizes[i]);
			CHECK(lst == NULL, "*lst não ficou NULL");
		}
	TEST("lista vazia")
	{
		t_list	*lst = NULL;

		ft_lstclear(&lst, del);
		CHECK(g_dels == 0 && lst == NULL, "del foi chamada %d vezes", g_dels);
	}
}
