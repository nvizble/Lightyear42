package cmd

import (
	"fmt"
	"io"
	"runtime"

	"github.com/nvizble/Lightyear42/internal/changelog"
	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Commit is set at build time via -ldflags.
var Commit = "none"

// BuildDate is set at build time via -ldflags.
var BuildDate = "unknown"

func newVersionCmd() *cobra.Command {
	var full bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Mostra a versão da CLI e as novidades",
		Long: `Exibe a versão, o commit e a data de build do binário lightyear e o
que mudou nesta versão (--changelog mostra o histórico inteiro).
"lightyear --version" faz o mesmo.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			printVersion(cmd.OutOrStdout(), full)
			return nil
		},
	}
	cmd.Flags().BoolVar(&full, "changelog", false, "mostra o changelog inteiro")
	return cmd
}

// printVersion writes the build info and what changed in this version (or
// the whole changelog). The changelog is embedded, so it works offline.
func printVersion(out io.Writer, full bool) {
	fmt.Fprintf(out, "%s %s\n", Name, Version)
	fmt.Fprintf(out, "  commit:     %s\n", Commit)
	fmt.Fprintf(out, "  built:      %s\n", BuildDate)
	fmt.Fprintf(out, "  go:         %s\n", runtime.Version())
	fmt.Fprintf(out, "  platform:   %s/%s\n\n", runtime.GOOS, runtime.GOARCH)

	releases := changelog.Embedded()
	if full {
		fmt.Fprintln(out, tui.RenderChangelog(releases))
		return
	}
	switch r, ok := changelog.Find(releases, Version); {
	case ok:
		fmt.Fprintln(out, tui.RenderRelease(r))
	case len(releases) > 0:
		// Development builds (and versions without notes) show the latest.
		fmt.Fprintln(out, "(versão sem notas no changelog; mostrando a mais recente)")
		fmt.Fprintln(out, tui.RenderRelease(releases[0]))
	}
	fmt.Fprintln(out, "\nHistórico completo: lightyear version --changelog")
}
