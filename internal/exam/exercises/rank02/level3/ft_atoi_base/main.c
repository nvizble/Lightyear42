#include <stdio.h>
#include <stdlib.h>

int	ft_atoi_base(const char *str, int str_base);

/* argv holds (str, base) pairs; prints one result per line. */
int	main(int argc, char **argv)
{
	int	i;

	i = 1;
	while (i + 1 < argc)
	{
		printf("%d\n", ft_atoi_base(argv[i], atoi(argv[i + 1])));
		i += 2;
	}
	return (0);
}
