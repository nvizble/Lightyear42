#include <unistd.h>

static void	put_new(char *s, char *seen)
{
	while (*s)
	{
		if (!seen[(unsigned char)*s])
		{
			seen[(unsigned char)*s] = 1;
			write(1, s, 1);
		}
		s++;
	}
}

int	main(int argc, char **argv)
{
	char	seen[256];
	int		i;

	if (argc == 3)
	{
		i = 0;
		while (i < 256)
			seen[i++] = 0;
		put_new(argv[1], seen);
		put_new(argv[2], seen);
	}
	write(1, "\n", 1);
	return (0);
}
