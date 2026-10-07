#include <unistd.h>

void	ft_putstr(char *str);

int	main(int argc, char **argv)
{
	int	i;

	i = 1;
	while (i < argc)
	{
		write(1, "[", 1);
		ft_putstr(argv[i]);
		write(1, "]\n", 2);
		i++;
	}
	return (0);
}
