#include <stdio.h>
#include <stdlib.h>

int	*ft_rrange(int start, int end);

/* argv holds (start, end) pairs; prints each array on its own line. */
int	main(int argc, char **argv)
{
	int		i;
	int		start;
	int		end;
	long	len;
	long	j;
	int		*tab;

	i = 1;
	while (i + 1 < argc)
	{
		start = atoi(argv[i]);
		end = atoi(argv[i + 1]);
		len = (long)end - (long)start;
		if (len < 0)
			len = -len;
		len++;
		tab = ft_rrange(start, end);
		if (!tab)
			return (1);
		j = 0;
		while (j < len)
		{
			printf(j ? " %d" : "%d", tab[j]);
			j++;
		}
		printf("\n");
		free(tab);
		i += 2;
	}
	return (0);
}
