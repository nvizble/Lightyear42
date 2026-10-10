#include <unistd.h>
#include "ft_printf.h"

static int	convert(char c, va_list *ap)
{
	if (c == 'c')
		return (pf_char(va_arg(*ap, int)));
	if (c == 's')
		return (pf_str(va_arg(*ap, const char *)));
	if (c == 'p')
		return (pf_ptr(va_arg(*ap, void *)));
	if (c == 'd' || c == 'i')
		return (pf_int(va_arg(*ap, int)));
	if (c == 'u')
		return (pf_unsigned(va_arg(*ap, unsigned), 10, "0123456789"));
	if (c == 'x')
		return (pf_unsigned(va_arg(*ap, unsigned), 16, "0123456789abcdef"));
	if (c == 'X')
		return (pf_unsigned(va_arg(*ap, unsigned), 16, "0123456789ABCDEF"));
	if (c == '%')
		return (pf_char('%'));
	return (0);
}

int	ft_printf(const char *format, ...)
{
	va_list	ap;
	int		total;
	int		r;

	va_start(ap, format);
	total = 0;
	while (*format)
	{
		if (*format == '%' && format[1])
			r = convert(*++format, &ap);
		else
			r = pf_char(*format);
		if (r < 0)
			return (va_end(ap), -1);
		total += r;
		format++;
	}
	va_end(ap);
	return (total);
}
