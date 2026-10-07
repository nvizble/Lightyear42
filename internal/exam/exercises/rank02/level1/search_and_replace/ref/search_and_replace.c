#include <unistd.h>

static int	is_single_char(char *s)
{
	return (s[0] != '\0' && s[1] == '\0');
}

int	main(int argc, char **argv)
{
	char	*s;

	if (argc == 4 && is_single_char(argv[2]) && is_single_char(argv[3]))
	{
		s = argv[1];
		while (*s)
		{
			if (*s == argv[2][0])
				write(1, &argv[3][0], 1);
			else
				write(1, s, 1);
			s++;
		}
	}
	write(1, "\n", 1);
	return (0);
}
