package tester

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// born2beroot checks the machine it runs on (the evaluated VM), not a
// folder: sh/born2beroot.sh, which also works on its own (bash
// born2beroot.sh) for a VM without lightyear.
var born2beroot = Project{
	Name:    "born2beroot",
	Subject: "42next-born2beroot",
	Machine: true,
	run:     runBorn2beroot,
}

func runBorn2beroot(ctx context.Context, r *runner) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("o born2beroot se testa dentro da VM avaliada (Linux): instale o lightyear nela e rode sudo lightyear test born2beroot")
	}
	bash, err := lookPath("bash")
	if err != nil {
		return fmt.Errorf("bash não encontrado: %w", err)
	}
	build := filepath.Join(r.work, "build")
	if err := writeEmbedded(build); err != nil {
		return err
	}
	r.step("checando a máquina")
	ctx, cancel := context.WithTimeout(ctx, suiteTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bash, filepath.Join(build, "sh", "born2beroot.sh"))
	cmd.Env = append(os.Environ(), "LT_RAW=1")
	if err := r.collect(ctx, cmd); err != nil {
		return err
	}
	// The script exits 1 when something failed; that is in the cases.
	kept := r.cases[:0]
	for _, c := range r.cases {
		if c.Group != "testes" || c.Name != "os testes terminaram bem" {
			kept = append(kept, c)
		}
	}
	r.cases = kept
	return nil
}
