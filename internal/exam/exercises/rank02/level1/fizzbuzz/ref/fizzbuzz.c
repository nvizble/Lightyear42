#include <unistd.h>

static void	put_num(int n)
{
	char	c;

	if (n > 9)
		put_num(n / 10);
	c = n % 10 + '0';
	write(1, &c, 1);
}

int	main(void)
{
	int	i;

	i = 1;
	while (i <= 100)
	{
		if (i % 15 == 0)
			write(1, "fizzbuzz", 8);
		else if (i % 3 == 0)
			write(1, "fizz", 4);
		else if (i % 5 == 0)
			write(1, "buzz", 4);
		else
			put_num(i);
		write(1, "\n", 1);
		i++;
	}
	return (0);
}
