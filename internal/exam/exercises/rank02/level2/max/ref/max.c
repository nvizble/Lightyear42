int	max(int *tab, unsigned int len)
{
	unsigned int	i;
	int				best;

	if (len == 0)
		return (0);
	best = tab[0];
	i = 1;
	while (i < len)
	{
		if (tab[i] > best)
			best = tab[i];
		i++;
	}
	return (best);
}
