package completion

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeGen(shell string, w io.Writer) error {
	_, err := io.WriteString(w, "# fake "+shell+" completion\n")
	return err
}

func TestInstall_Zsh(t *testing.T) {
	home := t.TempDir()
	res, err := Install("zsh", home, fakeGen)
	if err != nil {
		t.Fatal(err)
	}
	if res.Shell != "zsh" {
		t.Fatalf("shell=%s", res.Shell)
	}
	data, err := os.ReadFile(res.ScriptPath)
	if err != nil || !strings.Contains(string(data), "fake zsh") {
		t.Fatalf("script=%q err=%v", data, err)
	}
	rc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rc), beginMarker) || !strings.Contains(string(rc), "compinit") {
		t.Fatalf("zshrc=%s", rc)
	}

	// Idempotent: second install should not duplicate markers.
	res2, err := Install("zsh", home, fakeGen)
	if err != nil {
		t.Fatal(err)
	}
	rc2, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if strings.Count(string(rc2), beginMarker) != 1 {
		t.Fatalf("marcadores duplicados:\n%s", rc2)
	}
	if res2.RCUpdated {
		// script rewrite may keep rc same — ok either way if single marker
	}
}

func TestInstall_Bash(t *testing.T) {
	home := t.TempDir()
	res, err := Install("bash", home, fakeGen)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.ScriptPath); err != nil {
		t.Fatal(err)
	}
	rc, _ := os.ReadFile(filepath.Join(home, ".bashrc"))
	if !strings.Contains(string(rc), "bash-completion/completions/lightyear") {
		t.Fatalf("%s", rc)
	}
}

func TestInstall_Fish(t *testing.T) {
	home := t.TempDir()
	res, err := Install("fish", home, fakeGen)
	if err != nil {
		t.Fatal(err)
	}
	if res.RCPath != "" {
		t.Fatalf("fish não deve editar rc: %+v", res)
	}
	if _, err := os.Stat(res.ScriptPath); err != nil {
		t.Fatal(err)
	}
}

func TestDetectShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	if DetectShell() != "zsh" {
		t.Fatal(DetectShell())
	}
	t.Setenv("SHELL", "/usr/bin/bash")
	if DetectShell() != "bash" {
		t.Fatal(DetectShell())
	}
}
