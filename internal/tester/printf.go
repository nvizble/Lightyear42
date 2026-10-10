package tester

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

var ftPrintf = Project{
	Name:    "ft_printf",
	Subject: "42cursus-ft_printf",
	Markers: []string{"Makefile", "ft_printf.h"},
	run:     runPrintf,
}

func runPrintf(ctx context.Context, r *runner) error {
	if _, err := lookPath(r.opts.CC); err != nil {
		return fmt.Errorf("compilador %q não encontrado: %w", r.opts.CC, err)
	}
	r.readme("README.md")
	r.norminette(ctx, "norminette")

	lib := filepath.Join(r.proj, "libftprintf.a")
	built, err := r.makefile(ctx, "Makefile", "libftprintf.a")
	if err != nil || !built {
		return err
	}
	defined, undefined, err := symbols(ctx, lib)
	if err != nil {
		return err
	}
	r.globals("variáveis globais", defined)
	// va_start and friends are compiler builtins: they never show up in nm.
	r.forbidden("funções permitidas", undefined, "malloc", "free", "write")
	if _, ok := defined["ft_printf"]; !ok {
		r.add("ft_printf", "existe na libftprintf.a", KO, "a função ft_printf não está na libftprintf.a")
		return nil
	}
	if err := r.runSuites(ctx, []suite{{dir: "ft_printf", name: "mandatory", group: "ft_printf"}}, nil, []string{lib}); err != nil {
		return err
	}

	has, err := r.hasRule(ctx, "bonus")
	if err != nil {
		return err
	}
	if !has {
		r.add("bônus", "regra bonus no Makefile", Skip, "o bônus não foi feito")
		return nil
	}
	r.step("make bonus")
	out, ok, err := r.make(ctx, "bonus")
	if err != nil {
		return err
	}
	r.check("Makefile", "make bonus compila", ok, tail(out, 10))
	if !ok {
		return nil
	}
	before := len(r.cases)
	err = r.runSuites(ctx, []suite{
		{dir: "ft_printf", name: "bonus_flags", group: "bônus: - 0 . e largura"},
		{dir: "ft_printf", name: "bonus_signs", group: "bônus: # espaço +"},
	}, nil, []string{lib})
	r.skipUndoneBonus(before)
	return err
}

// skipUndoneBonus turns a bonus group where every case failed into a
// single skipped case: the subject doesn't ask for every bonus, so a group
// that was never started isn't a failure.
func (r *runner) skipUndoneBonus(from int) {
	cases := r.cases[from:]
	failed := map[string]int{}
	total := map[string]int{}
	for _, c := range cases {
		if strings.HasPrefix(c.Group, "bônus") {
			total[c.Group]++
			if c.Status.Failed() {
				failed[c.Group]++
			}
		}
	}
	kept := r.cases[:from]
	skipped := map[string]bool{}
	for _, c := range cases {
		if total[c.Group] > 0 && failed[c.Group] == total[c.Group] {
			if !skipped[c.Group] {
				skipped[c.Group] = true
				kept = append(kept, Case{Group: c.Group, Name: "flags deste bônus", Status: Skip,
					Detail: "nenhum caso passou: parece que esta parte do bônus não foi feita (o subject não pede todas)"})
			}
			continue
		}
		kept = append(kept, c)
	}
	r.cases = kept
}
