#include "helpers.h"

/* run fills [off, off+n) of a guarded buffer and compares it with memset. */
static void	run(int c, size_t n, size_t off)
{
	unsigned char	got[256];
	unsigned char	want[256];
	void			*ret;

	memset(got, '.', sizeof(got));
	memset(want, '.', sizeof(want));
	ret = ft_memset(got + off, c, n);
	memset(want + off, c, n);
	CHECK(ret == got + off, "devolveu %p, esperado o próprio ponteiro %p", ret, (void *)(got + off));
	for (size_t i = 0; i < sizeof(got); i++)
		CHECK(got[i] == want[i], "byte %zu: esperado 0x%02x, recebido 0x%02x%s", i, want[i], got[i],
			i >= off + n ? " (escreveu além de n)" : i < off ? " (escreveu antes do início)" : "");
}

void	lt_suite_ft_memset(void)
{
	static const int	values[] = {'A', 0, 255, -1, 'A' + 256, 1024, -255, 127, 128};
	static const size_t	sizes[] = {0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 31, 32, 33, 100, 200};

	lt_group("ft_memset");
	for (size_t v = 0; v < LEN(values); v++)
		TEST(lt_fmt("ft_memset(buf, %d, n) para vários n", values[v]))
			for (size_t s = 0; s < LEN(sizes); s++)
				run(values[v], sizes[s], 8);
	TEST("n = 0 não escreve nada")
		run('X', 0, 0);
	TEST("ponteiro desalinhado (offsets 1 a 7)")
		for (size_t off = 1; off < 8; off++)
			run('Z', 37, off);
	TEST("devolve o ponteiro recebido")
	{
		char	buf[10];

		CHECK(ft_memset(buf, 0, 10) == buf, "não devolveu buf");
	}
	TEST("buffer grande (1 MB)")
	{
		char	*buf = malloc(1 << 20);

		ft_memset(buf, 'q', 1 << 20);
		for (size_t i = 0; i < 1 << 20; i++)
			CHECK(buf[i] == 'q', "byte %zu não foi preenchido", i);
		free(buf);
	}
}
