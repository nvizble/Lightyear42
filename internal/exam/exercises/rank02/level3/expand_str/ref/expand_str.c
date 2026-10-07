#include <unistd.h>

static int	is_blank(char c)
{
	return (c == ' ' || c == '\t');
}

int	main(int argc, char **argv)
{
	char	*s;
	int		printed;

	if (argc == 2)
	{
		s = argv[1];
		printed = 0;
		while (*s)
		{
			while (is_blank(*s))
				s++;
			if (*s && printed)
				write(1, "   ", 3);
			while (*s && !is_blank(*s))
			{
				write(1, s++, 1);
				printed = 1;
			}
		}
	}
	write(1, "\n", 1);
	return (0);
}
