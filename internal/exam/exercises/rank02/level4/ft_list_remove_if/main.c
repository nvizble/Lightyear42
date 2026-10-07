#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "ft_list.h"

void	ft_list_remove_if(t_list **begin_list, void *data_ref, int (*cmp)());

static int	ly_cmp(void *a, void *b)
{
	return (strcmp((char *)a, (char *)b));
}

/*
** argv[1] is the reference data; argv[2..] become the list, in order.
** Prints what is left of the list after the removal.
*/
int	main(int argc, char **argv)
{
	t_list	*head;
	t_list	*node;
	int		i;

	if (argc < 2)
		return (1);
	head = NULL;
	i = argc - 1;
	while (i >= 2)
	{
		node = malloc(sizeof(t_list));
		node->data = argv[i];
		node->next = head;
		head = node;
		i--;
	}
	ft_list_remove_if(&head, argv[1], (int (*)())ly_cmp);
	printf("list:");
	while (head)
	{
		printf(" [%s]", (char *)head->data);
		node = head->next;
		free(head);
		head = node;
	}
	printf("\n");
	return (0);
}
