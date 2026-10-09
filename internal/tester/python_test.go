package tester

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Each Python module's reference solution passes its whole suite (flake8
// and mypy are skipped where they aren't installed).
func TestPythonModulesReferencePasses(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 não encontrado")
	}
	for _, p := range projects {
		if len(p.Markers) == 0 || p.Markers[0] != "ex0" {
			continue
		}
		t.Run(p.Name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join("testdata", p.Name)
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("falta a solução de referência em %s", dir)
			}
			rep, err := p.Run(context.Background(), dir, Options{})
			if err != nil {
				t.Fatal(err)
			}
			passed, _, _ := rep.Count()
			if passed < 20 {
				t.Errorf("só %d casos passaram", passed)
			}
			for _, c := range rep.Cases {
				if c.Status.Failed() {
					t.Errorf("%s / %s: %s %s", c.Group, c.Name, c.Status, c.Detail)
				}
			}
		})
	}
}

// A folder that isn't a module fails CheckRoot; an empty ex0 runs and fails.
func TestPythonModuleMissingFiles(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 não encontrado")
	}
	p, err := Lookup("python-module-00")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := p.CheckRoot(dir); err == nil {
		t.Error("CheckRoot aceitou uma pasta sem ex0")
	}
	if err := os.Mkdir(filepath.Join(dir, "ex0"), 0o755); err != nil {
		t.Fatal(err)
	}
	rep, err := p.Run(context.Background(), dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, failed, _ := rep.Count(); failed == 0 {
		t.Errorf("um módulo vazio deveria falhar: %+v", rep.Cases)
	}
}
