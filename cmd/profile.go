package cmd

import (
	"fmt"

	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/spf13/cobra"
)

func newProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile <login>",
		Short: "Mostra o perfil de um usuário da 42",
		Long:  "Exibe o perfil público de qualquer usuário da 42 Network pelo login da Intra.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			user, err := deps.Users.Profile(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderUser(user))
			return nil
		},
	}
}
