#include <stdio.h>
#include <stddef.h>

size_t	ft_strspn(const char *s, const char *accept);

int	main(int argc, char **argv)
{
	int	i;

	i = 1;
	while (i + 1 < argc)
	{
		printf("%zu\n", ft_strspn(argv[i], argv[i + 1]));
		i += 2;
	}
	return (0);
}
