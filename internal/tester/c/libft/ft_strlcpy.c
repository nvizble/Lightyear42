#include "helpers.h"

static void	run(const char *src, size_t size)
{
	char	got[64];
	char	want[64];
	size_t	len = strlen(src);
	size_t	ret;

	memset(got, 'X', sizeof(got));
	memset(want, 'X', sizeof(want));
	if (size > 0)
	{
		size_t	n = len < size - 1 ? len : size - 1;

		memcpy(want, src, n);
		want[n] = '\0';
	}
	ret = ft_strlcpy(got, src, size);
	CHECK(ret == len, "devolveu %zu, esperado %zu (o tamanho de src)", ret, len);
	for (size_t i = 0; i < sizeof(got); i++)
		CHECK(got[i] == want[i], "dst ficou %s, esperado %s (byte %zu)%s", lt_repr(got, 20), lt_repr(want, 20),
			i, i >= size ? ": escreveu além de size" : "");
}

void	lt_suite_ft_strlcpy(void)
{
	static const char	*srcs[] = {"", "a", "hello", "hello world 42", "\x80\xff"};
	static const size_t	sizes[] = {0, 1, 2, 5, 6, 7, 14, 15, 16, 63};

	lt_group("ft_strlcpy");
	for (size_t s = 0; s < LEN(srcs); s++)
		for (size_t z = 0; z < LEN(sizes); z++)
			TEST(lt_fmt("ft_strlcpy(dst, %s, %zu)", lt_str(srcs[s]), sizes[z]))
				run(srcs[s], sizes[z]);
	TEST("src maior que dst, size = sizeof(dst)")
	{
		char	dst[6];

		CHECK(ft_strlcpy(dst, "lorem ipsum", sizeof(dst)) == 11, "devolveu outro valor que 11");
		CHECK(strcmp(dst, "lorem") == 0, "dst ficou %s", lt_str(dst));
	}
	TEST("src longa (10000 caracteres) com size pequeno")
	{
		char	dst[8];

		CHECK(ft_strlcpy(dst, filled('a', 10000), sizeof(dst)) == 10000, "não devolveu 10000");
		CHECK(strcmp(dst, "aaaaaaa") == 0, "dst ficou %s", lt_str(dst));
	}
}
