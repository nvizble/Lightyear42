char	*ft_strpbrk(const char *s1, const char *s2)
{
	const char	*p;

	while (*s1)
	{
		p = s2;
		while (*p)
		{
			if (*p == *s1)
				return ((char *)s1);
			p++;
		}
		s1++;
	}
	return (0);
}
