/* Generated format strings for the bonus suites: every flag set × width ×
** precision for a conversion, over several values, one test per flag set
** and conversion. Combinations the C standard leaves undefined (0 with %s,
** precision with %c...) are never generated. */
#ifndef COMBOS_H
# define COMBOS_H

# include "pf.h"

/* VAL holds one argument of any of the types the conversions take. */
typedef struct s_val
{
	char		kind;	/* 'i' int, 'u' unsigned, 's' string, 'p' pointer */
	long long	i;
	const char	*s;
	void		*p;
}	t_val;

static inline void	call(const char *fmt, t_val v)
{
	switch (v.kind)
	{
		case 'i': PF(fmt, (int)v.i); break ;
		case 'u': PF(fmt, (unsigned)v.i); break ;
		case 's': PF(fmt, v.s); break ;
		default: PF(fmt, v.p); break ;
	}
}

/* sweep tests conv with flags over widths × precisions × values; the
** failing format and value go in the note. */
static inline void	sweep(const char *flags, char conv, const char **widths, int nw,
	const char **precs, int np, const t_val *vals, int nv)
{
	TEST(lt_fmt("%%%s…%c", flags, conv))
	{
		for (int w = 0; w < nw; w++)
			for (int p = 0; p < np; p++)
				for (int v = 0; v < nv; v++)
				{
					char	fmt[64];

					snprintf(fmt, sizeof(fmt), "[%%%s%s%s%c]", flags, widths[w], precs[p], conv);
					if (vals[v].kind == 's')
						lt_note("formato %s com %s", fmt, lt_str(vals[v].s));
					else if (vals[v].kind == 'p')
						lt_note("formato %s com %p", fmt, vals[v].p);
					else
						lt_note("formato %s com %lld", fmt, vals[v].i);
					call(fmt, vals[v]);
				}
		lt_note("%s", "");
	}
}

# define N(a) ((int)(sizeof(a) / sizeof(*(a))))

static const t_val	g_ints[] = {{'i', 0, 0, 0}, {'i', 1, 0, 0}, {'i', -1, 0, 0}, {'i', 42, 0, 0},
	{'i', -42, 0, 0}, {'i', 12345, 0, 0}, {'i', INT_MAX, 0, 0}, {'i', INT_MIN, 0, 0}};
static const t_val	g_uints[] = {{'u', 0, 0, 0}, {'u', 1, 0, 0}, {'u', 42, 0, 0}, {'u', 255, 0, 0},
	{'u', 0xabcdef, 0, 0}, {'u', UINT_MAX, 0, 0}};
static const t_val	g_chars[] = {{'i', 'a', 0, 0}, {'i', '0', 0, 0}, {'i', ' ', 0, 0}};
static const t_val	g_strs[] = {{'s', 0, "hello", 0}, {'s', 0, "", 0}, {'s', 0, "a", 0},
	{'s', 0, "lorem ipsum dolor", 0}, {'s', 0, NULL, 0}};

#endif
