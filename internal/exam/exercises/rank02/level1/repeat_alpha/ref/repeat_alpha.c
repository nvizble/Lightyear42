#include <unistd.h>

int	main(int argc, char **argv)
{
	char	*s;
	int		n;

	if (argc == 2)
	{
		s = argv[1];
		while (*s)
		{
			n = 1;
			if (*s >= 'a' && *s <= 'z')
				n = *s - 'a' + 1;
			else if (*s >= 'A' && *s <= 'Z')
				n = *s - 'A' + 1;
			while (n-- > 0)
				write(1, s, 1);
			s++;
		}
	}
	write(1, "\n", 1);
	return (0);
}
