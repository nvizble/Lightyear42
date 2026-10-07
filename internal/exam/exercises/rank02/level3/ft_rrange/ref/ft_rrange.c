#include <stdlib.h>

int	*ft_rrange(int start, int end)
{
	int	*tab;
	int	step;
	int	len;
	int	i;

	step = 1;
	if (end > start)
		step = -1;
	len = (start - end) * step + 1;
	tab = malloc(sizeof(int) * len);
	if (!tab)
		return (NULL);
	i = 0;
	while (i < len)
	{
		tab[i] = end + i * step;
		i++;
	}
	return (tab);
}
