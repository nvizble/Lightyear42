package cmd

import (
	"fmt"
	"strconv"

	"github.com/joaodiniz/42cli/internal/services"
	"github.com/joaodiniz/42cli/internal/tui"
	"github.com/spf13/cobra"
)

func newSlotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slots",
		Short: "Gerencia seus slots de disponibilidade para avaliar",
		Long: `Lista, abre e fecha janelas em que você se declara disponível para
avaliar outros alunos.

Requer o scope OAuth "projects" na sua aplicação Intra e um novo login
após ativá-lo (` + "`lightyear logout && lightyear login`" + `).`,
		Args: cobra.NoArgs,
		RunE: runSlotsList,
	}

	cmd.AddCommand(newSlotsListCmd())
	cmd.AddCommand(newSlotsOpenCmd())
	cmd.AddCommand(newSlotsCloseCmd())
	return cmd
}

func newSlotsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista seus slots futuros",
		Args:  cobra.NoArgs,
		RunE:  runSlotsList,
	}
}

func runSlotsList(cmd *cobra.Command, _ []string) error {
	deps, cleanup, err := newDeps(cmd.Context())
	if err != nil {
		return err
	}
	defer cleanup()

	slots, err := deps.Slots.List(cmd.Context())
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), tui.RenderSlots(slots))
	return nil
}

func newSlotsOpenCmd() *cobra.Command {
	var from, to, duration string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Abre disponibilidade para avaliar (cria slots)",
		Long: `Cria um ou mais slots de 15 minutos cobrindo o intervalo informado.

Combinações:
  lightyear slots open --duration 1h
      → começa no momento mais cedo permitido (~30 min, grade de 15 min)
  lightyear slots open --from "2026-07-18 14:00" --duration 1h
  lightyear slots open --from "2026-07-18 14:00" --to "2026-07-18 15:00"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			created, err := deps.Slots.Open(cmd.Context(), services.OpenRequest{
				From:     from,
				To:       to,
				Duration: duration,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Abertos %d slot(s):\n", len(created))
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderSlots(created))
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "início (hora local, YYYY-MM-DD HH:MM); omita com só --duration")
	cmd.Flags().StringVar(&to, "to", "", "fim (hora local); exige --from; use --to ou --duration")
	cmd.Flags().StringVar(&duration, "duration", "", "duração (ex.: 30m, 1h); sozinha = começa o mais cedo possível")

	return cmd
}

func newSlotsCloseCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "close [id]",
		Short: "Fecha um slot livre pelo id, ou todos com --all",
		Long: `Remove slots futuros livres (sem avaliação agendada).

  lightyear slots close 12345
  lightyear slots close --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return fmt.Errorf("use --all ou um id, não os dois")
			}
			if !all && len(args) == 0 {
				return fmt.Errorf("informe o id do slot ou use --all")
			}

			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			if all {
				closed, skipped, err := deps.Slots.CloseAll(cmd.Context())
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Fechados %d slot(s)", closed)
				if skipped > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), " (%d agendado(s) mantido(s))", skipped)
				}
				fmt.Fprintln(cmd.OutOrStdout(), ".")
				return nil
			}

			id, err := strconv.Atoi(args[0])
			if err != nil || id < 1 {
				return fmt.Errorf("id inválido: %q", args[0])
			}
			if err := deps.Slots.Close(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Slot %d fechado.\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "fecha todos os slots futuros livres")
	return cmd
}
