#include <stdlib.h>
#include "get_next_line_bonus.h"

size_t	gnl_len(const char *s)
{
	size_t	n;

	n = 0;
	while (s && s[n])
		n++;
	return (n);
}

char	*gnl_chr(const char *s, int c)
{
	while (s && *s)
	{
		if (*s == (char)c)
			return ((char *)s);
		s++;
	}
	return (NULL);
}

/* gnl_join frees stash; returns NULL (stash freed) when malloc fails. */
char	*gnl_join(char *stash, const char *buf, size_t n)
{
	size_t	a;
	size_t	i;
	char	*r;

	a = gnl_len(stash);
	r = malloc(a + n + 1);
	if (!r)
	{
		free(stash);
		return (NULL);
	}
	i = 0;
	while (i < a)
	{
		r[i] = stash[i];
		i++;
	}
	i = 0;
	while (i < n)
	{
		r[a + i] = buf[i];
		i++;
	}
	r[a + n] = '\0';
	free(stash);
	return (r);
}

char	*gnl_sub(const char *s, size_t start, size_t len)
{
	char	*r;
	size_t	i;

	r = malloc(len + 1);
	if (!r)
		return (NULL);
	i = 0;
	while (i < len)
	{
		r[i] = s[start + i];
		i++;
	}
	r[len] = '\0';
	return (r);
}
