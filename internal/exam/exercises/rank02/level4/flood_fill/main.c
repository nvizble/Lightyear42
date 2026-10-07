#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "flood_fill.h"

/* Each case: a grid (rows of equal length), and the starting point. */
typedef struct s_ly_case
{
	const char	*rows[8];
	int			bx;
	int			by;
}	t_ly_case;

static const t_ly_case	g_ly_cases[] = {
	/* 0: the subject example */
	{{"11111111", "10001001", "10010001", "10110001", "11100001", NULL}, 7, 4},
	/* 1: same grid, start inside the zone of 0s on the left */
	{{"11111111", "10001001", "10010001", "10110001", "11100001", NULL}, 1, 1},
	/* 2: a single cell */
	{{"a", NULL}, 0, 0},
	/* 3: checkerboard, only diagonal neighbours share the character */
	{{"0101", "1010", "0101", "1010", NULL}, 2, 2},
	/* 4: whole grid is one zone, start at the bottom-right corner */
	{{"......", "......", "......", NULL}, 5, 2},
	/* 5: tall grid, x and y differ, mixed characters and an existing 'F' */
	{{"xxo", "xox", "oox", "oFo", "ooo", "xxo", NULL}, 1, 4},
	/* 6: winding corridor that forces going left and up */
	{{"#########", "#.......#", "#.#####.#", "#.#...#.#", "#.#.#.#.#",
		"#...#...#", "#########", NULL}, 3, 3},
	/* 7: one row, start in the middle of a run */
	{{"aaabbbaaa", NULL}, 4, 0},
};

static void	ly_print(char **tab, int h)
{
	int	i;

	i = 0;
	while (i < h)
		printf("%s\n", tab[i++]);
}

int	main(int argc, char **argv)
{
	const t_ly_case	*c;
	char			**tab;
	t_point			size;
	t_point			begin;
	int				i;

	if (argc != 2 || atoi(argv[1]) < 0
		|| atoi(argv[1]) >= (int)(sizeof(g_ly_cases) / sizeof(g_ly_cases[0])))
		return (1);
	c = &g_ly_cases[atoi(argv[1])];
	size.y = 0;
	while (c->rows[size.y])
		size.y++;
	size.x = strlen(c->rows[0]);
	tab = malloc(sizeof(char *) * size.y);
	i = 0;
	while (i < size.y)
	{
		tab[i] = strdup(c->rows[i]);
		i++;
	}
	ly_print(tab, size.y);
	printf("\n");
	begin.x = c->bx;
	begin.y = c->by;
	flood_fill(tab, size, begin);
	ly_print(tab, size.y);
	while (i > 0)
		free(tab[--i]);
	free(tab);
	return (0);
}
