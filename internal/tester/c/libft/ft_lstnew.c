#include "lists.h"

static void	*call(void)
{
	return (ft_lstnew("content"));
}

void	lt_suite_ft_lstnew(void)
{
	lt_group("ft_lstnew");
	TEST("guarda o conteúdo e next = NULL")
	{
		char	*s = "hello";
		t_list	*n = ft_lstnew(s);

		CHECK(n != NULL, "devolveu NULL");
		CHECK(n->content == s, "content não é o ponteiro recebido");
		CHECK(n->next == NULL, "next não é NULL");
		CHECK(lt_block_size(n) >= sizeof(t_list), "alocou %zu bytes, um t_list tem %zu", lt_block_size(n),
			sizeof(t_list));
		free(n);
	}
	TEST("conteúdo NULL")
	{
		t_list	*n = ft_lstnew(NULL);

		CHECK(n != NULL && n->content == NULL && n->next == NULL, "nó errado");
		free(n);
	}
	TEST("não copia o conteúdo (guarda o mesmo ponteiro)")
	{
		int		x = 42;
		t_list	*n = ft_lstnew(&x);

		CHECK(n && n->content == &x, "content não é &x");
		free(n);
	}
	TEST("malloc falhando devolve NULL")
		lt_alloc_fails(call, free);
}
