#include <stdlib.h>

char	*ft_itoa(int nbr)
{
	long	n;
	int		len;
	char	*str;

	n = nbr;
	len = (n <= 0);
	while (n)
	{
		n /= 10;
		len++;
	}
	str = malloc(len + 1);
	if (!str)
		return (NULL);
	str[len] = '\0';
	n = nbr;
	if (n < 0)
		n = -n;
	while (len--)
	{
		str[len] = n % 10 + '0';
		n /= 10;
	}
	if (nbr < 0)
		str[0] = '-';
	return (str);
}
