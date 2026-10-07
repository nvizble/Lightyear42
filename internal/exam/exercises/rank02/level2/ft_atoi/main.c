#include <stdio.h>

int	ft_atoi(const char *str);

int	main(int argc, char **argv)
{
	int	i;

	i = 1;
	while (i < argc)
	{
		printf("%d\n", ft_atoi(argv[i]));
		i++;
	}
	return (0);
}
