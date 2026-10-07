package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRootHelp(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute --help: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"lightyear", "version", "update", "subject", "config"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q:\n%s", want, out)
		}
	}
}

func TestVersionCmd(t *testing.T) {
	Version = "test-version"
	Commit = "abc123"
	BuildDate = "2026-01-01"
	t.Cleanup(func() {
		Version = "dev"
		Commit = "none"
		BuildDate = "unknown"
	})

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute version: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "test-version") {
		t.Fatalf("version output missing version:\n%s", out)
	}
	if !strings.Contains(out, "abc123") {
		t.Fatalf("version output missing commit:\n%s", out)
	}
}

func TestSubjectImportHelp(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"subject", "import", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute subject import --help: %v", err)
	}
	if !strings.Contains(buf.String(), "import") {
		t.Fatalf("help missing import:\n%s", buf.String())
	}
}

func TestSubjectSetIDHelp(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"subject", "set-id", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute subject set-id --help: %v", err)
	}
	if !strings.Contains(buf.String(), "set-id") {
		t.Fatalf("help missing set-id:\n%s", buf.String())
	}
}

func TestConfigPathCmd(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"config", "path"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute config path: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if !strings.HasSuffix(out, "config.yaml") {
		t.Fatalf("path = %q, want suffix config.yaml", out)
	}
}

func TestVersionShowsChangelog(t *testing.T) {
	t.Cleanup(func() { Version = "dev" })
	run := func(args ...string) string {
		t.Helper()
		root := NewRootCmd()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetErr(buf)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return ansi.Strip(buf.String())
	}

	Version = "1.2.1"
	for _, args := range [][]string{{"version"}, {"--version"}, {"-v"}} {
		out := run(args...)
		if !strings.Contains(out, "Novidades da 1.2.1 · 2026-10-06") || !strings.Contains(out, "• lightyear update --canary e --stable") {
			t.Fatalf("%v sem as novidades da 1.2.1:\n%s", args, out)
		}
		if strings.Contains(out, "Novidades da 1.2.0") {
			t.Fatalf("%v deveria mostrar só a versão instalada:\n%s", args, out)
		}
	}

	out := run("version", "--changelog")
	for _, want := range []string{"1.3.0-canary.10", "1.2.1", "0.1.0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("--changelog sem %s:\n%s", want, out)
		}
	}

	Version = "dev"
	if out := run("--version"); !strings.Contains(out, "mostrando a mais recente") || !strings.Contains(out, "Novidades da") {
		t.Fatalf("build de dev deveria mostrar a versão mais recente:\n%s", out)
	}
}
