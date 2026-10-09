#include "helpers.h"

void	lt_suite_ft_atoi(void)
{
	static const char	*cases[] = {"0", "42", "-42", "+42", "   42", "\t\n\v\f\r 42",
		"42abc", "abc", "", " ", "-", "+", "--42", "+-42", "-+42", " - 42", " + 42",
		"2147483647", "-2147483648", "000042", "-0", "+0", "12 34", "\x01 42", "\x1b" "42",
		"  +0042a", "999", "-999999", "4 2", "-2147483647", "2147483646", "1",
		"9", "-9", "10", "-10", "\n\n\n  -46\b9 \n5d6", "  \t  \n  -1234ab567",
		"0000000000000000000000000000042", "+", "-+-+-+1", "  \v\f +7"};

	lt_group("ft_atoi");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("ft_atoi(%s)", lt_str(cases[i])))
			CHECK(ft_atoi(cases[i]) == atoi(cases[i]), "esperado %d, recebido %d",
				atoi(cases[i]), ft_atoi(cases[i]));
}
