#include <stdlib.h>
#include "libft.h"

t_list	*ft_lstnew(void *content)
{
	t_list	*n = malloc(sizeof(t_list));

	if (!n)
		return (NULL);
	n->content = content;
	n->next = NULL;
	return (n);
}

void	ft_lstadd_front(t_list **lst, t_list *new)
{
	new->next = *lst;
	*lst = new;
}

int	ft_lstsize(t_list *lst)
{
	int	n = 0;

	for (; lst; lst = lst->next)
		n++;
	return (n);
}

t_list	*ft_lstlast(t_list *lst)
{
	while (lst && lst->next)
		lst = lst->next;
	return (lst);
}

void	ft_lstadd_back(t_list **lst, t_list *new)
{
	if (!*lst)
		*lst = new;
	else
		ft_lstlast(*lst)->next = new;
}

void	ft_lstdelone(t_list *lst, void (*del)(void *))
{
	del(lst->content);
	free(lst);
}

void	ft_lstclear(t_list **lst, void (*del)(void *))
{
	while (*lst)
	{
		t_list	*next = (*lst)->next;

		ft_lstdelone(*lst, del);
		*lst = next;
	}
}

void	ft_lstiter(t_list *lst, void (*f)(void *))
{
	for (; lst; lst = lst->next)
		f(lst->content);
}

t_list	*ft_lstmap(t_list *lst, void *(*f)(void *), void (*del)(void *))
{
	t_list	*head = NULL;

	for (; lst; lst = lst->next)
	{
		void	*c = f(lst->content);
		t_list	*n = ft_lstnew(c);

		if (!n)
		{
			del(c);
			ft_lstclear(&head, del);
			return (NULL);
		}
		ft_lstadd_back(&head, n);
	}
	return (head);
}
