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
			c = *s;
			if (c >= 'a' && c <= 'z')
				c = 'z' - (c - 'a');
			else if (c >= 'A' && c <= 'Z')
				c = 'Z' - (c - 'A');
			write(1, &c, 1);
			s++;
		}
	}
	write(1, "\n", 1);
	return (0);
}
