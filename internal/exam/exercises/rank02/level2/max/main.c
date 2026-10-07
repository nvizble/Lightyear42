#include <stdio.h>
#include <stdlib.h>

int	max(int *tab, unsigned int len);

/* Every argument is one element of the array; no arguments means len 0. */
int	main(int argc, char **argv)
{
	int	*tab;
	int	i;

	tab = malloc(sizeof(int) * argc);
	if (!tab)
		return (1);
	i = 1;
	while (i < argc)
	{
		tab[i - 1] = atoi(argv[i]);
		i++;
	}
	printf("%d\n", max(tab, (unsigned int)(argc - 1)));
	free(tab);
	return (0);
}
