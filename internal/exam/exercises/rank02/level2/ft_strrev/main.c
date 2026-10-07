#include <stdio.h>

char	*ft_strrev(char *str);

/*
** Prints the returned string and the original buffer: both must hold the
** reversed text, since the reversal is in place and the parameter is returned.
*/
int	main(int argc, char **argv)
{
	int		i;
	char	*r;

	i = 1;
	while (i < argc)
	{
		r = ft_strrev(argv[i]);
		printf("[%s] [%s] %s\n", r ? r : "NULL", argv[i],
			r == argv[i] ? "same" : "other");
		i++;
	}
	return (0);
}
