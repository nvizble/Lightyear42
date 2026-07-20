package cmd

import (
	"fmt"
	"time"

	"github.com/nvizble/Lightyear42/internal/auth"
	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/spf13/cobra"
)

func newEvaluationsCmd() *cobra.Command {
	var open bool

	cmd := &cobra.Command{
		Use:     "evaluations",
		Aliases: []string{"evals"},
		Short:   "Lista suas próximas avaliações agendadas",
		Long: `Mostra as avaliações futuras do usuário autenticado — como avaliador
ou como avaliado — ordenadas da mais próxima para a mais distante.

Com --open, abre no navegador a página da Intra para iniciar/preencher a
próxima avaliação em que você é o avaliador (prioriza as que já começaram).

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

			if open {
				return openEvaluation(cmd, deps)
			}

			evaluations, err := deps.Users.UpcomingEvaluations(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderEvaluations(evaluations, me.Login, 0))
			return nil
		},
	}

	cmd.Flags().BoolVar(&open, "open", false, "abre no navegador a próxima avaliação para preencher (como avaliador)")
	return cmd
}

func openEvaluation(cmd *cobra.Command, deps *appDeps) error {
	st, err := deps.Users.OpenableEvaluation(cmd.Context(), time.Now())
	if err != nil {
		return err
	}

	url := st.FillURL()
	if url == "" {
		return fmt.Errorf("avaliação sem id válido")
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Abrindo avaliação #%d", st.ID)
	if st.Team.Name != "" {
		fmt.Fprintf(out, " (%s)", st.Team.Name)
	}
	fmt.Fprintln(out)
	if !st.HasStarted(time.Now()) {
		fmt.Fprintln(out, "aviso: o horário ainda não chegou — a Intra pode bloquear o início.")
	}
	fmt.Fprintln(out, url)

	return auth.OpenBrowser(url)
}
