// Package completion installs shell autocompletion scripts for lightyear.
package completion

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	binName = "lightyear"

	beginMarker = "# >>> lightyear completion >>>"
	endMarker   = "# <<< lightyear completion <<<"
)

// Generator writes a shell completion script for the given shell name
// (zsh, bash, fish) into w.
type Generator func(shell string, w io.Writer) error

// Result describes what Install configured.
type Result struct {
	Shell      string
	ScriptPath string
	RCPath     string
	RCUpdated  bool
	ReloadHint string
	AlreadyOK  bool
}

// DetectShell returns zsh, bash or fish from $SHELL (or runtime GOOS hints).
func DetectShell() string {
	shell := filepath.Base(strings.TrimSpace(os.Getenv("SHELL")))
	switch shell {
	case "zsh", "bash", "fish":
		return shell
	}
	if runtime.GOOS == "darwin" {
		return "zsh"
	}
	return "bash"
}

// Install writes the completion script and, when needed, a sourced block in the
// shell rc file. Idempotent: re-running updates the script and avoids duplicate rc lines.
func Install(shell string, home string, gen Generator) (*Result, error) {
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home: %w", err)
		}
	}
	shell = strings.TrimSpace(shell)
	if shell == "" {
		shell = DetectShell()
	}
	if gen == nil {
		return nil, fmt.Errorf("generator obrigatório")
	}

	switch shell {
	case "zsh":
		return installZsh(home, gen)
	case "bash":
		return installBash(home, gen)
	case "fish":
		return installFish(home, gen)
	default:
		return nil, fmt.Errorf("shell não suportado: %s (use zsh, bash ou fish)", shell)
	}
}

func installZsh(home string, gen Generator) (*Result, error) {
	dir := filepath.Join(home, ".zfunc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	script := filepath.Join(dir, "_"+binName)
	if err := writeGenerated(script, "zsh", gen); err != nil {
		return nil, err
	}

	rc := filepath.Join(home, ".zshrc")
	block := strings.Join([]string{
		beginMarker,
		`fpath=("$HOME/.zfunc" $fpath)`,
		"autoload -Uz compinit",
		"compinit",
		endMarker,
	}, "\n") + "\n"

	updated, err := ensureRCBlock(rc, block)
	if err != nil {
		return nil, err
	}
	return &Result{
		Shell:      "zsh",
		ScriptPath: script,
		RCPath:     rc,
		RCUpdated:  updated,
		ReloadHint: "exec zsh   # ou abra um novo terminal",
	}, nil
}

func installBash(home string, gen Generator) (*Result, error) {
	dir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	script := filepath.Join(dir, binName)
	if err := writeGenerated(script, "bash", gen); err != nil {
		return nil, err
	}

	rc := filepath.Join(home, ".bashrc")
	block := strings.Join([]string{
		beginMarker,
		`if [ -f "$HOME/.local/share/bash-completion/completions/lightyear" ]; then`,
		`  . "$HOME/.local/share/bash-completion/completions/lightyear"`,
		"fi",
		endMarker,
	}, "\n") + "\n"

	updated, err := ensureRCBlock(rc, block)
	if err != nil {
		return nil, err
	}
	return &Result{
		Shell:      "bash",
		ScriptPath: script,
		RCPath:     rc,
		RCUpdated:  updated,
		ReloadHint: "source ~/.bashrc   # ou abra um novo terminal",
	}, nil
}

func installFish(home string, gen Generator) (*Result, error) {
	dir := filepath.Join(home, ".config", "fish", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	script := filepath.Join(dir, binName+".fish")
	if err := writeGenerated(script, "fish", gen); err != nil {
		return nil, err
	}
	// Fish auto-loads ~/.config/fish/completions — no rc edit needed.
	return &Result{
		Shell:      "fish",
		ScriptPath: script,
		ReloadHint: "abra um novo shell fish (completions carregam automaticamente)",
	}, nil
}

func writeGenerated(path, shell string, gen Generator) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("escrever %s: %w", path, err)
	}
	defer f.Close()
	if err := gen(shell, f); err != nil {
		return fmt.Errorf("gerar completion %s: %w", shell, err)
	}
	return nil
}

// ensureRCBlock inserts or replaces the marked block in path.
// Returns whether the file content changed.
func ensureRCBlock(path, block string) (bool, error) {
	var existing string
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
	} else {
		existing = string(data)
	}

	if strings.Contains(existing, beginMarker) && strings.Contains(existing, endMarker) {
		start := strings.Index(existing, beginMarker)
		end := strings.Index(existing, endMarker)
		if start >= 0 && end > start {
			end += len(endMarker)
			// consume trailing newline after end marker
			for end < len(existing) && (existing[end] == '\n' || existing[end] == '\r') {
				end++
			}
			updated := existing[:start] + block + existing[end:]
			if updated == existing {
				return false, nil
			}
			return true, writeFile(path, updated)
		}
	}

	var b strings.Builder
	b.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		b.WriteByte('\n')
	}
	if existing != "" {
		b.WriteByte('\n')
	}
	b.WriteString(block)
	return true, writeFile(path, b.String())
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
