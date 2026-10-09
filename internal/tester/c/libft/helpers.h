/* Shared by the libft suites. Each suite is compiled on its own against the
** student's libft.h, so a wrong prototype only breaks that function's tests. */
#ifndef HELPERS_H
# define HELPERS_H

# include <limits.h>
# include <stdint.h>
# include <stdlib.h>
# include <string.h>
# include <unistd.h>
# include "lt.h"
# include "libft.h"

# define LEN(a) (sizeof(a) / sizeof(*(a)))

/* Character classes the way the subject defines them (ASCII only), so the
** expectations don't depend on the machine's locale. */
static inline int	want_alpha(int c) { return ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')); }
static inline int	want_digit(int c) { return (c >= '0' && c <= '9'); }
static inline int	want_alnum(int c) { return (want_alpha(c) || want_digit(c)); }
static inline int	want_ascii(int c) { return (c >= 0 && c <= 127); }
static inline int	want_print(int c) { return (c >= 32 && c <= 126); }

/* check_class runs one test per range of characters. Since version 19 the
** subject asks for exactly 1 or 0 (the libc may return any non-zero). */
static inline void	check_class(int (*ft)(int), int (*want)(int))
{
	static const struct { const char *name; int from; int to; } ranges[] = {
		{"EOF (-1)", -1, -1},
		{"caracteres de controle 0-31", 0, 31},
		{"espaço e pontuação 32-47", 32, 47},
		{"dígitos 0-9", '0', '9'},
		{"pontuação 58-64", 58, 64},
		{"maiúsculas A-Z", 'A', 'Z'},
		{"pontuação 91-96", 91, 96},
		{"minúsculas a-z", 'a', 'z'},
		{"pontuação e DEL 123-127", 123, 127},
		{"bytes não ASCII 128-255", 128, 255},
	};

	for (size_t r = 0; r < LEN(ranges); r++)
		TEST(ranges[r].name)
		{
			for (int c = ranges[r].from; c <= ranges[r].to; c++)
			{
				int	got = ft(c);

				CHECK(got == want(c), "c = %d (%s): esperado %d, recebido %d%s", c,
					lt_repr(&(char){(char)c}, c < 0 ? 0 : 1), want(c), got,
					got && want(c) ? " (o subject pede exatamente 1)" : "");
			}
		}
}

/* check_case does the same for toupper/tolower: only letters change. */
static inline void	check_case(int (*ft)(int), int upper)
{
	static const struct { const char *name; int from; int to; } ranges[] = {
		{"EOF (-1)", -1, -1},
		{"controle e pontuação 0-64", 0, 64},
		{"maiúsculas A-Z", 'A', 'Z'},
		{"pontuação 91-96", 91, 96},
		{"minúsculas a-z", 'a', 'z'},
		{"123-127", 123, 127},
		{"bytes não ASCII 128-255", 128, 255},
	};

	for (size_t r = 0; r < LEN(ranges); r++)
		TEST(ranges[r].name)
		{
			for (int c = ranges[r].from; c <= ranges[r].to; c++)
			{
				int	want = c;

				if (upper && c >= 'a' && c <= 'z')
					want = c - 32;
				if (!upper && c >= 'A' && c <= 'Z')
					want = c + 32;
				CHECK(ft(c) == want, "c = %d: esperado %d, recebido %d", c, want, ft(c));
			}
		}
}

/* sign is -1, 0 or 1: only the sign of comparisons is specified. */
static inline int	sign(int n) { return ((n > 0) - (n < 0)); }

static inline void	free_split(char **words)
{
	if (words == NULL)
		return ;
	for (size_t i = 0; words[i]; i++)
		free(words[i]);
	free(words);
}

/* filled returns a static buffer of n bytes of c, '\0'-terminated. */
static inline char	*filled(char c, size_t n)
{
	static char	buf[1 << 20];

	memset(buf, c, n);
	buf[n] = '\0';
	return (buf);
}

#endif
