#include <unistd.h>

static int	is_blank(char c)
{
	return (c == ' ' || c == '\t');
}

int	main(int argc, char **argv)
{
	char	*s;
	int		end;
	int		start;
	int		first;

	if (argc == 2)
	{
		s = argv[1];
		end = 0;
		while (s[end])
			end++;
		first = 1;
		while (end > 0)
		{
			while (end > 0 && is_blank(s[end - 1]))
				end--;
			start = end;
			while (start > 0 && !is_blank(s[start - 1]))
				start--;
			if (start < end)
			{
				if (!first)
					write(1, " ", 1);
				write(1, s + start, end - start);
				first = 0;
			}
			end = start;
		}
	}
	write(1, "\n", 1);
	return (0);
}
