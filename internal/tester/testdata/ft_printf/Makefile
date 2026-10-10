NAME = libftprintf.a
CC = cc
CFLAGS = -Wall -Wextra -Werror
OBJS = ft_printf.o ft_printf_utils.o
BONUS_OBJS = ft_printf_bonus.o ft_printf_utils.o

all: $(NAME)

$(NAME): $(OBJS)
	ar rcs $(NAME) $(OBJS)

bonus: .bonus

.bonus: $(BONUS_OBJS)
	rm -f $(NAME)
	ar rcs $(NAME) $(BONUS_OBJS)
	touch .bonus

%.o: %.c ft_printf.h
	$(CC) $(CFLAGS) -c $< -o $@

clean:
	rm -f $(OBJS) $(BONUS_OBJS) .bonus

fclean: clean
	rm -f $(NAME)

re: fclean all

.PHONY: all bonus clean fclean re
