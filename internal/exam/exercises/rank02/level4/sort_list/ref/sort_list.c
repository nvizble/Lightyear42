#include "list.h"

t_list	*sort_list(t_list *lst, int (*cmp)(int, int))
{
	t_list	*node;
	int		tmp;

	node = lst;
	while (node && node->next)
	{
		if (!cmp(node->data, node->next->data))
		{
			tmp = node->data;
			node->data = node->next->data;
			node->next->data = tmp;
			node = lst;
		}
		else
			node = node->next;
	}
	return (lst);
}
