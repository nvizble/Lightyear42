package cmd

import (
	"errors"
	"fmt"

	"github.com/joaodiniz/42cli/internal/auth"
	"github.com/joaodiniz/42cli/internal/services"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Encerra a sessão e remove o token local",
		Long:  "Remove o token OAuth2 do keyring do sistema. Não revoga a autorização na Intra.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc := services.NewAuthService(nil, auth.NewKeyringStore())

			err := svc.Logout()
			if errors.Is(err, auth.ErrNoToken) {
				fmt.Fprintln(cmd.OutOrStdout(), "Nenhuma sessão ativa — você já estava deslogado.")
				return nil
			}
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Logout concluído. Token removido do keyring.")
			return nil
		},
	}
}
