#include "helpers.h"

/* ref is the BSD strlcat, the behavior the subject asks for. */
static size_t	ref(char *dst, const char *src, size_t size)
{
	size_t	dl = strnlen(dst, size);
	size_t	sl = strlen(src);
	size_t	n;

	if (dl == size)
		return (size + sl);
	n = sl < size - dl - 1 ? sl : size - dl - 1;
	memcpy(dst + dl, src, n);
	dst[dl + n] = '\0';
	return (dl + sl);
}

static void	run(const char *start, const char *src, size_t size)
{
	char	got[64];
	char	want[64];
	size_t	r1;
	size_t	r2;

	memset(got, 'X', sizeof(got));
	memset(want, 'X', sizeof(want));
	memcpy(got, start, strlen(start) + 1);
	memcpy(want, start, strlen(start) + 1);
	r1 = ft_strlcat(got, src, size);
	r2 = ref(want, src, size);
	CHECK(r1 == r2, "devolveu %zu, esperado %zu", r1, r2);
	for (size_t i = 0; i < sizeof(got); i++)
		CHECK(got[i] == want[i], "dst ficou %s, esperado %s (byte %zu)%s", lt_repr(got, 24), lt_repr(want, 24),
			i, i >= size ? ": escreveu além de size" : "");
}

void	lt_suite_ft_strlcat(void)
{
	static const char	*dsts[] = {"", "ab", "hello"};
	static const char	*srcs[] = {"", "x", "world", "lorem ipsum dolor"};
	static const size_t	sizes[] = {0, 1, 2, 3, 5, 6, 7, 10, 11, 20, 63};

	lt_group("ft_strlcat");
	for (size_t d = 0; d < LEN(dsts); d++)
		for (size_t s = 0; s < LEN(srcs); s++)
			for (size_t z = 0; z < LEN(sizes); z++)
				TEST(lt_fmt("dst %s + src %s, size %zu", lt_str(dsts[d]), lt_str(srcs[s]), sizes[z]))
					run(dsts[d], srcs[s], sizes[z]);
	TEST("dst sem '\\0' dentro de size: devolve size + strlen(src) e não escreve")
	{
		char	dst[10];

		memset(dst, 'a', sizeof(dst));
		CHECK(ft_strlcat(dst, "123", 5) == 8, "não devolveu 8 (size + strlen(src))");
		for (size_t i = 0; i < sizeof(dst); i++)
			CHECK(dst[i] == 'a', "escreveu no byte %zu", i);
	}
}
