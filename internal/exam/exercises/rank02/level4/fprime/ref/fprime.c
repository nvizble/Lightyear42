#include <stdio.h>
#include <stdlib.h>

static void	fprime(int n)
{
	int	div;

	if (n == 1)
	{
		printf("1");
		return ;
	}
	div = 2;
	while (n > 1)
	{
		if (n % div == 0)
		{
			printf("%d", div);
			n /= div;
			if (n > 1)
				printf("*");
		}
		else if (div > n / div)
			div = n;
		else
			div++;
	}
}

int	main(int argc, char **argv)
{
	if (argc == 2)
		fprime(atoi(argv[1]));
	printf("\n");
	return (0);
}
