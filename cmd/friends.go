package cmd

import (
	"fmt"

	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/spf13/cobra"
)

// newFriendsService builds the friends service over the config file store.
func newFriendsService() *services.FriendsService {
	return services.NewFriendsService(config.NewFriendsFile())
}

func newFriendsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "friends",
		Short: "Gerencia sua lista local de amigos",
		Long: `Mantém uma lista de logins favoritos no config.yaml da CLI.

A API pública da 42 não tem conceito de amizades; esta lista é local
e alimenta filtros como "lightyear campus --friends" e "lightyear friends online".`,
	}

	cmd.AddCommand(newFriendsAddCmd())
	cmd.AddCommand(newFriendsRemoveCmd())
	cmd.AddCommand(newFriendsListCmd())
	cmd.AddCommand(newFriendsOnlineCmd())

	return cmd
}

func newFriendsAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <login>",
		Short: "Adiciona um login à lista de amigos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			added, err := newFriendsService().Add(args[0])
			if err != nil {
				return err
			}
			if !added {
				fmt.Fprintf(cmd.OutOrStdout(), "%s já está na sua lista de amigos.\n", args[0])
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s adicionado à lista de amigos.\n", args[0])
			return nil
		},
	}
}

func newFriendsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <login>",
		Short: "Remove um login da lista de amigos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			removed, err := newFriendsService().Remove(args[0])
			if err != nil {
				return err
			}
			if !removed {
				fmt.Fprintf(cmd.OutOrStdout(), "%s não estava na sua lista de amigos.\n", args[0])
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s removido da lista de amigos.\n", args[0])
			return nil
		},
	}
}

func newFriendsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Mostra a lista de amigos",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			friends, err := newFriendsService().List()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderFriends(friends))
			return nil
		},
	}
}

func newFriendsOnlineCmd() *cobra.Command {
	var campusID int

	cmd := &cobra.Command{
		Use:   "online",
		Short: "Mostra quais amigos estão online no campus e onde",
		Long:  "Cruza sua lista de amigos com as sessões ativas do campus (padrão: seu campus primário).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			friends, err := newFriendsService().List()
			if err != nil {
				return err
			}
			if len(friends) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Sua lista de amigos está vazia. Adicione com `lightyear friends add <login>`.")
				return nil
			}

			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			id := campusID
			if id == 0 {
				id, _, err = primaryCampusID(cmd.Context(), deps)
				if err != nil {
					return err
				}
			}

			locations, err := deps.Campus.Online(cmd.Context(), id)
			if err != nil {
				return err
			}

			online := services.FilterLocationsByLogin(locations, friends)
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderFriendsOnline(online, len(friends)))
			return nil
		},
	}

	cmd.Flags().IntVar(&campusID, "id", 0, "ID do campus (padrão: seu campus primário)")

	return cmd
}
