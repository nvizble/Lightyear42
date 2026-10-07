#include <stdio.h>
#include <stdlib.h>

unsigned char	reverse_bits(unsigned char octet);

static void	put_bits(unsigned char octet)
{
	int	i;

	i = 7;
	while (i >= 0)
		putchar('0' + ((octet >> i--) & 1));
}

/* Each argument is a byte value (0-255); prints "input -> result" in binary. */
int	main(int argc, char **argv)
{
	unsigned char	in;
	int				i;

	i = 1;
	while (i < argc)
	{
		in = (unsigned char)atoi(argv[i]);
		put_bits(in);
		printf(" -> ");
		put_bits(reverse_bits(in));
		printf(" (%u)\n", (unsigned int)reverse_bits(in));
		i++;
	}
	return (0);
}
