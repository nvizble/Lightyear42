package cmd

import (
	"fmt"

	"github.com/joaodiniz/42cli/internal/tui"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <termo>",
		Short: "Busca usuários da 42 por prefixo de login",
		Long: `Lista usuários cujo login começa com o termo informado.

Exemplo:
  42 search jdi        # logins que começam com "jdi"
  42 search jdi -n 30  # até 30 resultados`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			users, err := deps.Users.Search(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderUserList(users))
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "número máximo de resultados (até 100)")

	return cmd
}
