#include "pf.h"

void	lt_suite_mandatory(void)
{
	char	*null_str = NULL;
	void	*null_ptr = NULL;
	int		local = 42;
	char	*big;

	lt_group("ft_printf: texto e %%");
	T("");
	T("hello world");
	T("\n");
	T("%%");
	T("%%%%");
	T("100%% sure");
	T("%% %% %%");
	T("tabs\tand\nnewlines\n");
	T("\x01\x7f\xff");

	lt_group("ft_printf: %c");
	T("%c", 'a');
	T("%c", 'Z');
	T("%c", ' ');
	T("%c", '\n');
	T("%c", 0);
	T("%c%c%c", 'a', 0, 'b');
	T("%c", 127);
	T("%c", 255);
	T("%c", -1);
	T("[%c]", '%');
	T("%c%c%c%c%c", 'h', 'e', 'l', 'l', 'o');

	lt_group("ft_printf: %s");
	T("%s", "hello");
	T("%s", "");
	T("%s", null_str);
	T("[%s]", null_str);
	T("%s%s", "a", "b");
	T("%s %s %s", "lorem", "", "ipsum");
	T("%s", "\t\n\x80\xff");
	T("%s", "%d %s %%");
	T("before %s after", "middle");
	big = malloc(20001);
	memset(big, 'x', 20000);
	big[20000] = '\0';
	T("%s", big);
	free(big);

	lt_group("ft_printf: %p");
	T("%p", &local);
	T("%p", null_ptr);
	T("[%p]", null_ptr);
	T("%p", (void *)1);
	T("%p", (void *)15);
	T("%p", (void *)16);
	T("%p", (void *)0xabcdef);
	T("%p", (void *)-1);
	T("%p", (void *)LONG_MIN);
	T("%p", (void *)LONG_MAX);
	T("%p", (void *)ULONG_MAX);
	T("%p %p", (void *)&local, null_ptr);
	T("%p", (void *)"string literal");

	lt_group("ft_printf: %d");
	T("%d", 0);
	T("%d", 1);
	T("%d", -1);
	T("%d", 9);
	T("%d", 10);
	T("%d", -10);
	T("%d", 42);
	T("%d", -42);
	T("%d", 100);
	T("%d", 123456789);
	T("%d", -123456789);
	T("%d", INT_MAX);
	T("%d", INT_MIN);
	T("%d", INT_MAX - 1);
	T("%d", INT_MIN + 1);
	T("%d%d", 1, 2);
	T("%d %d %d", INT_MIN, 0, INT_MAX);
	T("[%d]", -0);

	lt_group("ft_printf: %i");
	T("%i", 0);
	T("%i", -1);
	T("%i", 42);
	T("%i", -42);
	T("%i", INT_MAX);
	T("%i", INT_MIN);
	T("%i %i", 10, -10);
	T("%i", 1000000);

	lt_group("ft_printf: %u");
	T("%u", 0);
	T("%u", 1);
	T("%u", 42);
	T("%u", 4294967295u);
	T("%u", -1);
	T("%u", -42);
	T("%u", INT_MIN);
	T("%u", INT_MAX);
	T("%u", 2147483648u);
	T("%u %u", 10u, UINT_MAX);

	lt_group("ft_printf: %x");
	T("%x", 0);
	T("%x", 1);
	T("%x", 9);
	T("%x", 10);
	T("%x", 15);
	T("%x", 16);
	T("%x", 255);
	T("%x", 256);
	T("%x", 0xabcdef);
	T("%x", 0xdeadbeef);
	T("%x", -1);
	T("%x", INT_MIN);
	T("%x", INT_MAX);
	T("%x", UINT_MAX);
	T("%x %x", 42, 4242);

	lt_group("ft_printf: %X");
	T("%X", 0);
	T("%X", 10);
	T("%X", 15);
	T("%X", 255);
	T("%X", 0xabcdef);
	T("%X", 0xDEADBEEF);
	T("%X", -1);
	T("%X", INT_MIN);
	T("%X", UINT_MAX);
	T("%x %X", 0xabc, 0xabc);

	lt_group("ft_printf: misturado");
	T("%c%s%p%d%i%u%x%X%%", 'a', "bc", (void *)0x42, -1, 2, 3u, 255, 255);
	T("char %c, str %s, int %d, hex %x", 'q', "word", -2147483647 - 1, 3735928559u);
	T("%d%%", 50);
	T("%s: %d/%d (%u%%)", "progress", 3, 4, 75u);
	T("%d %s %d %s %d %s %d %s %d %s", 1, "a", 2, "b", 3, "c", 4, "d", 5, "e");
	T("%c%c%c%c%c%c%c%c%c%c%c%c%c%c%c%c%c%c%c%c", 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
		'n', 'o', 'p', 'q', 'r', 's', 't');
	T("%x%X%x%X", 1, 2, 3, 4);
	T("%s%s%s%s", "", "", "", "");
	T("%d\n%d\n%d\n", 1, 22, 333);

	lt_group("ft_printf: comportamento");
	TEST("sem buffer: a saída sai na hora, intercalada com write")
	{
		int			saved = lt_stdout_begin();
		size_t		len;
		const char	*got;

		ft_printf("a%d", 1);
		(void)!write(1, "|", 1);
		ft_printf("b%s", "2");
		(void)!write(1, "|", 1);
		ft_printf("c\n");
		got = lt_stdout_end(saved, &len);
		CHECK(len == 8 && memcmp(got, "a1|b2|c\n", 8) == 0, "esperado \"a1|b2|c\\n\", recebido %s (bufferizou a saída?)",
			lt_repr(got, len));
	}
	TEST("várias chamadas seguidas somam o retorno certo")
	{
		int			saved = lt_stdout_begin();
		int			total = 0;
		size_t		len;

		for (int i = 0; i < 100; i++)
			total += ft_printf("%d,", i);
		lt_stdout_end(saved, &len);
		CHECK((size_t)total == len && len == 290, "somou %d, escreveu %zu (esperado 290)", total, len);
	}
	TEST("devolve -1 quando o write falha (stdout fechado)")
	{
		int	saved = dup(1);
		int	ret;

		close(1);
		ret = ft_printf("abc %d\n", 42);
		dup2(saved, 1);
		close(saved);
		CHECK(ret == -1, "devolveu %d; o printf devolve -1 quando não consegue escrever", ret);
	}
	TEST("não vaza memória")
	{
		int		saved = lt_stdout_begin();
		size_t	len;

		ft_printf("%s %d %x %p %u %c %%\n", "leak", -42, 255, (void *)&saved, 7u, 'z');
		lt_stdout_end(saved, &len);
	}
}
