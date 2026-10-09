#include <stdlib.h>
#include <unistd.h>
#include "libft.h"

char	*ft_substr(char const *s, unsigned int start, size_t len)
{
	size_t	sl = ft_strlen(s);

	if (start >= sl)
		return (ft_strdup(""));
	if (len > sl - start)
		len = sl - start;
	char	*r = malloc(len + 1);
	if (!r)
		return (NULL);
	ft_memcpy(r, s + start, len);
	r[len] = '\0';
	return (r);
}

char	*ft_strjoin(char const *s1, char const *s2)
{
	size_t	a = ft_strlen(s1);
	size_t	b = ft_strlen(s2);
	char	*r = malloc(a + b + 1);

	if (!r)
		return (NULL);
	ft_memcpy(r, s1, a);
	ft_memcpy(r + a, s2, b + 1);
	return (r);
}

char	*ft_strtrim(char const *s1, char const *set)
{
	size_t	start = 0;
	size_t	end = ft_strlen(s1);

	while (s1[start] && ft_strchr(set, s1[start]))
		start++;
	while (end > start && ft_strchr(set, s1[end - 1]))
		end--;
	return (ft_substr(s1, start, end - start));
}

static size_t	words(char const *s, char c)
{
	size_t	n = 0;

	for (size_t i = 0; s[i]; i++)
		if (s[i] != c && (i == 0 || s[i - 1] == c))
			n++;
	return (n);
}

char	**ft_split(char const *s, char c)
{
	size_t	n = words(s, c);
	char	**r = malloc((n + 1) * sizeof(char *));
	size_t	k = 0;

	if (!r)
		return (NULL);
	while (k < n)
	{
		size_t	len = 0;

		while (*s == c)
			s++;
		while (s[len] && s[len] != c)
			len++;
		r[k] = ft_substr(s, 0, len);
		if (!r[k])
		{
			while (k > 0)
				free(r[--k]);
			free(r);
			return (NULL);
		}
		s += len;
		k++;
	}
	r[n] = NULL;
	return (r);
}

char	*ft_itoa(int n)
{
	char	buf[12];
	int		i = 11;
	long	v = n;

	buf[i] = '\0';
	if (v == 0)
		buf[--i] = '0';
	if (v < 0)
		v = -v;
	while (v > 0)
	{
		buf[--i] = (char)('0' + v % 10);
		v /= 10;
	}
	if (n < 0)
		buf[--i] = '-';
	return (ft_strdup(buf + i));
}

char	*ft_strmapi(char const *s, char (*f)(unsigned int, char))
{
	char	*r = ft_strdup(s);

	if (!r)
		return (NULL);
	for (unsigned int i = 0; r[i]; i++)
		r[i] = f(i, s[i]);
	return (r);
}

void	ft_striteri(char *s, void (*f)(unsigned int, char *))
{
	for (unsigned int i = 0; s[i]; i++)
		f(i, s + i);
}

void	ft_putchar_fd(char c, int fd) { write(fd, &c, 1); }
void	ft_putstr_fd(char *s, int fd) { write(fd, s, ft_strlen(s)); }

void	ft_putendl_fd(char *s, int fd)
{
	ft_putstr_fd(s, fd);
	ft_putchar_fd('\n', fd);
}

void	ft_putnbr_fd(int n, int fd)
{
	long	v = n;

	if (v < 0)
	{
		ft_putchar_fd('-', fd);
		v = -v;
	}
	if (v >= 10)
		ft_putnbr_fd((int)(v / 10), fd);
	ft_putchar_fd((char)('0' + v % 10), fd);
}
