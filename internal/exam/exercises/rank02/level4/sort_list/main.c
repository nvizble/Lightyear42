#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "list.h"

t_list	*sort_list(t_list *lst, int (*cmp)(int, int));

static int	ly_ascending(int a, int b)
{
	return (a <= b);
}

static int	ly_descending(int a, int b)
{
	return (a >= b);
}

/* Builds a list holding the ints of argv[1..argc-1], in order. */
static t_list	*ly_build(int argc, char **argv)
{
	t_list	*head;
	t_list	*node;
	int		i;

	head = NULL;
	i = argc - 1;
	while (i >= 1)
	{
		node = malloc(sizeof(t_list));
		node->data = (int)strtol(argv[i], NULL, 10);
		node->next = head;
		head = node;
		i--;
	}
	return (head);
}

static void	ly_print_free(const char *label, t_list *lst)
{
	t_list	*next;

	printf("%s:", label);
	while (lst)
	{
		printf(" %d", lst->data);
		next = lst->next;
		free(lst);
		lst = next;
	}
	printf("\n");
}

/* Sorts the same input once with each cmp and prints the results. */
int	main(int argc, char **argv)
{
	if (argc < 2)
		return (1);
	ly_print_free("ascending", sort_list(ly_build(argc, argv), ly_ascending));
	ly_print_free("descending", sort_list(ly_build(argc, argv), ly_descending));
	return (0);
}
