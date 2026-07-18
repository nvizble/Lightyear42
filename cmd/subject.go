package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/nvizble/Lightyear42/internal/auth"
	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/spf13/cobra"
)

func newSubjectCmd() *cobra.Command {
	var (
		lang   string
		force  bool
		noOpen bool
	)

	cmd := &cobra.Command{
		Use:   "subject <projeto>",
		Short: "Baixa e abre o PDF do subject de um projeto",
		Long: `Localiza o subject (PDF) do projeto na API 42, salva em cache local
e abre no visualizador padrão do sistema.

Exemplos:
  lightyear subject push_swap
  lightyear subject libft --lang fr
  lightyear subject minishell --force     # baixa de novo
  lightyear subject ft_printf --no-open   # só baixa`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			paths, err := config.ResolvePaths()
			if err != nil {
				return err
			}
			subjectsDir := filepath.Join(paths.DataDir, "subjects")

			campusID := 0
			if id, _, err := primaryCampusID(cmd.Context(), deps); err == nil {
				campusID = id
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Buscando subject de %q…\n", args[0])

			res, err := deps.Subjects.EnsureSubject(cmd.Context(), services.SubjectOptions{
				Query:    args[0],
				Lang:     lang,
				CampusID: campusID,
				Force:    force,
				Dir:      subjectsDir,
			})
			if err != nil {
				return err
			}

			if res.Cached {
				fmt.Fprintf(out, "Cache: %s\n", res.Path)
			} else {
				fmt.Fprintf(out, "Baixado: %s\n", res.Path)
			}
			fmt.Fprintf(out, "Projeto: %s (%s)  idioma: %s\n", res.Project.Name, res.Project.Slug, res.Language)

			if noOpen {
				return nil
			}
			if err := auth.OpenPath(res.Path); err != nil {
				return fmt.Errorf("abrir PDF: %w\nArquivo salvo em: %s", err, res.Path)
			}
			fmt.Fprintln(out, "Aberto no visualizador padrão.")
			return nil
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "idioma preferido do subject (en, fr, pt, …)")
	cmd.Flags().BoolVar(&force, "force", false, "baixa novamente mesmo se já existir em cache")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "apenas baixa; não abre o PDF")

	return cmd
}
