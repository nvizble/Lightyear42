/* Test double for the bonus suites only: the libc does the formatting. */
#include <stdarg.h>
#include <stdio.h>
#include "ft_printf.h"

int	ft_printf(const char *format, ...)
{
	va_list	ap;
	int		r;

	va_start(ap, format);
	r = vdprintf(1, format, ap);
	va_end(ap);
	return (r);
}
