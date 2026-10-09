package tester

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// libftFunctions are the subject's functions (version 19), in its order.
// The list part (Part 3) used to be the bonus; it is mandatory now.
var libftFunctions = []string{
	"ft_isalpha", "ft_isdigit", "ft_isalnum", "ft_isascii", "ft_isprint", "ft_strlen",
	"ft_memset", "ft_bzero", "ft_memcpy", "ft_memmove", "ft_strlcpy", "ft_strlcat",
	"ft_toupper", "ft_tolower", "ft_strchr", "ft_strrchr", "ft_strncmp", "ft_memchr",
	"ft_memcmp", "ft_strnstr", "ft_atoi", "ft_calloc", "ft_strdup",
	"ft_substr", "ft_strjoin", "ft_strtrim", "ft_split", "ft_itoa", "ft_strmapi",
	"ft_striteri", "ft_putchar_fd", "ft_putstr_fd", "ft_putendl_fd", "ft_putnbr_fd",
	"ft_lstnew", "ft_lstadd_front", "ft_lstsize", "ft_lstlast", "ft_lstadd_back",
	"ft_lstdelone", "ft_lstclear", "ft_lstiter", "ft_lstmap",
}

var libft = Project{
	Name:    "libft",
	Subject: "42cursus-libft",
	Markers: []string{"Makefile", "libft.h"},
	run:     runLibft,
}

func runLibft(ctx context.Context, r *runner) error {
	r.readme("README.md")
	r.norminette(ctx, "norminette")
	r.libftFiles("arquivos")

	lib := filepath.Join(r.proj, "libft.a")
	built, err := r.makefile(ctx, "Makefile", "libft.a")
	if err != nil || !built {
		return err
	}

	defined, undefined, err := symbols(ctx, lib)
	if err != nil {
		return err
	}
	// Libfts made for the old subject build the lists with `make bonus`.
	if _, ok := defined["ft_lstnew"]; !ok {
		if has, err := r.hasRule(ctx, "bonus"); err != nil {
			return err
		} else if has {
			r.step("make bonus")
			out, ok, err := r.make(ctx, "bonus")
			if err != nil {
				return err
			}
			r.check("Makefile", "make bonus compila", ok, tail(out, 10))
			if defined, undefined, err = symbols(ctx, lib); err != nil {
				return err
			}
		}
	}

	r.globals("variáveis globais", defined)
	r.forbidden("funções permitidas", undefined, "malloc", "free", "write")

	var suites []suite
	for _, fn := range libftFunctions {
		if _, ok := defined[fn]; !ok {
			detail := "a função não está na libft.a"
			if strings.HasPrefix(fn, "ft_lst") {
				detail += " (as listas são obrigatórias desde a versão 19 do subject)"
			}
			r.add(fn, "existe na libft.a", KO, detail)
			continue
		}
		if len(r.opts.Only) == 0 || slices.Contains(r.opts.Only, fn) {
			suites = append(suites, suite{dir: "libft", name: fn, group: fn})
		}
	}
	header, _ := findFile(r.proj, "libft.h")
	return r.runSuites(ctx, suites, []string{header}, []string{lib})
}

// readmeTitle is the first line the subjects ask for.
var readmeTitle = regexp.MustCompile(`^([*_])This project has been created as part of the 42 curriculum by [^*_]+[*_]\s*$`)

// readme checks the README.md every Common Core subject asks for: the
// italic first line and the Description, Instructions and Resources sections.
func (r *runner) readme(group string) {
	data, err := os.ReadFile(filepath.Join(r.proj, "README.md"))
	if err != nil {
		r.add(group, "README.md na raiz", KO, "não tem README.md na raiz do repositório")
		return
	}
	r.add(group, "README.md na raiz", OK, "")
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	first, _, _ := strings.Cut(text, "\n")
	r.check(group, "primeira linha em itálico com os logins", readmeTitle.MatchString(strings.TrimSpace(first)),
		"a 1ª linha deve ser, em itálico: *This project has been created as part of the 42 curriculum by <login>.*\nrecebido: "+first)
	for _, section := range []string{"Description", "Instructions", "Resources"} {
		re := regexp.MustCompile(`(?im)^#+\s*` + section)
		r.check(group, "seção "+section, re.MatchString(text), "falta um título \"## "+section+"\"")
	}
}

// libftFiles checks the layout: everything at the root, only ft_*.c sources.
func (r *runner) libftFiles(group string) {
	var nested, odd []string
	_ = filepath.WalkDir(r.proj, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(r.proj, path)
		ext := filepath.Ext(rel)
		if ext != ".c" && ext != ".h" {
			return nil
		}
		switch {
		case strings.Contains(rel, string(filepath.Separator)):
			nested = append(nested, rel)
		case ext == ".c" && !strings.HasPrefix(rel, "ft_"):
			odd = append(odd, rel)
		case ext == ".h" && rel != "libft.h":
			odd = append(odd, rel)
		}
		return nil
	})
	r.check(group, "todos os arquivos na raiz", len(nested) == 0, "fora da raiz: "+strings.Join(nested, ", "))
	r.check(group, "só ft_*.c e libft.h", len(odd) == 0, "o subject pede Makefile, libft.h e ft_*.c; sobram: "+strings.Join(odd, ", "))
}

// globals fails when the library defines global variables (data symbols).
func (r *runner) globals(group string, defined map[string]string) {
	var vars []string
	for name, kind := range defined {
		if kind != "T" && kind != "W" && !compilerSymbols[name] {
			vars = append(vars, name)
		}
	}
	slices.Sort(vars)
	r.check(group, "nenhuma variável global", len(vars) == 0, "variáveis globais: "+strings.Join(vars, ", "))
}

// makefile checks the Makefile of a library named name: the rules, the
// flags, ar (not libtool), no relink, clean, fclean and re. It leaves the
// library built and reports whether it is.
func (r *runner) makefile(ctx context.Context, group, name string) (bool, error) {
	// Nothing is built yet, so every rule has something to do: "Nothing to
	// be done" means a rule that is only listed in .PHONY.
	for _, rule := range []string{name, "all", "clean", "fclean", "re"} {
		out, _, err := r.make(ctx, "-n", rule)
		if err != nil {
			return false, err
		}
		ok := !strings.Contains(out, "No rule to make target") && !strings.Contains(out, "Nothing to be done")
		r.check(group, "regra "+rule, ok, "a regra "+rule+" não existe ou não faz nada")
	}

	dry, _, err := r.make(ctx, "-n")
	if err != nil {
		return false, err
	}
	var compiles, missing []string
	for line := range strings.SplitSeq(dry, "\n") {
		f := strings.Fields(line)
		if !slices.Contains(f, "-c") {
			continue
		}
		compiles = append(compiles, line)
		for _, flag := range []string{"-Wall", "-Wextra", "-Werror"} {
			if !slices.Contains(f, flag) && !slices.Contains(missing, flag) {
				missing = append(missing, flag)
			}
		}
	}
	if len(compiles) == 0 {
		r.add(group, "compila com -Wall -Wextra -Werror", Skip, "não deu para ver os comandos de compilação (make -n)")
	} else {
		r.check(group, "compila com -Wall -Wextra -Werror", len(missing) == 0, "falta "+strings.Join(missing, " ")+" em: "+compiles[0])
	}
	r.check(group, "usa ar (libtool é proibido)", !strings.Contains(dry, "libtool"), "o Makefile usa libtool")

	r.step("make")
	out, ok, err := r.make(ctx)
	if err != nil {
		return false, err
	}
	lib := filepath.Join(r.proj, name)
	_, statErr := os.Stat(lib)
	r.check(group, "make compila "+name, ok && statErr == nil, tail(out, 15))
	if !ok || statErr != nil {
		r.add("testes", "rodar os testes", Skip, "o projeto não compilou")
		return false, nil
	}

	before := modTime(lib)
	out, _, err = r.make(ctx)
	if err != nil {
		return false, err
	}
	r.check(group, "make de novo não refaz nada (relink)", modTime(lib) == before, name+" foi refeita:\n"+tail(out, 5))

	if _, _, err := r.make(ctx, "clean"); err != nil {
		return false, err
	}
	objs := findExt(r.proj, ".o")
	r.check(group, "make clean apaga os .o", len(objs) == 0, "sobraram: "+strings.Join(objs, ", "))
	_, statErr = os.Stat(lib)
	r.check(group, "make clean mantém "+name, statErr == nil, name+" sumiu")

	if _, _, err := r.make(ctx, "fclean"); err != nil {
		return false, err
	}
	_, statErr = os.Stat(lib)
	r.check(group, "make fclean apaga "+name, statErr != nil, name+" continua lá")

	r.step("make re")
	out, ok, err = r.make(ctx, "re")
	if err != nil {
		return false, err
	}
	_, statErr = os.Stat(lib)
	r.check(group, "make re recompila", ok && statErr == nil, tail(out, 10))
	if statErr != nil {
		if out, ok, err = r.make(ctx); err != nil || !ok {
			r.add("testes", "rodar os testes", Skip, "o projeto não compilou: "+tail(out, 5))
			return false, err
		}
	}
	return true, nil
}

func modTime(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano()
}

// findExt lists the files under root with extension ext, relative to root.
func findExt(root, ext string) []string {
	var found []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ext {
			rel, _ := filepath.Rel(root, path)
			found = append(found, rel)
		}
		return nil
	})
	return found
}
