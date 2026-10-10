/* ft_printf suites: each call is compared with the libc's printf, output
** and return value. ft_printf is declared here, so the student's header
** can't break the build. */
#ifndef PF_H
# define PF_H

# include <limits.h>
# include <stdint.h>
# include <stdio.h>
# include <stdlib.h>
# include <string.h>
# include <unistd.h>
# include "lt.h"

int	ft_printf(const char *format, ...);

static char	g_want[1 << 17];

/* nil turns "(nil)" (glibc) into "0x0" (macOS): the subject doesn't say
** which one %p prints for NULL, so either is accepted. */
static inline size_t	nil(const char *s, size_t n, char *out)
{
	size_t	o = 0;

	for (size_t i = 0; i < n; i++)
	{
		if (i + 5 <= n && memcmp(s + i, "(nil)", 5) == 0)
		{
			memcpy(out + o, "0x0", 3);
			o += 3;
			i += 4;
			continue ;
		}
		out[o++] = s[i];
	}
	return (o);
}

static inline void	pf_compare(const char *want, int wn, const char *got, size_t gl, int gn)
{
	static char	a[1 << 17];
	static char	b[1 << 17];
	size_t		an;
	size_t		bn;

	CHECK(gn == (int)gl, "devolveu %d, mas escreveu %zu bytes: %s", gn, gl, lt_repr(got, gl));
	if (wn >= 0 && (size_t)wn == gl && memcmp(want, got, gl) == 0)
		return ;
	an = nil(want, (size_t)wn, a);
	bn = nil(got, gl, b);
	CHECK(an == bn && memcmp(a, b, an) == 0, "esperado %s (%d), recebido %s (%d)", lt_repr(want, (size_t)wn),
		wn, lt_repr(got, gl), gn);
}

/* PF calls ft_printf and snprintf with the same arguments and compares. */
# define PF(...) do { \
	int			wn_ = snprintf(g_want, sizeof(g_want), __VA_ARGS__); \
	int			saved_ = lt_stdout_begin(); \
	int			gn_ = ft_printf(__VA_ARGS__); \
	size_t		gl_; \
	const char	*got_ = lt_stdout_end(saved_, &gl_); \
	pf_compare(g_want, wn_, got_, gl_, gn_); \
} while (0)

/* T is one test case: its name is the call. */
# define T(...) TEST("ft_printf(" #__VA_ARGS__ ")") PF(__VA_ARGS__)

#endif
