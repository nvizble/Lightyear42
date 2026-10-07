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
	(void)argv;
	put_nbr(argc - 1);
	write(1, "\n", 1);
	return (0);
}
