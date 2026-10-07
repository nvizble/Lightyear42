#include <unistd.h>

static int	is_blank(char c)
{
	return (c == ' ' || c == '\t');
}

static void	rcapitalize(char *s)
{
	char	c;
	int		i;

	i = 0;
	while (s[i])
	{
		c = s[i];
		if (c >= 'A' && c <= 'Z')
			c += 'a' - 'A';
		if (c >= 'a' && c <= 'z' && (s[i + 1] == '\0' || is_blank(s[i + 1])))
			c -= 'a' - 'A';
		write(1, &c, 1);
		i++;
	}
}

int	main(int argc, char **argv)
{
	int	i;

	if (argc == 1)
		write(1, "\n", 1);
	i = 1;
	while (i < argc)
	{
		rcapitalize(argv[i]);
		write(1, "\n", 1);
		i++;
	}
	return (0);
}
