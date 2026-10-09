#include <stdlib.h>
#include "libft.h"

int	ft_isalpha(int c) { return ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')); }
int	ft_isdigit(int c) { return (c >= '0' && c <= '9'); }
int	ft_isalnum(int c) { return (ft_isalpha(c) || ft_isdigit(c)); }
int	ft_isascii(int c) { return (c >= 0 && c <= 127); }
int	ft_isprint(int c) { return (c >= 32 && c <= 126); }
int	ft_toupper(int c) { return (c >= 'a' && c <= 'z' ? c - 32 : c); }
int	ft_tolower(int c) { return (c >= 'A' && c <= 'Z' ? c + 32 : c); }

size_t	ft_strlen(const char *s)
{
	size_t	n = 0;

	while (s[n])
		n++;
	return (n);
}

void	*ft_memset(void *b, int c, size_t len)
{
	unsigned char	*p = b;

	while (len--)
		*p++ = (unsigned char)c;
	return (b);
}

void	ft_bzero(void *s, size_t n) { ft_memset(s, 0, n); }

void	*ft_memcpy(void *dst, const void *src, size_t n)
{
	unsigned char		*d = dst;
	const unsigned char	*s = src;

	while (n--)
		*d++ = *s++;
	return (dst);
}

void	*ft_memmove(void *dst, const void *src, size_t len)
{
	unsigned char		*d = dst;
	const unsigned char	*s = src;

	if (d > s)
		while (len--)
			d[len] = s[len];
	else
		ft_memcpy(dst, src, len);
	return (dst);
}

size_t	ft_strlcpy(char *dst, const char *src, size_t size)
{
	size_t	len = ft_strlen(src);
	size_t	i = 0;

	if (size == 0)
		return (len);
	while (src[i] && i < size - 1)
	{
		dst[i] = src[i];
		i++;
	}
	dst[i] = '\0';
	return (len);
}

size_t	ft_strlcat(char *dst, const char *src, size_t size)
{
	size_t	dl = 0;
	size_t	sl = ft_strlen(src);
	size_t	i = 0;

	while (dl < size && dst[dl])
		dl++;
	if (dl == size)
		return (size + sl);
	while (src[i] && dl + i < size - 1)
	{
		dst[dl + i] = src[i];
		i++;
	}
	dst[dl + i] = '\0';
	return (dl + sl);
}

char	*ft_strchr(const char *s, int c)
{
	while (*s != (char)c)
		if (!*s++)
			return (NULL);
	return ((char *)s);
}

char	*ft_strrchr(const char *s, int c)
{
	const char	*last = NULL;

	do
		if (*s == (char)c)
			last = s;
	while (*s++);
	return ((char *)last);
}

int	ft_strncmp(const char *s1, const char *s2, size_t n)
{
	size_t	i = 0;

	while (i < n && (s1[i] || s2[i]))
	{
		if (s1[i] != s2[i])
			return ((unsigned char)s1[i] - (unsigned char)s2[i]);
		i++;
	}
	return (0);
}

void	*ft_memchr(const void *s, int c, size_t n)
{
	const unsigned char	*p = s;

	for (size_t i = 0; i < n; i++)
		if (p[i] == (unsigned char)c)
			return ((void *)(p + i));
	return (NULL);
}

int	ft_memcmp(const void *s1, const void *s2, size_t n)
{
	const unsigned char	*a = s1;
	const unsigned char	*b = s2;

	for (size_t i = 0; i < n; i++)
		if (a[i] != b[i])
			return (a[i] - b[i]);
	return (0);
}

char	*ft_strnstr(const char *haystack, const char *needle, size_t len)
{
	size_t	n = ft_strlen(needle);

	if (n == 0)
		return ((char *)haystack);
	for (size_t i = 0; haystack[i] && i + n <= len; i++)
		if (ft_strncmp(haystack + i, needle, n) == 0)
			return ((char *)haystack + i);
	return (NULL);
}

int	ft_atoi(const char *str)
{
	long	n = 0;
	int		sign = 1;

	while (*str == ' ' || (*str >= '\t' && *str <= '\r'))
		str++;
	if (*str == '-' || *str == '+')
		if (*str++ == '-')
			sign = -1;
	while (*str >= '0' && *str <= '9')
		n = n * 10 + (*str++ - '0');
	return ((int)(n * sign));
}

void	*ft_calloc(size_t count, size_t size)
{
	void	*p;

	if (size && count > (size_t)-1 / size)
		return (NULL);
	p = malloc(count * size);
	if (p)
		ft_bzero(p, count * size);
	return (p);
}

char	*ft_strdup(const char *s1)
{
	size_t	n = ft_strlen(s1) + 1;
	char	*d = malloc(n);

	if (d)
		ft_memcpy(d, s1, n);
	return (d);
}
