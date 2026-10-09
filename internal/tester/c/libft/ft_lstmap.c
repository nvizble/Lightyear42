#include "lists.h"

static int	g_maps;
static int	g_dels;
static int	g_out[2000];

/* twice doesn't allocate, so every failing malloc is one of lstmap's. */
static void	*twice(void *content)
{
	g_out[g_maps] = *(int *)content * 2;
	return (&g_out[g_maps++]);
}

static void	del_count(void *content)
{
	(void)content;
	g_dels++;
}

static void	*upper(void *content)
{
	char	*s = text(content);

	for (char *p = s; *p; p++)
		if (*p >= 'a' && *p <= 'z')
			*p -= 32;
	return (s);
}

static t_list	*g_src;

static void	*call(void)
{
	return (ft_lstmap(g_src, twice, del_count));
}

void	lt_suite_ft_lstmap(void)
{
	lt_group("ft_lstmap");
	TEST("aplica f em cada nó, numa lista nova")
	{
		t_list	*src = list(5);
		t_list	*dst = ft_lstmap(src, twice, del_count);
		int		i = 0;

		CHECK(dst != NULL && dst != src, "não devolveu uma lista nova");
		for (t_list *l = dst; l; l = l->next, i++)
			CHECK(*(int *)l->content == 2 * (i + 1), "nó %d: esperado %d, recebido %d", i, 2 * (i + 1),
				*(int *)l->content);
		CHECK(i == 5, "a lista nova tem %d nós, esperado 5", i);
		CHECK(g_dels == 0, "del foi chamada %d vezes sem nenhum erro", g_dels);
		for (t_list *l = src; l; l = l->next)
			CHECK(*(int *)l->content <= 5, "a lista original foi alterada");
		drop(src);
		drop(dst);
	}
	TEST("lista vazia devolve NULL")
		CHECK(ft_lstmap(NULL, twice, del_count) == NULL, "não devolveu NULL");
	TEST("conteúdos alocados por f não vazam")
	{
		t_list	*src = node("hello", node("world", NULL));
		t_list	*dst = ft_lstmap(src, upper, free);

		CHECK(dst && strcmp(dst->content, "HELLO") == 0 && strcmp(dst->next->content, "WORLD") == 0,
			"conteúdos errados");
		drop_all(dst);
		drop(src);
	}
	TEST("malloc falhando: devolve NULL, libera os nós e chama del em tudo que f criou")
	{
		g_src = list(4);
		for (long n = 0; n < 4; n++)
		{
			size_t	before = lt_live();
			t_list	*got;

			g_maps = 0;
			g_dels = 0;
			lt_note("com o %ldº malloc devolvendo NULL", n + 1);
			lt_fail_alloc(n);
			got = call();
			lt_fail_alloc(-1);
			CHECK(got == NULL, "não devolveu NULL");
			CHECK(lt_live() == before, "não liberou os nós já criados (%zu bloco(s))", lt_live() - before);
			CHECK(g_dels == g_maps, "f criou %d conteúdo(s), mas del só foi chamada %d vez(es)", g_maps, g_dels);
		}
		lt_note("%s", "");
		drop(g_src);
	}
}
