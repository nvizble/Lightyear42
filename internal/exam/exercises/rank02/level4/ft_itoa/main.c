#include <stdio.h>
#include <stdlib.h>

char	*ft_itoa(int nbr);

int	main(int argc, char **argv)
{
	char	*s;
	int		i;

	i = 1;
	while (i < argc)
	{
		s = ft_itoa((int)strtol(argv[i], NULL, 10));
		printf("%s\n", s ? s : "(null)");
		free(s);
		i++;
	}
	return (0);
}
