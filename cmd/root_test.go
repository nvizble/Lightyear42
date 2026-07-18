package cmd

import (
	"bytes"
	"strings"
	"testing"
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
