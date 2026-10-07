#include <stdio.h>
#include <stdlib.h>
#include <string.h>

char	*ft_strdup(char *src);

/*
** The source is overwritten after the call, so a "copy" that is really the
** original pointer prints the overwritten text instead of the original one.
*/
int	main(int argc, char **argv)
{
	int		i;
	char	*dup;

	i = 1;
	while (i < argc)
	{
		dup = ft_strdup(argv[i]);
		memset(argv[i], '#', strlen(argv[i]));
		if (!dup)
			printf("NULL\n");
		else
			printf("[%s]\n", dup);
		if (dup && dup != argv[i])
			free(dup);
		i++;
	}
	return (0);
}
