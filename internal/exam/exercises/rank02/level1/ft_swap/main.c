#include <stdio.h>
#include <stdlib.h>

void	ft_swap(int *a, int *b);

/* Arguments are read in pairs: a b [a b ...]. */
int	main(int argc, char **argv)
{
	int	i;
	int	a;
	int	b;

	i = 1;
	while (i + 1 < argc)
	{
		a = atoi(argv[i]);
		b = atoi(argv[i + 1]);
		printf("before: a=%d b=%d\n", a, b);
		ft_swap(&a, &b);
		printf("after:  a=%d b=%d\n", a, b);
		i += 2;
	}
	return (0);
}
