#include "lists.h"

static int		g_dels;
static void		*g_last;

static void	del(void *content)
{
	g_dels++;
	g_last = content;
	free(content);
}

void	lt_suite_ft_lstdelone(void)
{
	lt_group("ft_lstdelone");
	TEST("chama del uma vez com o conteúdo e libera o nó")
	{
		char	*s = text("hello");
		t_list	*n = node(s, NULL);

		ft_lstdelone(n, del);
		CHECK(g_dels == 1, "del foi chamada %d vezes", g_dels);
		CHECK(g_last == s, "del não recebeu o conteúdo do nó");
	}
	TEST("não mexe no próximo nó")
	{
		t_list	*next = node(text("b"), NULL);
		t_list	*n = node(text("a"), next);

		ft_lstdelone(n, del);
		CHECK(g_dels == 1, "del foi chamada %d vezes", g_dels);
		CHECK(strcmp(next->content, "b") == 0, "o próximo nó foi alterado");
		free(next->content);
		free(next);
	}
}
