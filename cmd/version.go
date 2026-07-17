package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Commit is set at build time via -ldflags.
var Commit = "none"

// BuildDate is set at build time via -ldflags.
var BuildDate = "unknown"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Mostra a versão da CLI",
		Long:  "Exibe a versão, o commit e a data de build do binário lightyear.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s %s\n", Name, Version)
			fmt.Fprintf(out, "  commit:     %s\n", Commit)
			fmt.Fprintf(out, "  built:      %s\n", BuildDate)
			fmt.Fprintf(out, "  go:         %s\n", runtime.Version())
			fmt.Fprintf(out, "  platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
}
