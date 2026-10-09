#include "helpers.h"

static void	run(size_t n, size_t off)
{
	unsigned char	buf[256];

	memset(buf, '.', sizeof(buf));
	ft_bzero(buf + off, n);
	for (size_t i = 0; i < sizeof(buf); i++)
	{
		unsigned char	want = (i >= off && i < off + n) ? 0 : '.';

		CHECK(buf[i] == want, "n = %zu, byte %zu: esperado 0x%02x, recebido 0x%02x%s", n, i, want, buf[i],
			i >= off + n ? " (zerou além de n)" : "");
	}
}

void	lt_suite_ft_bzero(void)
{
	static const size_t	sizes[] = {0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 64, 100, 200};

	lt_group("ft_bzero");
	for (size_t s = 0; s < LEN(sizes); s++)
		TEST(lt_fmt("ft_bzero(buf, %zu)", sizes[s]))
			run(sizes[s], 8);
	TEST("ponteiro desalinhado (offsets 1 a 7)")
		for (size_t off = 1; off < 8; off++)
			run(23, off);
	TEST("zera bytes que já eram texto")
	{
		char	s[] = "hello";

		ft_bzero(s, 3);
		CHECK(memcmp(s, "\0\0\0lo", 6) == 0, "recebido %s", lt_repr(s, 6));
	}
}
