#include <stdio.h>
#include <stdlib.h>

void	sort_int_tab(int *tab, unsigned int size);

/* Sorts the ints given as arguments and prints them on one line. */
int	main(int argc, char **argv)
{
	int	*tab;
	int	i;

	if (argc < 2)
		return (1);
	tab = malloc(sizeof(int) * (argc - 1));
	i = 0;
	while (i < argc - 1)
	{
		tab[i] = (int)strtol(argv[i + 1], NULL, 10);
		i++;
	}
	sort_int_tab(tab, argc - 1);
	i = 0;
	while (i < argc - 1)
	{
		printf(i ? " %d" : "%d", tab[i]);
		i++;
	}
	printf("\n");
	free(tab);
	return (0);
}
