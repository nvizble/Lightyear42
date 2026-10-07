#include <unistd.h>

int	main(int argc, char **argv)
{
	char	*s;

	if (argc == 2)
	{
		s = argv[1];
		while (*s == ' ' || *s == '\t')
			s++;
		while (*s && *s != ' ' && *s != '\t')
			write(1, s++, 1);
	}
	write(1, "\n", 1);
	return (0);
}
