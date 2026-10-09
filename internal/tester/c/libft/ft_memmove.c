#include "helpers.h"

/* run moves n bytes from buf+src to buf+dst inside one buffer (they may
** overlap) and compares with memmove. */
static void	run(size_t dst, size_t src, size_t n)
{
	unsigned char	got[300];
	unsigned char	want[300];
	void			*ret;

	for (size_t i = 0; i < sizeof(got); i++)
		got[i] = want[i] = (unsigned char)(i * 13 + 1);
	ret = ft_memmove(got + dst, got + src, n);
	memmove(want + dst, want + src, n);
	CHECK(ret == got + dst, "devolveu %p, esperado dst (%p)", ret, (void *)(got + dst));
	for (size_t i = 0; i < sizeof(got); i++)
		CHECK(got[i] == want[i], "dst = buf+%zu, src = buf+%zu, n = %zu: byte %zu esperado 0x%02x, recebido 0x%02x",
			dst, src, n, i, want[i], got[i]);
}

void	lt_suite_ft_memmove(void)
{
	static const size_t	sizes[] = {0, 1, 2, 3, 7, 8, 9, 16, 17, 64, 100};
	static const size_t	shifts[] = {1, 2, 3, 4, 7, 8, 15, 16, 50};

	lt_group("ft_memmove");
	for (size_t s = 0; s < LEN(sizes); s++)
		TEST(lt_fmt("sem sobreposição, %zu bytes", sizes[s]))
		{
			run(150, 10, sizes[s]);
			run(10, 150, sizes[s]);
		}
	for (size_t s = 0; s < LEN(shifts); s++)
		TEST(lt_fmt("sobreposição com dst depois de src (dst = src + %zu)", shifts[s]))
			for (size_t n = 0; n <= 100; n += 7)
				run(100 + shifts[s], 100, n);
	for (size_t s = 0; s < LEN(shifts); s++)
		TEST(lt_fmt("sobreposição com dst antes de src (dst = src - %zu)", shifts[s]))
			for (size_t n = 0; n <= 100; n += 7)
				run(100, 100 + shifts[s], n);
	TEST("dst == src")
		run(42, 42, 50);
	TEST("string clássica: \"lorem ipsum\" uma posição à frente")
	{
		char	s[] = "lorem ipsum dolor";

		ft_memmove(s + 1, s, 10);
		CHECK(strcmp(s, "llorem ipsu dolor") == 0, "recebido %s", lt_str(s));
	}
	TEST("copia bytes '\\0' no meio")
	{
		char	s[10] = "ab\0cd\0ef";

		ft_memmove(s + 1, s, 8);
		CHECK(memcmp(s, "aab\0cd\0ef", 9) == 0, "recebido %s", lt_repr(s, 9));
	}
	TEST("bloco grande sobreposto (100 KB)")
	{
		char	*buf = malloc(200000);
		char	*want = malloc(200000);

		for (size_t i = 0; i < 200000; i++)
			buf[i] = want[i] = (char)(i % 241);
		ft_memmove(buf + 3, buf, 100000);
		memmove(want + 3, want, 100000);
		CHECK(memcmp(buf, want, 200000) == 0, "o resultado ficou diferente do memmove");
		ft_memmove(buf, buf + 5, 100000);
		memmove(want, want + 5, 100000);
		CHECK(memcmp(buf, want, 200000) == 0, "o resultado (para trás) ficou diferente do memmove");
		free(buf);
		free(want);
	}
}
