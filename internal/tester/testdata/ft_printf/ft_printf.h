#ifndef FT_PRINTF_H
# define FT_PRINTF_H

# include <stdarg.h>

int	ft_printf(const char *format, ...);
int	pf_char(int c);
int	pf_str(const char *s);
int	pf_unsigned(unsigned long n, unsigned base, const char *digits);
int	pf_int(int n);
int	pf_ptr(void *p);

#endif
