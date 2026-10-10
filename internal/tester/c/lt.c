#include "lt.h"

#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <stdarg.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/resource.h>
#include <sys/wait.h>
#include <unistd.h>
#ifdef __linux__
# include <sys/prctl.h>
#endif

/* ---- shared state between a test's child and the runner ---------------- */

struct s_shared
{
	char	msg[2048];
	char	note[512];
};

static const char		*g_group = "";
static char				**g_only;
static int				g_nonly;
static unsigned int		g_timeout = 5;
static struct s_shared	*g_shared;
static int				g_child;
static int				g_hangs;
static int				g_tracking;

void	lt_init(int argc, char **argv)
{
	const char	*t;

	g_only = argv + 1;
	g_nonly = argc - 1;
	t = getenv("LT_TIMEOUT");
	if (t && atoi(t) > 0)
		g_timeout = (unsigned int)atoi(t);
	g_shared = mmap(NULL, sizeof(*g_shared), PROT_READ | PROT_WRITE,
			MAP_SHARED | MAP_ANON, -1, 0);
	if (g_shared == MAP_FAILED)
	{
		perror("lightyear: mmap");
		exit(2);
	}
	signal(SIGPIPE, SIG_IGN);
	/* A crashing case must not dump core: with the arena mapped, writing it
	** out (or handing it to systemd-coredump/apport) takes minutes. */
	setrlimit(RLIMIT_CORE, &(struct rlimit){0, 0});
#ifdef __linux__
	prctl(PR_SET_DUMPABLE, 0, 0, 0, 0);
#endif
}

void	lt_group(const char *group)
{
	g_group = group;
	g_hangs = 0;
}

static int	selected(void)
{
	for (int i = 0; i < g_nonly; i++)
		if (strcmp(g_only[i], g_group) == 0)
			return (1);
	return (g_nonly == 0);
}

/* emit writes one result line; tabs and line breaks in the fields would
** break the protocol, so they become spaces. */
static void	emit(const char *name, const char *status, const char *detail)
{
	char		line[4096];
	const char	*parts[] = {"R", g_group, name, status, detail};
	size_t		n;

	n = 0;
	for (size_t p = 0; p < 5; p++)
	{
		for (const char *s = parts[p]; *s && n < sizeof(line) - 2; s++)
			line[n++] = (*s == '\t' || *s == '\n' || *s == '\r') ? ' ' : *s;
		line[n++] = p < 4 ? '\t' : '\n';
	}
	(void)!write(1, line, n);
}

static const char	*signal_text(int sig)
{
	static char	buf[64];

	if (sig == SIGSEGV)
		return ("segmentation fault");
	if (sig == SIGBUS)
		return ("bus error");
	if (sig == SIGABRT)
		return ("abort");
	if (sig == SIGFPE)
		return ("erro aritmético (divisão por zero?)");
	if (sig == SIGILL)
		return ("instrução ilegal");
	snprintf(buf, sizeof(buf), "sinal %d", sig);
	return (buf);
}

int	lt_begin(const char *name)
{
	pid_t	pid;
	int		st;
	char	detail[3072];
	char	note[600];

	if (!selected())
		return (0);
	if (g_hangs >= 3)
	{
		emit(name, "SKIP", "pulado: a função já travou em 3 testes");
		return (0);
	}
	memset(g_shared, 0, sizeof(*g_shared));
	pid = fork();
	if (pid < 0)
	{
		emit(name, "KO", "fork falhou");
		return (0);
	}
	if (pid == 0)
	{
		int	null = open("/dev/null", O_WRONLY);

		dup2(null, 1);
		dup2(null, 2);
		close(null);
		g_child = 1;
		alarm(g_timeout);
		g_tracking = 1;
		return (1);
	}
	while (waitpid(pid, &st, 0) < 0 && errno == EINTR)
		;
	note[0] = '\0';
	if (g_shared->note[0])
		snprintf(note, sizeof(note), " [%s]", g_shared->note);
	if (WIFEXITED(st) && WEXITSTATUS(st) == 0)
		emit(name, "OK", "");
	else if (WIFEXITED(st) && WEXITSTATUS(st) == 1 && g_shared->msg[0])
	{
		snprintf(detail, sizeof(detail), "%s%s", g_shared->msg, note);
		emit(name, "KO", detail);
	}
	else if (WIFEXITED(st))
	{
		snprintf(detail, sizeof(detail), "o programa saiu com exit(%d) no meio do teste%s",
			WEXITSTATUS(st), note);
		emit(name, "KO", detail);
	}
	else if (WIFSIGNALED(st) && WTERMSIG(st) == SIGALRM)
	{
		snprintf(detail, sizeof(detail), "passou de %us (loop infinito?)%s", g_timeout, note);
		emit(name, "TIMEOUT", detail);
		g_hangs++;
	}
	else
	{
		snprintf(detail, sizeof(detail), "%s%s", signal_text(WTERMSIG(st)), note);
		emit(name, "CRASH", detail);
	}
	return (0);
}

int	lt_end(void)
{
	size_t	live;

	live = lt_live();
	g_tracking = 0;
	if (live)
		lt_fail("vazamento de memória: %zu bloco(s) alocado(s) e não liberado(s) no fim"
			" (o que ficou guardado numa static também conta: tudo tem que ser liberado)", live);
	_exit(0);
}

void	lt_fail(const char *fmt, ...)
{
	va_list	ap;

	g_tracking = 0;
	va_start(ap, fmt);
	if (g_child && g_shared->msg[0] == '\0')
		vsnprintf(g_shared->msg, sizeof(g_shared->msg), fmt, ap);
	else if (!g_child)
		vfprintf(stderr, fmt, ap);
	va_end(ap);
	if (!g_child)
		abort();
	_exit(1);
}

void	lt_note(const char *fmt, ...)
{
	va_list	ap;

	va_start(ap, fmt);
	vsnprintf(g_shared->note, sizeof(g_shared->note), fmt, ap);
	va_end(ap);
}

/* ---- formatting helpers (rotating static buffers) ---------------------- */

static char	*scratch(void)
{
	static char	bufs[16][1024];
	static int	i;

	i = (i + 1) % 16;
	return (bufs[i]);
}

const char	*lt_fmt(const char *fmt, ...)
{
	char	*buf;
	va_list	ap;

	buf = scratch();
	va_start(ap, fmt);
	vsnprintf(buf, 1024, fmt, ap);
	va_end(ap);
	return (buf);
}

/* lt_repr quotes n bytes C style ("a\tb\x80"), shortened past 80 bytes. */
const char	*lt_repr(const void *s, size_t n)
{
	const unsigned char	*p = s;
	char				*buf;
	size_t				o;
	size_t				shown;

	if (s == NULL)
		return ("NULL");
	buf = scratch();
	o = 0;
	buf[o++] = '"';
	shown = n > 80 ? 60 : n;
	for (size_t i = 0; i < shown; i++)
	{
		if (p[i] == '"' || p[i] == '\\')
			o += snprintf(buf + o, 1024 - o, "\\%c", p[i]);
		else if (p[i] == '\n')
			o += snprintf(buf + o, 1024 - o, "\\n");
		else if (p[i] == '\t')
			o += snprintf(buf + o, 1024 - o, "\\t");
		else if (p[i] < 32 || p[i] >= 127)
			o += snprintf(buf + o, 1024 - o, "\\x%02x", p[i]);
		else
			buf[o++] = (char)p[i];
	}
	if (shown < n)
		o += snprintf(buf + o, 1024 - o, "\"… (%zu bytes)", n);
	else
		buf[o++] = '"';
	buf[o] = '\0';
	return (buf);
}

const char	*lt_str(const char *s)
{
	return (s ? lt_repr(s, strlen(s)) : "NULL");
}

/* ---- capture and temporary files --------------------------------------- */

static int	g_capfd = -1;

static int	anon_file(void)
{
	char		path[256];
	const char	*dir;
	int			fd;

	dir = getenv("TMPDIR");
	snprintf(path, sizeof(path), "%s/lightyear-XXXXXX", dir && *dir ? dir : "/tmp");
	fd = mkstemp(path);
	if (fd < 0)
		lt_fail("não deu para criar um arquivo temporário");
	unlink(path);
	return (fd);
}

int	lt_capture(void)
{
	g_capfd = anon_file();
	return (g_capfd);
}

const char	*lt_captured(size_t *len)
{
	static char	buf[1 << 20];
	ssize_t		r;
	size_t		n;

	n = 0;
	lseek(g_capfd, 0, SEEK_SET);
	while (n < sizeof(buf) - 1 && (r = read(g_capfd, buf + n, sizeof(buf) - 1 - n)) > 0)
		n += (size_t)r;
	buf[n] = '\0';
	if (len)
		*len = n;
	return (buf);
}

int	lt_stdout_begin(void)
{
	int	saved;

	saved = dup(1);
	dup2(lt_capture(), 1);
	return (saved);
}

const char	*lt_stdout_end(int saved, size_t *len)
{
	dup2(saved, 1);
	close(saved);
	return (lt_captured(len));
}

int	lt_tmpfile(const char *content, size_t len)
{
	int		fd;
	size_t	done;
	ssize_t	w;

	fd = anon_file();
	done = 0;
	while (done < len && (w = write(fd, content + done, len - done)) > 0)
		done += (size_t)w;
	lseek(fd, 0, SEEK_SET);
	return (fd);
}

/* ---- allocator --------------------------------------------------------- */

/*
** Bump allocator over one big lazily-committed mapping: blocks are never
** reused, so a double free or a free of a pointer that isn't a block start
** is always detected. Large freed blocks give their pages back (madvise).
**
**   [hdr 16][payload size, filled with 0xbe][canary 16 x 0xcd][pad to 16]
*/

#define MAGIC 0x4c594152u
#define CANARY 16
#define LIVE 1
#define FREED 2

struct s_hdr
{
	size_t			size;
	unsigned int	magic;
	unsigned char	state;
	unsigned char	tracked;
	unsigned char	pad[2];
};

static char		*g_arena;
static char		*g_top;
static char		*g_end;
static size_t	g_live;
static long		g_allocs;
static long		g_failat = -1;
static long		g_counter;
static int		g_failed;

static int	arena_init(void)
{
	static const size_t	sizes[] = {(size_t)1 << 36, (size_t)1 << 33, (size_t)1 << 30};

	for (size_t i = 0; i < sizeof(sizes) / sizeof(*sizes); i++)
	{
		void	*p = mmap(NULL, sizes[i], PROT_READ | PROT_WRITE,
				MAP_PRIVATE | MAP_ANON | MAP_NORESERVE, -1, 0);

		if (p != MAP_FAILED)
		{
#ifdef MADV_DONTDUMP
			madvise(p, sizes[i], MADV_DONTDUMP);
#endif
			g_arena = p;
			g_top = p;
			g_end = g_arena + sizes[i];
			return (1);
		}
	}
	return (0);
}

static void	*alloc(size_t size, size_t align)
{
	uintptr_t		payload;
	struct s_hdr	*h;

	if (g_tracking && g_failat >= 0 && g_counter++ == g_failat)
	{
		g_failed = 1;
		errno = ENOMEM;
		return (NULL);
	}
	if (!g_arena && !arena_init())
		return (NULL);
	if (align < 16)
		align = 16;
	payload = ((uintptr_t)g_top + sizeof(struct s_hdr) + align - 1) & ~(uintptr_t)(align - 1);
	if (size > (size_t)(g_end - (char *)payload) || (size_t)(g_end - (char *)payload) - size < CANARY + 16)
	{
		errno = ENOMEM;
		return (NULL);
	}
	h = (struct s_hdr *)(payload - sizeof(struct s_hdr));
	h->size = size;
	h->magic = MAGIC;
	h->state = LIVE;
	h->tracked = (unsigned char)g_tracking;
	/* Garbage shows a missing '\0'; past 64 KB it only costs time (a
	** BUFFER_SIZE of 10000000 allocates 10 MB per call). */
	memset((char *)payload, 0xbe, size < (1 << 16) ? size : (1 << 16));
	memset((char *)payload + size, 0xcd, CANARY);
	g_top = (char *)((payload + size + CANARY + 15) & ~(uintptr_t)15);
	if (g_tracking)
	{
		g_live++;
		g_allocs++;
	}
	return ((void *)payload);
}

static struct s_hdr	*header(void *p)
{
	struct s_hdr	*h;

	if (!g_arena || (char *)p < g_arena + sizeof(struct s_hdr) || (char *)p >= g_top
		|| ((uintptr_t)p & 15) != 0)
		return (NULL);
	h = (struct s_hdr *)((char *)p - sizeof(struct s_hdr));
	if (h->magic != MAGIC)
		return (NULL);
	return (h);
}

void	*malloc(size_t size)
{
	return (alloc(size, 16));
}

void	free(void *p)
{
	struct s_hdr	*h;
	unsigned char	*canary;

	if (p == NULL)
		return ;
	h = header(p);
	if (h == NULL)
	{
		if (g_tracking)
			lt_fail("free(%p): ponteiro que não é o início de um bloco do malloc", p);
		return ;
	}
	if (h->state == FREED)
	{
		if (g_tracking)
			lt_fail("double free: o mesmo bloco de %zu bytes foi liberado duas vezes", h->size);
		return ;
	}
	canary = (unsigned char *)p + h->size;
	for (size_t i = 0; i < CANARY; i++)
		if (canary[i] != 0xcd && g_tracking)
			lt_fail("escreveu além do fim de um bloco de %zu bytes (heap overflow: faltou +1 no malloc?)",
				h->size);
	h->state = FREED;
	if (h->tracked && g_live)
		g_live--;
	if (h->size >= 1 << 16)
	{
		uintptr_t	from = ((uintptr_t)p + 4095) & ~(uintptr_t)4095;
		uintptr_t	to = ((uintptr_t)p + h->size) & ~(uintptr_t)4095;

		if (to > from)
			madvise((void *)from, to - from, MADV_DONTNEED);
	}
	else
		memset(p, 0xdf, h->size);
}

void	*calloc(size_t count, size_t size)
{
	void	*p;

	if (size && count > SIZE_MAX / size)
	{
		errno = ENOMEM;
		return (NULL);
	}
	p = alloc(count * size, 16);
	if (p)
		memset(p, 0, count * size);
	return (p);
}

void	*realloc(void *old, size_t size)
{
	struct s_hdr	*h;
	void			*p;

	if (old == NULL)
		return (malloc(size));
	h = header(old);
	p = malloc(size);
	if (p && h)
		memcpy(p, old, h->size < size ? h->size : size);
	if (p)
		free(old);
	return (p);
}

#ifdef __linux__

int	posix_memalign(void **out, size_t align, size_t size)
{
	void	*p;

	p = alloc(size, align);
	if (p == NULL)
		return (ENOMEM);
	*out = p;
	return (0);
}

void	*aligned_alloc(size_t align, size_t size)
{
	return (alloc(size, align));
}

void	*memalign(size_t align, size_t size)
{
	return (alloc(size, align));
}

void	*valloc(size_t size)
{
	return (alloc(size, 4096));
}

void	*pvalloc(size_t size)
{
	return (alloc((size + 4095) & ~(size_t)4095, 4096));
}

size_t	malloc_usable_size(void *p)
{
	struct s_hdr	*h;

	h = p ? header(p) : NULL;
	return (h ? h->size : 0);
}

#endif

size_t	lt_live(void)
{
	return (g_live);
}

long	lt_allocs(void)
{
	return (g_allocs);
}

size_t	lt_block_size(const void *p)
{
	struct s_hdr	*h;

	h = p ? header((void *)p) : NULL;
	return (h && h->state == LIVE ? h->size : 0);
}

void	lt_fail_alloc(long n)
{
	g_failat = n;
	g_counter = 0;
	g_failed = 0;
}

int	lt_alloc_failed(void)
{
	return (g_failed);
}

void	lt_alloc_fails(void *(*call)(void), void (*release)(void *))
{
	for (long n = 0; n < 100000; n++)
	{
		size_t	before = lt_live();
		void	*r;
		int		failed;

		lt_note("com o %ldº malloc devolvendo NULL", n + 1);
		lt_fail_alloc(n);
		r = call();
		failed = lt_alloc_failed();
		lt_fail_alloc(-1);
		if (!failed)
		{
			g_shared->note[0] = '\0';
			if (release)
				release(r);
			return ;
		}
		CHECK(r == NULL, "o %ldº malloc devolveu NULL, mas a função não devolveu NULL", n + 1);
		CHECK(lt_live() == before,
			"o %ldº malloc devolveu NULL e a função não liberou o que já tinha alocado (%zu bloco(s))",
			n + 1, lt_live() - before);
	}
}
