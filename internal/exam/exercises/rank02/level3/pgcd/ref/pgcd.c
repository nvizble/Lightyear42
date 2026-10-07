#include <stdio.h>
#include <stdlib.h>

int	main(int argc, char **argv)
{
	int	a;
	int	b;
	int	t;

	if (argc == 3)
	{
		a = atoi(argv[1]);
		b = atoi(argv[2]);
		while (b)
		{
			t = a % b;
			a = b;
			b = t;
		}
		printf("%d", a);
	}
	printf("\n");
	return (0);
}
