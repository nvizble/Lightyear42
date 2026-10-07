#include <unistd.h>

static int	is_blank(char c)
{
	return (c == ' ' || c == '\t');
}

/* Prints every word from s, one space between them; returns how many. */
static int	put_words(char *s)
{
	int	count;
	int	len;

	count = 0;
	while (*s)
	{
		while (*s && is_blank(*s))
			s++;
		len = 0;
		while (s[len] && !is_blank(s[len]))
			len++;
		if (len)
		{
			if (count++)
				write(1, " ", 1);
			write(1, s, len);
		}
		s += len;
	}
	return (count);
}

int	main(int argc, char **argv)
{
	char	*s;
	int		len;

	if (argc >= 2)
	{
		s = argv[1];
		while (*s && is_blank(*s))
			s++;
		len = 0;
		while (s[len] && !is_blank(s[len]))
			len++;
		if (put_words(s + len) && len)
			write(1, " ", 1);
		write(1, s, len);
	}
	write(1, "\n", 1);
	return (0);
}
