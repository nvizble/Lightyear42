#include "helpers.h"

static void	run(size_t n, size_t doff, size_t soff)
{
	unsigned char	src[512];
	unsigned char	dst[512];
	void			*ret;

	for (size_t i = 0; i < sizeof(src); i++)
		src[i] = (unsigned char)(i * 7 + 3);
	memset(dst, '.', sizeof(dst));
	ret = ft_memcpy(dst + doff, src + soff, n);
	CHECK(ret == dst + doff, "n = %zu: devolveu %p, esperado dst (%p)", n, ret, (void *)(dst + doff));
	for (size_t i = 0; i < sizeof(dst); i++)
	{
		unsigned char	want = (i >= doff && i < doff + n) ? src[soff + i - doff] : '.';

		CHECK(dst[i] == want, "n = %zu, byte %zu: esperado 0x%02x, recebido 0x%02x%s", n, i, want, dst[i],
			i >= doff + n ? " (escreveu além de n)" : "");
	}
}

void	lt_suite_ft_memcpy(void)
{
	static const size_t	sizes[] = {0, 1, 2, 3, 4, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 255};

	lt_group("ft_memcpy");
	for (size_t s = 0; s < LEN(sizes); s++)
		TEST(lt_fmt("copia %zu bytes", sizes[s]))
			run(sizes[s], 16, 16);
	TEST("origem e destino desalinhados")
		for (size_t d = 0; d < 8; d++)
			for (size_t s = 0; s < 8; s++)
				run(41, d, s);
	TEST("copia bytes '\\0' no meio (não para no fim da string)")
	{
		char	dst[12];

		memset(dst, '.', sizeof(dst));
		ft_memcpy(dst, "ab\0cd\0ef", 9);
		CHECK(memcmp(dst, "ab\0cd\0ef", 9) == 0 && dst[9] == '.', "recebido %s", lt_repr(dst, 12));
	}
	TEST("n = 0 devolve dst sem tocar em nada")
	{
		char	dst[4] = "abc";

		CHECK(ft_memcpy(dst, "xyz", 0) == dst && strcmp(dst, "abc") == 0, "recebido %s", lt_str(dst));
	}
	TEST("bloco grande (1 MB)")
	{
		char	*src = malloc(1 << 20);
		char	*dst = malloc(1 << 20);

		for (size_t i = 0; i < 1 << 20; i++)
			src[i] = (char)(i % 251);
		ft_memcpy(dst, src, 1 << 20);
		CHECK(memcmp(dst, src, 1 << 20) == 0, "a cópia ficou diferente");
		free(src);
		free(dst);
	}
}
