#include <stdio.h>
#include <stddef.h>

size_t	ft_strcspn(const char *s, const char *reject);

int	main(int argc, char **argv)
{
	int	i;

	i = 1;
	while (i + 1 < argc)
	{
		printf("%zu\n", ft_strcspn(argv[i], argv[i + 1]));
		i += 2;
	}
	return (0);
}
