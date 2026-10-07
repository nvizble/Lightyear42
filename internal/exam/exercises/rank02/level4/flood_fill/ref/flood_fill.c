#include "flood_fill.h"

static void	fill(char **tab, t_point size, t_point cur, char target)
{
	if (cur.x < 0 || cur.y < 0 || cur.x >= size.x || cur.y >= size.y
		|| tab[cur.y][cur.x] != target)
		return ;
	tab[cur.y][cur.x] = 'F';
	fill(tab, size, (t_point){cur.x - 1, cur.y}, target);
	fill(tab, size, (t_point){cur.x + 1, cur.y}, target);
	fill(tab, size, (t_point){cur.x, cur.y - 1}, target);
	fill(tab, size, (t_point){cur.x, cur.y + 1}, target);
}

void	flood_fill(char **tab, t_point size, t_point begin)
{
	fill(tab, size, begin, tab[begin.y][begin.x]);
}
