#include <stdio.h>
#include <stdlib.h>

char	**ft_split(char *str);

/* Splits argv[1] and prints each word between brackets, one per line. */
int	main(int argc, char **argv)
{
	char	**words;
	int		i;

	if (argc != 2)
		return (1);
	words = ft_split(argv[1]);
	if (!words)
	{
		printf("(null)\n");
		return (0);
	}
	i = 0;
	while (words[i])
	{
		printf("[%s]\n", words[i]);
		free(words[i]);
		i++;
	}
	printf("words: %d\n", i);
	free(words);
	return (0);
}
