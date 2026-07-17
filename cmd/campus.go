package cmd

import (
	"fmt"
	"strconv"

	"github.com/joaodiniz/42cli/internal/services"
	"github.com/joaodiniz/42cli/internal/tui"
	"github.com/spf13/cobra"
)

func newCampusCmd() *cobra.Command {
	var campusID int
	var friendsOnly bool

	cmd := &cobra.Command{
		Use:   "campus",
		Short: "Mostra quem está online no campus, organizado como um mapa",
		Long: `Lista todos os usuários logados no campus e organiza os postos por
cluster/fileira/posto (ex.: c1r2p3), desenhando um mapa de ocupação.

Por padrão usa o seu campus primário; use --id para ver outro campus.
Com --friends, mostra apenas os logins da sua lista de amigos.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			id, name := campusID, fmt.Sprintf("Campus %d", campusID)
			if id == 0 {
				id, name, err = primaryCampusID(cmd.Context(), deps)
				if err != nil {
					return err
				}
			}

			locations, err := deps.Campus.Online(cmd.Context(), id)
			if err != nil {
				return err
			}

			if friendsOnly {
				friends, err := newFriendsService().List()
				if err != nil {
					return err
				}
				if len(friends) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Sua lista de amigos está vazia. Adicione com `42 friends add <login>`.")
					return nil
				}
				locations = services.FilterLocationsByLogin(locations, friends)
				name += " (amigos)"
			}

			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderCampusMap(name, locations, campusLayout()))
			return nil
		},
	}

	cmd.Flags().IntVar(&campusID, "id", 0, "ID do campus (padrão: seu campus primário)")
	cmd.Flags().BoolVarP(&friendsOnly, "friends", "f", false, "mostra apenas os amigos da sua lista")

	return cmd
}

// campusLayout converts the campus_layout config section (keys are cluster
// numbers as strings) into the grid map the renderer expects. Invalid
// entries are ignored, falling back to the inferred grid.
func campusLayout() map[int]tui.ClusterGrid {
	if len(rootCfg.CampusLayout) == 0 {
		return nil
	}
	layout := make(map[int]tui.ClusterGrid, len(rootCfg.CampusLayout))
	for key, grid := range rootCfg.CampusLayout {
		cluster, err := strconv.Atoi(key)
		if err != nil || cluster < 1 || grid.Rows < 1 || grid.Posts < 1 {
			continue
		}
		layout[cluster] = tui.ClusterGrid{Rows: grid.Rows, Posts: grid.Posts}
	}
	return layout
}
