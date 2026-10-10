#include <unistd.h>
#include "ft_printf.h"

int	pf_char(int c)
{
	unsigned char	b;

	b = (unsigned char)c;
	return (write(1, &b, 1));
}

int	pf_str(const char *s)
{
	int	n;

	if (!s)
		s = "(null)";
	n = 0;
	while (s[n])
		n++;
	return (write(1, s, n));
}

int	pf_unsigned(unsigned long n, unsigned base, const char *digits)
{
	char	buf[32];
	int		i;

	i = 32;
	if (n == 0)
		buf[--i] = '0';
	while (n > 0)
	{
		buf[--i] = digits[n % base];
		n /= base;
	}
	return (write(1, buf + i, 32 - i));
}

int	pf_int(int n)
{
	long	v;
	int		r;

	v = n;
	if (v >= 0)
		return (pf_unsigned((unsigned long)v, 10, "0123456789"));
	if (pf_char('-') < 0)
		return (-1);
	r = pf_unsigned((unsigned long)-v, 10, "0123456789");
	if (r < 0)
		return (-1);
	return (r + 1);
}

int	pf_ptr(void *p)
{
	int	r;

	if (!p)
		return (pf_str("(nil)"));
	if (pf_str("0x") < 0)
		return (-1);
	r = pf_unsigned((unsigned long)p, 16, "0123456789abcdef");
	if (r < 0)
		return (-1);
	return (r + 2);
}
