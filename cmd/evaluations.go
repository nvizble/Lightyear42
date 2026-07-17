package cmd

import (
	"fmt"

	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/spf13/cobra"
)

func newEvaluationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "evaluations",
		Aliases: []string{"evals"},
		Short:   "Lista suas próximas avaliações agendadas",
		Long: `Mostra as avaliações futuras do usuário autenticado — como avaliador
ou como avaliado — ordenadas da mais próxima para a mais distante.

Os mesmos dados aparecem no painel "Próximas avaliações" do dashboard.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			me, err := deps.Users.Me(cmd.Context())
			if err != nil {
				return err
			}

			evaluations, err := deps.Users.UpcomingEvaluations(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderEvaluations(evaluations, me.Login, 0))
			return nil
		},
	}
}
