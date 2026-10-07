#include <unistd.h>

static void	put_nbr(int n)
{
	char	c;

	if (n >= 10)
		put_nbr(n / 10);
	c = '0' + n % 10;
	write(1, &c, 1);
}

static int	is_prime(int n)
{
	int	i;

	if (n < 2)
		return (0);
	i = 2;
	while (i <= n / i)
	{
		if (n % i == 0)
			return (0);
		i++;
	}
	return (1);
}

static int	parse_positive(char *s)
{
	int	n;
	int	i;

	n = 0;
	i = 0;
	while (s[i] >= '0' && s[i] <= '9')
	{
		n = n * 10 + s[i] - '0';
		i++;
	}
	if (i == 0 || s[i] != '\0')
		return (0);
	return (n);
}

int	main(int argc, char **argv)
{
	int	n;
	int	sum;
	int	i;

	n = 0;
	if (argc == 2)
		n = parse_positive(argv[1]);
	sum = 0;
	i = 2;
	while (i <= n)
	{
		if (is_prime(i))
			sum += i;
		i++;
	}
	put_nbr(sum);
	write(1, "\n", 1);
	return (0);
}
