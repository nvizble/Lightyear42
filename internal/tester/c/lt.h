/*
** lightyear tester runtime. Every TEST runs in a forked child, so a crash,
** an infinite loop (alarm) or an exit() only fails that test. The runtime
** also replaces malloc/free: blocks are filled with garbage (missing '\0'
** shows up), guarded by canaries (writes past the end fail the test),
** counted (whatever a test leaves allocated is a leak) and can be made to
** fail on purpose (lt_fail_alloc), to check that every malloc is protected.
**
** Results go to stdout, one line per test:
**   R <tab> group <tab> name <tab> OK|KO|CRASH|TIMEOUT <tab> detail
*/
#ifndef LT_H
# define LT_H

# include <stddef.h>

void		lt_init(int argc, char **argv);
void		lt_group(const char *group);
int			lt_begin(const char *name);
int			lt_end(void);
void		lt_fail(const char *fmt, ...)
			__attribute__((format(printf, 1, 2), noreturn));
void		lt_note(const char *fmt, ...) __attribute__((format(printf, 1, 2)));
const char	*lt_fmt(const char *fmt, ...) __attribute__((format(printf, 1, 2)));
const char	*lt_repr(const void *s, size_t n);
const char	*lt_str(const char *s);

/* Allocations made inside the current test. */
size_t		lt_live(void);
long		lt_allocs(void);
size_t		lt_block_size(const void *p);
void		lt_fail_alloc(long n);
int			lt_alloc_failed(void);

/* Makes each allocation of call() fail in turn (the 1st, then the 2nd...):
** call() must return NULL and free what it had allocated so far. Stops at
** the first run where no allocation failed and hands its result to release. */
void		lt_alloc_fails(void *(*call)(void), void (*release)(void *));

/* Output written to a file descriptor: lt_capture() returns a fd to hand
** to the function, lt_captured() what was written to it (static buffer). */
int			lt_capture(void);
const char	*lt_captured(size_t *len);

/* What a call writes to stdout: lt_stdout_begin() points fd 1 at a
** temporary file, lt_stdout_end() puts it back and returns what was written
** (static buffer). */
int			lt_stdout_begin(void);
const char	*lt_stdout_end(int saved, size_t *len);

/* Temporary file with the given content, opened for reading. */
int			lt_tmpfile(const char *content, size_t len);

/* Runs body in a child; the test fails with the first CHECK that fails. */
# define TEST(name) for (int lt_run_ = lt_begin(name); lt_run_; lt_run_ = lt_end())
# define CHECK(cond, ...) do { if (!(cond)) lt_fail(__VA_ARGS__); } while (0)

/* Expected-vs-got for strings (NULL-safe). */
# define CHECK_STR(got, want) do { \
	const char *g_ = (got), *w_ = (want); \
	if (g_ == NULL || w_ == NULL ? g_ != w_ : strcmp(g_, w_) != 0) \
		lt_fail("esperado %s, recebido %s", lt_str(w_), lt_str(g_)); \
} while (0)

#endif
