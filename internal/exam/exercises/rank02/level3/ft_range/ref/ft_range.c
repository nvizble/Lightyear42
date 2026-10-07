#include <stdlib.h>

int	*ft_range(int start, int end)
{
	int	*tab;
	int	step;
	int	len;
	int	i;

	step = 1;
	if (start > end)
		step = -1;
	len = (end - start) * step + 1;
	tab = malloc(sizeof(int) * len);
	if (!tab)
		return (NULL);
	i = 0;
	while (i < len)
	{
		tab[i] = start + i * step;
		i++;
	}
	return (tab);
}
