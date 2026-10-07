#include <stdio.h>

char	*ft_strpbrk(const char *s1, const char *s2);

/* Prints the offset of the match inside s1 (and the rest of s1), or NULL. */
int	main(int argc, char **argv)
{
	int		i;
	char	*r;

	i = 1;
	while (i + 1 < argc)
	{
		r = ft_strpbrk(argv[i], argv[i + 1]);
		if (r)
			printf("%d [%s]\n", (int)(r - argv[i]), r);
		else
			printf("NULL\n");
		i += 2;
	}
	return (0);
}
