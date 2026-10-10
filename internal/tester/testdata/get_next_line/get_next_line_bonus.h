#ifndef GET_NEXT_LINE_BONUS_H
# define GET_NEXT_LINE_BONUS_H

# include <stddef.h>

# ifndef BUFFER_SIZE
#  define BUFFER_SIZE 42
# endif

char	*get_next_line(int fd);
size_t	gnl_len(const char *s);
char	*gnl_chr(const char *s, int c);
char	*gnl_join(char *stash, const char *buf, size_t n);
char	*gnl_sub(const char *s, size_t start, size_t len);

#endif
