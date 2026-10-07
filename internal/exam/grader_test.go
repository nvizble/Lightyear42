package exam

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func requireCC(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("cc não disponível")
	}
}

func TestLoadEmbeddedCatalog(t *testing.T) {
	exercises, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(exercises) == 0 {
		t.Fatal("catálogo vazio")
	}
	for _, ex := range exercises {
		if ex.Rank == "" || ex.Level < 1 {
			t.Errorf("%s: exercício incompleto: %+v", ex.Name, ex)
		}
		for _, name := range ex.Files {
			if _, ok := ex.ref[name]; !ok {
				t.Errorf("%s: ref/ sem %s", ex.Name, name)
			}
		}
		for _, lang := range []string{"en", "pt"} {
			if ex.Subjects[lang] == "" {
				t.Errorf("%s: falta subject.%s.txt", ex.Name, lang)
			}
		}
	}
}

// Every reference solution, turned in as the student's answer, must pass:
// this catches broken references and tests that disagree with them.
func TestReferenceSolutionsPass(t *testing.T) {
	requireCC(t)
	exercises, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, ex := range exercises {
		t.Run(ex.Name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, src := range ex.ref {
				writeFile(t, dir, name, src)
			}
			res, err := NewGrader().Grade(context.Background(), ex, dir)
			if err != nil {
				t.Fatal(err)
			}
			if !res.Passed {
				t.Fatalf("referência reprovada:\n%s", res.Trace)
			}
		})
	}
}

func TestGradeFailures(t *testing.T) {
	requireCC(t)
	ex := findExercise(t, "first_word")

	tests := []struct {
		name  string
		src   string // empty: file not turned in
		trace string
	}{
		{"not turned in", "", "arquivo não entregue"},
		{"compile error", "int main(void) { return x; }", "erro de compilação"},
		{"wrong output", "#include <unistd.h>\nint main(void) { write(1, \"x\\n\", 2); return (0); }", "esperado:"},
		{"timeout", "int main(void) { while (1) ; }", "timeout"},
		{"crash", "int main(void) { volatile int *p = 0; return *p; }", "crash"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.src != "" {
				writeFile(t, dir, "first_word.c", tt.src)
			}
			g := NewGrader()
			g.Timeout = time.Second
			res, err := g.Grade(context.Background(), ex, dir)
			if err != nil {
				t.Fatal(err)
			}
			if res.Passed {
				t.Fatal("submissão errada foi aprovada")
			}
			if !strings.Contains(res.Trace, tt.trace) {
				t.Fatalf("trace sem %q:\n%s", tt.trace, res.Trace)
			}
		})
	}
}

func findExercise(t *testing.T, name string) Exercise {
	t.Helper()
	exercises, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, ex := range exercises {
		if ex.Name == name {
			return ex
		}
	}
	t.Fatalf("exercício %s não encontrado", name)
	return Exercise{}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
