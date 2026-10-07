#include <stdio.h>
#include <stdlib.h>
#include "ft_list.h"

void	ft_list_foreach(t_list *begin_list, void (*f)(void *));

static int	g_ly_calls;

static void	ly_print(void *data)
{
	g_ly_calls++;
	printf("[%s]\n", (char *)data);
}

/* Builds a list holding argv[1..argc-1] in order and walks it. */
int	main(int argc, char **argv)
{
	t_list	*head;
	t_list	*node;
	int		i;

	head = NULL;
	i = argc - 1;
	while (i >= 1)
	{
		node = malloc(sizeof(t_list));
		node->data = argv[i];
		node->next = head;
		head = node;
		i--;
	}
	ft_list_foreach(head, ly_print);
	printf("calls: %d\n", g_ly_calls);
	while (head)
	{
		node = head->next;
		free(head);
		head = node;
	}
	return (0);
}
