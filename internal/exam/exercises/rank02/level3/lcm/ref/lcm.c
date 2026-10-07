unsigned int	lcm(unsigned int a, unsigned int b)
{
	unsigned int	x;
	unsigned int	y;
	unsigned int	t;

	if (a == 0 || b == 0)
		return (0);
	x = a;
	y = b;
	while (y)
	{
		t = x % y;
		x = y;
		y = t;
	}
	return (a / x * b);
}
