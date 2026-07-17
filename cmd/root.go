// Package cmd implements the Cobra command tree for the 42 CLI.
// Commands parse flags/args and delegate business logic to internal services.
package cmd

import (
	"fmt"
	"os"

	"github.com/joaodiniz/42cli/internal/config"
	"github.com/spf13/cobra"
)

// rootCfg holds the loaded configuration for the current process.
// Populated in PersistentPreRunE; commands must not mutate it in place.
var rootCfg config.Config

// NewRootCmd builds the root command and registers subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "42",
		Short: "CLI moderna para a 42 Network",
		Long: `42 é uma CLI open source para interagir com a API oficial da 42 Network.

Gerencie autenticação, perfil, projetos, exames e mais — direto do terminal.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Skip config load for version (fast path, no side effects needed).
			if cmd.Name() == "version" || cmd.Name() == "help" {
				return nil
			}
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			rootCfg = cfg
			return nil
		},
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newLoginCmd())
	root.AddCommand(newLogoutCmd())
	root.AddCommand(newMeCmd())
	root.AddCommand(newProfileCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newProjectsCmd())
	root.AddCommand(newCampusCmd())
	root.AddCommand(newFriendsCmd())
	root.AddCommand(newCacheCmd())

	return root
}

// Execute runs the root command.
func Execute() error {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return err
	}
	return nil
}
