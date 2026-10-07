#include <stddef.h>

size_t	ft_strspn(const char *s, const char *accept)
{
	size_t	i;
	size_t	j;

	i = 0;
	while (s[i])
	{
		j = 0;
		while (accept[j] && accept[j] != s[i])
			j++;
		if (!accept[j])
			return (i);
		i++;
	}
	return (i);
}
