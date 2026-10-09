/* Helpers for the bonus (t_list) suites. */
#ifndef LISTS_H
# define LISTS_H

# include "helpers.h"

/* node builds a node by hand, so a list test doesn't depend on ft_lstnew. */
static inline t_list	*node(void *content, t_list *next)
{
	t_list	*n = malloc(sizeof(t_list));

	n->content = content;
	n->next = next;
	return (n);
}

/* list builds n nodes whose contents are the ints 1..n (not allocated). */
static inline t_list	*list(int n)
{
	static int	values[2000];
	t_list		*head = NULL;

	for (int i = n; i > 0; i--)
	{
		values[i - 1] = i;
		head = node(&values[i - 1], head);
	}
	return (head);
}

static inline void	drop(t_list *l)
{
	while (l)
	{
		t_list	*next = l->next;

		free(l);
		l = next;
	}
}

static inline int	count(t_list *l)
{
	int	n = 0;

	for (; l; l = l->next)
		n++;
	return (n);
}

/* drop_all frees the nodes and their contents. */
static inline void	drop_all(t_list *l)
{
	while (l)
	{
		t_list	*next = l->next;

		free(l->content);
		free(l);
		l = next;
	}
}

/* text allocates a copy of s, to be freed by del. */
static inline char	*text(const char *s)
{
	char	*p = malloc(strlen(s) + 1);

	memcpy(p, s, strlen(s) + 1);
	return (p);
}

#endif
