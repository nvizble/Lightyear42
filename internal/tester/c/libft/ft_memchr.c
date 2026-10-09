#include "helpers.h"

void	lt_suite_ft_memchr(void)
{
	static const char	buf[] = "hello\0world\x80\xff";
	static const struct { int c; size_t n; const char *name; } cases[] = {
		{'h', 13, "primeiro byte"}, {'o', 13, "primeira ocorrência"},
		{'w', 13, "depois de um '\\0' (não para nele)"}, {'\0', 13, "o byte '\\0'"},
		{'z', 13, "não encontrado"}, {'w', 5, "fora dos n bytes"}, {'h', 0, "n = 0"},
		{'h' + 256, 13, "c = 'h' + 256 (convertido para unsigned char)"},
		{-1, 13, "c = -1 acha o byte 0xff"}, {0x80, 13, "byte 0x80"},
		{(char)0x80, 13, "c = (char)0x80 (negativo)"}, {'d', 11, "último byte dentro de n"},
		{'d', 10, "um byte além de n"},
	};

	lt_group("ft_memchr");
	for (size_t i = 0; i < LEN(cases); i++)
		TEST(lt_fmt("%s: ft_memchr(buf, %d, %zu)", cases[i].name, cases[i].c, cases[i].n))
		{
			void	*got = ft_memchr(buf, cases[i].c, cases[i].n);
			void	*want = memchr(buf, cases[i].c, cases[i].n);

			CHECK(got == want, "esperado %s, recebido %s",
				want ? lt_fmt("buf + %td", (char *)want - buf) : "NULL",
				got ? lt_fmt("buf + %td", (char *)got - buf) : "NULL");
		}
}
