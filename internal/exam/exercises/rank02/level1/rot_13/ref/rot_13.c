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
			if (c >= 'a' && c <= 'z')
				c = (c - 'a' + 13) % 26 + 'a';
			else if (c >= 'A' && c <= 'Z')
				c = (c - 'A' + 13) % 26 + 'A';
			write(1, &c, 1);
		}
	}
	write(1, "\n", 1);
	return (0);
}
