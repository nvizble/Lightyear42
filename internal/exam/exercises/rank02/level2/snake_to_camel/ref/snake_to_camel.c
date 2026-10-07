#include <unistd.h>

int	main(int argc, char **argv)
{
	char	*s;
	char	c;

	if (argc == 2)
	{
		s = argv[1];
		while (*s)
		{
			c = *s++;
			if (c == '_' && *s >= 'a' && *s <= 'z')
				c = *s++ - 'a' + 'A';
			write(1, &c, 1);
		}
	}
	write(1, "\n", 1);
	return (0);
}
