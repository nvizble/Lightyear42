#include "helpers.h"

static size_t	g_count;
static size_t	g_size;

static void	*call(void)
{
	return (ft_calloc(g_count, g_size));
}

static void	zeroed(size_t count, size_t size)
{
	unsigned char	*p = ft_calloc(count, size);

	CHECK(p != NULL, "devolveu NULL");
	CHECK(lt_block_size(p) >= count * size, "alocou %zu bytes, precisava de %zu", lt_block_size(p),
		count * size);
	for (size_t i = 0; i < count * size; i++)
		CHECK(p[i] == 0, "o byte %zu não foi zerado (0x%02x)", i, p[i]);
	free(p);
}

void	lt_suite_ft_calloc(void)
{
	static const size_t	shapes[][2] = {{1, 1}, {10, sizeof(int)}, {100, 1}, {1, 100}, {3, 7},
		{1000, 8}, {1, 1 << 20}};

	lt_group("ft_calloc");
	for (size_t i = 0; i < LEN(shapes); i++)
		TEST(lt_fmt("ft_calloc(%zu, %zu) zera a memória", shapes[i][0], shapes[i][1]))
			zeroed(shapes[i][0], shapes[i][1]);
	for (size_t i = 0; i < 3; i++)
	{
		static const size_t	zeros[][2] = {{0, 0}, {0, 10}, {10, 0}};

		TEST(lt_fmt("ft_calloc(%zu, %zu): um ponteiro único que dá para liberar", zeros[i][0], zeros[i][1]))
		{
			void	*a = ft_calloc(zeros[i][0], zeros[i][1]);
			void	*b = ft_calloc(zeros[i][0], zeros[i][1]);

			CHECK(a != NULL, "devolveu NULL; o subject pede um ponteiro único que possa ir para o free()");
			CHECK(a != b, "duas chamadas devolveram o mesmo ponteiro");
			free(a);
			free(b);
		}
	}
	TEST("ft_calloc(SIZE_MAX, 2): overflow da multiplicação devolve NULL")
	{
		void	*p = ft_calloc(SIZE_MAX, 2);

		CHECK(p == NULL, "não devolveu NULL (count * size estoura)");
	}
	TEST("ft_calloc(SIZE_MAX / 2 + 2, 2): overflow que dá um número pequeno devolve NULL")
	{
		void	*p = ft_calloc(SIZE_MAX / 2 + 2, 2);

		CHECK(p == NULL, "alocou %zu bytes: count * size estourou e virou um número pequeno",
			lt_block_size(p));
	}
	TEST("ft_calloc(2^33, 2^31): overflow que dá 0 devolve NULL")
	{
		void	*p = ft_calloc((size_t)1 << 33, (size_t)1 << 31);

		CHECK(p == NULL, "não devolveu NULL (count * size estourou e virou 0)");
	}
	TEST("ft_calloc(1, SIZE_MAX): o malloc falha e a função devolve NULL")
		CHECK(ft_calloc(1, SIZE_MAX) == NULL, "não devolveu NULL");
	TEST("malloc falhando: devolve NULL sem travar")
	{
		g_count = 10;
		g_size = 10;
		lt_alloc_fails(call, free);
	}
}
