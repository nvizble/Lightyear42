#include <stdio.h>
#include <stdlib.h>

unsigned int	lcm(unsigned int a, unsigned int b);

/* argv holds (a, b) pairs; prints one result per line. */
int	main(int argc, char **argv)
{
	int	i;

	i = 1;
	while (i + 1 < argc)
	{
		printf("%u\n", lcm((unsigned int)strtoul(argv[i], NULL, 10),
				(unsigned int)strtoul(argv[i + 1], NULL, 10)));
		i += 2;
	}
	return (0);
}
