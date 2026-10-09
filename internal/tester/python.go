package tester

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// The Python modules of the Common Core: ex0/, ex1/… at the root, checked
// by py/lt.py and the module's suite (py/moduleNN.py), which run every
// case in its own python3 process.
func pythonModule(n string) Project {
	return Project{
		Name:    "python-module-" + n,
		Subject: "python-module-" + n,
		Markers: []string{"ex0"},
		run: func(ctx context.Context, r *runner) error {
			return r.runPython(ctx, "module"+n+".py")
		},
	}
}

func (r *runner) runPython(ctx context.Context, suite string) error {
	python, err := lookPath("python3")
	if err != nil {
		return fmt.Errorf("python3 não encontrado: %w", err)
	}
	build := filepath.Join(r.work, "build")
	if err := writeEmbedded(build); err != nil {
		return err
	}
	r.step("rodando os testes")
	ctx, cancel := context.WithTimeout(ctx, suiteTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, filepath.Join(build, "py", "lt.py"), filepath.Join(build, "py", suite), r.proj)
	cmd.Dir = r.proj
	return r.collect(ctx, cmd)
}
