package cmd

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joaodiniz/42cli/internal/services"
	"github.com/joaodiniz/42cli/internal/tui"
	"github.com/spf13/cobra"
)

// dashboardMinInterval protects the API rate limit; the locations cache TTL
// is one minute, so refreshing much faster would only re-read the cache.
const dashboardMinInterval = 15 * time.Second

func newDashboardCmd() *cobra.Command {
	var campusID int
	var interval time.Duration

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Dashboard em tempo real do campus (Bubble Tea)",
		Long: `Abre uma TUI com seu perfil, a ocupação dos clusters, suas próximas
avaliações e os amigos online, atualizando automaticamente no intervalo definido.

Teclas: r atualiza na hora, q/esc sai.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if interval < dashboardMinInterval {
				return fmt.Errorf("intervalo mínimo é %s", dashboardMinInterval)
			}

			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			dashboard := services.NewDashboardService(deps.Users, deps.Campus, newFriendsService())
			model := tui.NewDashboard(dashboard, tui.DashboardOptions{
				CampusID: campusID,
				Interval: interval,
				Layout:   campusLayout(),
			})

			program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(cmd.Context()))
			_, err = program.Run()
			return err
		},
	}

	cmd.Flags().IntVar(&campusID, "id", 0, "ID do campus (padrão: seu campus primário)")
	cmd.Flags().DurationVar(&interval, "interval", time.Minute, "intervalo de atualização automática (mínimo 15s)")

	return cmd
}
