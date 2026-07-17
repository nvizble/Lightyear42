package cmd

import (
	"fmt"

	"github.com/joaodiniz/42cli/internal/tui"
	"github.com/spf13/cobra"
)

func newProjectsCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "projects [login]",
		Short: "Lista os projetos de um usuário da 42",
		Long: `Lista os projetos do usuário autenticado (ou de outro login), com status e nota.

Por padrão mostra apenas o cursus principal; use --all para incluir
piscine e outros cursus.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			login := ""
			if len(args) == 1 {
				login = args[0]
			}

			projects, err := deps.Users.Projects(cmd.Context(), login, all)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderProjects(projects))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "inclui todos os cursus (piscine, eventos, ...)")

	return cmd
}
