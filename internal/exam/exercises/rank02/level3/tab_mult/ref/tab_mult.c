#include <unistd.h>

static void	put_nbr(int n)
{
	char	c;

	if (n >= 10)
		put_nbr(n / 10);
	c = '0' + n % 10;
	write(1, &c, 1);
}

int	main(int argc, char **argv)
{
	int	n;
	int	i;

	if (argc != 2)
	{
		write(1, "\n", 1);
		return (0);
	}
	n = 0;
	i = 0;
	while (argv[1][i] >= '0' && argv[1][i] <= '9')
		n = n * 10 + argv[1][i++] - '0';
	i = 1;
	while (i <= 9)
	{
		put_nbr(i);
		write(1, " x ", 3);
		put_nbr(n);
		write(1, " = ", 3);
		put_nbr(i * n);
		write(1, "\n", 1);
		i++;
	}
	return (0);
}
