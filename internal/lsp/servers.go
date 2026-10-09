package lsp

import (
	"context"
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
	// Download, when set, is fetched when the server isn't installed
	// (download.go).
	Download *Download
	// RootMarkers are files that mark the project root (the nearest wins).
	RootMarkers []string
}

// clangd checks C and C++. Without a compile_commands.json it uses the
// flags the 42 graders use, so the warnings match.
func clangd(languageID string, flags ...string) Server {
	s := Server{
		Name: "clangd", LanguageID: languageID, Commands: [][]string{{"clangd"}}, Hint: "instale: sudo apt install clangd",
		Options:     map[string]any{"fallbackFlags": append(flags, "-Wall", "-Wextra", "-Werror")},
		RootMarkers: []string{"compile_commands.json", "compile_flags.txt", ".git"},
	}
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		s.Download = clangdLinux
	case "darwin/amd64", "darwin/arm64":
		s.Hint = "instale as Command Line Tools: xcode-select --install"
	}
	return s
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

// command finds the server's executable: on the PATH, for Go tools in
// GOPATH/bin (often off the PATH), or its Download, fetched when needed.
func (s Server) command(ctx context.Context) ([]string, error) {
	if argv := s.installed(); argv != nil {
		return argv, nil
	}
	if s.Download == nil {
		return nil, fmt.Errorf("%s não encontrado — %s", s.Name, s.Hint)
	}
	bin, err := s.Download.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s não encontrado, e baixá-lo falhou: %w — %s", s.Name, err, s.Hint)
	}
	return []string{bin}, nil
}

// installed is the command of an installed server (nil when none).
func (s Server) installed() []string {
	for _, argv := range s.Commands {
		if p, err := exec.LookPath(argv[0]); err == nil {
			return append([]string{p}, argv[1:]...)
		}
		if p := goBin(argv[0]); p != "" {
			return append([]string{p}, argv[1:]...)
		}
	}
	return nil
}

// WillDownload reports that starting s downloads it first: it isn't
// installed, and lightyear can fetch it.
func (s Server) WillDownload() bool {
	return s.Download != nil && s.installed() == nil && !s.Download.downloaded()
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
