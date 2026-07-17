package cmd

import (
	"fmt"

	"github.com/joaodiniz/42cli/internal/tui"
	"github.com/spf13/cobra"
)

func newMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Mostra o seu perfil na 42",
		Long:  "Exibe o perfil do usuário autenticado: nível, wallet, pontos de avaliação, campus e localização.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			user, err := deps.Users.Me(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderUser(user))
			return nil
		},
	}
}
