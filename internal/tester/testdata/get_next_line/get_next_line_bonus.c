#include <stdlib.h>
#include <unistd.h>
#include "get_next_line_bonus.h"

static char	*fill(int fd, char *stash)
{
	char	*buf;
	ssize_t	n;

	buf = malloc((size_t)BUFFER_SIZE + 1);
	if (!buf)
		return (free(stash), NULL);
	n = 1;
	while (!gnl_chr(stash, '\n') && n > 0)
	{
		n = read(fd, buf, BUFFER_SIZE);
		if (n < 0)
			return (free(buf), free(stash), NULL);
		if (n > 0)
		{
			stash = gnl_join(stash, buf, (size_t)n);
			if (!stash)
				return (free(buf), NULL);
		}
	}
	free(buf);
	return (stash);
}

static char	*take(char **stash)
{
	char	*nl;
	char	*line;
	char	*rest;
	size_t	len;

	nl = gnl_chr(*stash, '\n');
	len = gnl_len(*stash);
	if (nl)
		len = (size_t)(nl - *stash) + 1;
	line = gnl_sub(*stash, 0, len);
	rest = NULL;
	if (line && (*stash)[len])
		rest = gnl_sub(*stash, len, gnl_len(*stash) - len);
	if (!line || ((*stash)[len] && !rest))
	{
		free(line);
		line = NULL;
		free(rest);
		rest = NULL;
	}
	free(*stash);
	*stash = rest;
	return (line);
}

char	*get_next_line(int fd)
{
	static char	*stash[4096];

	if (fd < 0 || fd >= 4096 || BUFFER_SIZE <= 0)
		return (NULL);
	stash[fd] = fill(fd, stash[fd]);
	if (!stash[fd] || !*stash[fd])
	{
		free(stash[fd]);
		stash[fd] = NULL;
		return (NULL);
	}
	return (take(&stash[fd]));
}
