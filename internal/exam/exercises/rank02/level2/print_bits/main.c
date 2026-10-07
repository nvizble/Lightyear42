#include <stdlib.h>
#include <unistd.h>

void	print_bits(unsigned char octet);

/* Prints only with write() so the order matches the student's write() calls. */
int	main(int argc, char **argv)
{
	int	i;

	i = 1;
	while (i < argc)
	{
		print_bits((unsigned char)atoi(argv[i]));
		write(1, "|\n", 2);
		i++;
	}
	return (0);
}
