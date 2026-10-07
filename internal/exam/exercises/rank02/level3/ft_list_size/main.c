#include <stdio.h>
#include <stdlib.h>
#include "ft_list.h"

int	ft_list_size(t_list *begin_list);

/* Builds a list with one node per argument and prints its size. */
int	main(int argc, char **argv)
{
	t_list	*head;
	t_list	*node;
	int		i;

	head = NULL;
	i = argc - 1;
	while (i >= 1)
	{
		node = malloc(sizeof(*node));
		if (!node)
			return (1);
		node->data = argv[i];
		node->next = head;
		head = node;
		i--;
	}
	printf("%d\n", ft_list_size(head));
	while (head)
	{
		node = head->next;
		free(head);
		head = node;
	}
	return (0);
}
