/* Bonus: '-', '0', '.' and the minimum field width, on every conversion. */
#include "combos.h"

void	lt_suite_bonus_flags(void)
{
	static const char	*widths[] = {"", "1", "2", "5", "12"};
	static const char	*precs[] = {"", ".", ".0", ".1", ".3", ".10"};
	static const char	*none[] = {""};
	static const char	*numflags[] = {"", "-", "0", "-0"};
	static const char	*sflags[] = {"", "-"};
	int					x = 42;
	const t_val			ptrs[] = {{'p', 0, 0, &x}, {'p', 0, 0, (void *)0xff}}; /* not NULL: (nil) and 0x0 pad differently */

	lt_group("bônus: - 0 . e largura");
	for (int f = 0; f < N(numflags); f++)
	{
		sweep(numflags[f], 'd', widths, N(widths), precs, N(precs), g_ints, N(g_ints));
		sweep(numflags[f], 'i', widths, N(widths), precs, N(precs), g_ints, N(g_ints));
		sweep(numflags[f], 'u', widths, N(widths), precs, N(precs), g_uints, N(g_uints));
		sweep(numflags[f], 'x', widths, N(widths), precs, N(precs), g_uints, N(g_uints));
		sweep(numflags[f], 'X', widths, N(widths), precs, N(precs), g_uints, N(g_uints));
	}
	for (int f = 0; f < N(sflags); f++)
	{
		sweep(sflags[f], 'c', widths, N(widths), none, 1, g_chars, N(g_chars));
		sweep(sflags[f], 's', widths, N(widths), precs, N(precs), g_strs, N(g_strs));
		sweep(sflags[f], 'p', widths, N(widths), none, 1, ptrs, N(ptrs));
	}
	T("[%5%]");
	T("[%-5%]");
	T("[%-5c|%5s|%-8d|%08x]", 'z', "ab", -7, 255u);
	T("[%.0d]", 0);
	T("[%5.0d]", 0);
	T("[%.0x]", 0);
	T("[%-10.3s]", "truncate me");
}
