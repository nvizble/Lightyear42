// Package cmd implements the Cobra command tree for lightyear.
// Commands parse flags/args and delegate business logic to internal services.
package cmd

import (
	"fmt"
	"os"

	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/spf13/cobra"
)

// rootCfg holds the loaded configuration for the current process.
// Populated in PersistentPreRunE; commands must not mutate it in place.
var rootCfg config.Config

// NewRootCmd builds the root command and registers subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   Name,
		Short: "CLI moderna para a 42 Network",
		Long: `lightyear é uma CLI open source para interagir com a API oficial da 42 Network.

Gerencie autenticação, perfil, projetos, campus e mais — direto do terminal.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Skip config load for commands that do not need OAuth/config,
			// including shell completion (__complete), which must stay fast/side-effect free.
			switch cmd.Name() {
			case "version", "help", "update", "completion",
				cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
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
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newSetupCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newLoginCmd())
	root.AddCommand(newLogoutCmd())
	root.AddCommand(newMeCmd())
	root.AddCommand(newProfileCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newProjectsCmd())
	root.AddCommand(newSubjectCmd())
	root.AddCommand(newEvaluationsCmd())
	root.AddCommand(newSlotsCmd())
	root.AddCommand(newCampusCmd())
	root.AddCommand(newDashboardCmd())
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
