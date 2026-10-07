#include <stdio.h>
#include <stdlib.h>

int	is_power_of_2(unsigned int n);

int	main(int argc, char **argv)
{
	unsigned int	n;
	int				i;

	i = 1;
	while (i < argc)
	{
		n = (unsigned int)strtoul(argv[i], NULL, 10);
		printf("%u: %d\n", n, is_power_of_2(n));
		i++;
	}
	return (0);
}
