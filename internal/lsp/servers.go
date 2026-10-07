package lsp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Server describes how to run a language server.
type Server struct {
	Name       string     // shown to the user
	LanguageID string     // LSP languageId of the documents
	Commands   [][]string // candidates in order of preference; the first found runs
	Hint       string     // how to install it, shown when it's missing
	Options    any        // initializationOptions
	// RootMarkers are files that mark the project root (the nearest wins).
	RootMarkers []string
}

// clangd checks C and C++. Without a compile_commands.json it uses the
// flags the 42 graders use, so the warnings match.
func clangd(languageID string, flags ...string) Server {
	hint := "instale: sudo apt install clangd"
	if runtime.GOOS == "darwin" {
		hint = "instale as Command Line Tools: xcode-select --install"
	}
	return Server{
		Name: "clangd", LanguageID: languageID, Commands: [][]string{{"clangd"}}, Hint: hint,
		Options:     map[string]any{"fallbackFlags": append(flags, "-Wall", "-Wextra", "-Werror")},
		RootMarkers: []string{"compile_commands.json", "compile_flags.txt", ".git"},
	}
}

var (
	serverC   = clangd("c", "-xc")
	serverCpp = clangd("cpp", "-xc++", "-std=c++98")

	servers = map[string]Server{
		".c": serverC, ".h": serverC,
		".cpp": serverCpp, ".cc": serverCpp, ".cxx": serverCpp, ".hpp": serverCpp, ".hh": serverCpp, ".hxx": serverCpp,
		".go": {
			Name: "gopls", LanguageID: "go", Commands: [][]string{{"gopls"}},
			Hint:        "instale: go install golang.org/x/tools/gopls@latest",
			RootMarkers: []string{"go.work", "go.mod", ".git"},
		},
		".py": {
			Name: "pyright", LanguageID: "python",
			Commands:    [][]string{{"pyright-langserver", "--stdio"}, {"pylsp"}},
			Hint:        "instale: npm i -g pyright (ou pip install python-lsp-server)",
			RootMarkers: []string{"pyproject.toml", "setup.py", "setup.cfg", ".git"},
		},
		".rs": {
			Name: "rust-analyzer", LanguageID: "rust", Commands: [][]string{{"rust-analyzer"}},
			Hint:        "instale: rustup component add rust-analyzer",
			RootMarkers: []string{"Cargo.toml", ".git"},
		},
	}
)

// ServerFor returns the server for the file at path; false when there is
// none for its language.
func ServerFor(path string) (Server, bool) {
	s, ok := servers[strings.ToLower(filepath.Ext(path))]
	return s, ok
}

// command finds the server's executable: on the PATH or, for Go tools, in
// GOPATH/bin (often off the PATH).
func (s Server) command() ([]string, error) {
	for _, argv := range s.Commands {
		if p, err := exec.LookPath(argv[0]); err == nil {
			return append([]string{p}, argv[1:]...), nil
		}
		if p := goBin(argv[0]); p != "" {
			return append([]string{p}, argv[1:]...), nil
		}
	}
	return nil, fmt.Errorf("%s não encontrado — %s", s.Name, s.Hint)
}

func goBin(name string) string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		gopath = filepath.Join(home, "go")
	}
	p := filepath.Join(gopath, "bin", name)
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}
