#include <stdio.h>
#include <stdlib.h>
#include <string.h>

char	*ft_strcpy(char *s1, char *s2);

/*
** Each argument is copied into a buffer pre-filled with 'X' (larger than
** needed) so a missing terminator shows up as trailing X's.
*/
int	main(int argc, char **argv)
{
	int		i;
	size_t	len;
	char	*buf;
	char	*ret;

	i = 1;
	while (i < argc)
	{
		len = strlen(argv[i]);
		buf = malloc(len + 5);
		if (!buf)
			return (1);
		memset(buf, 'X', len + 4);
		buf[len + 4] = '\0';
		ret = ft_strcpy(buf, argv[i]);
		printf("[%s] terminated: %s, returns s1: %s\n", buf,
			buf[len] == '\0' ? "yes" : "no", ret == buf ? "yes" : "no");
		free(buf);
		i++;
	}
	return (0);
}
