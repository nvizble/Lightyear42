#include <stdlib.h>

static int	is_sep(char c)
{
	return (c == ' ' || c == '\t' || c == '\n');
}

static int	count_words(char *str)
{
	int	count;

	count = 0;
	while (*str)
	{
		while (*str && is_sep(*str))
			str++;
		if (*str)
			count++;
		while (*str && !is_sep(*str))
			str++;
	}
	return (count);
}

static char	*word_dup(char *start, int len)
{
	char	*word;
	int		i;

	word = malloc(len + 1);
	if (!word)
		return (NULL);
	i = 0;
	while (i < len)
	{
		word[i] = start[i];
		i++;
	}
	word[i] = '\0';
	return (word);
}

char	**ft_split(char *str)
{
	char	**words;
	int		w;
	int		len;

	words = malloc(sizeof(char *) * (count_words(str) + 1));
	if (!words)
		return (NULL);
	w = 0;
	while (*str)
	{
		while (*str && is_sep(*str))
			str++;
		len = 0;
		while (str[len] && !is_sep(str[len]))
			len++;
		if (len)
			words[w++] = word_dup(str, len);
		str += len;
	}
	words[w] = NULL;
	return (words);
}
