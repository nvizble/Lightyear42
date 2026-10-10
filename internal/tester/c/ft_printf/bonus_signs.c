/* Bonus: '#', ' ' and '+'. */
#include "combos.h"

void	lt_suite_bonus_signs(void)
{
	static const char	*widths[] = {"", "3", "12"};
	static const char	*precs[] = {"", ".0", ".5"};
	static const char	*sign[] = {"+", " ", "+ ", "+-", " -", "+0", " 0"};
	static const char	*alt[] = {"#", "#-", "#0"};

	lt_group("bônus: # espaço +");
	for (int f = 0; f < N(sign); f++)
	{
		sweep(sign[f], 'd', widths, N(widths), precs, N(precs), g_ints, N(g_ints));
		sweep(sign[f], 'i', widths, N(widths), precs, N(precs), g_ints, N(g_ints));
	}
	for (int f = 0; f < N(alt); f++)
	{
		sweep(alt[f], 'x', widths, N(widths), precs, N(precs), g_uints, N(g_uints));
		sweep(alt[f], 'X', widths, N(widths), precs, N(precs), g_uints, N(g_uints));
	}
	T("%+d", 0);
	T("% d", 0);
	T("%#x", 0);
	T("%#X", 0);
	T("%+d %+d", 42, -42);
	T("% d % d", 42, -42);
	T("%#x %#X", 255, 255);
	T("%+i", INT_MIN);
	T("% i", INT_MAX);
}
