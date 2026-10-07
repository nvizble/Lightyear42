#include <stdio.h>

int	ft_strcmp(char *s1, char *s2);

/* Only the sign of strcmp is specified, so only the sign is printed. */
int	main(int argc, char **argv)
{
	int	i;
	int	r;

	i = 1;
	while (i + 1 < argc)
	{
		r = ft_strcmp(argv[i], argv[i + 1]);
		printf("%d\n", (r > 0) - (r < 0));
		i += 2;
	}
	return (0);
}
