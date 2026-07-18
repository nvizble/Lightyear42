package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nvizble/Lightyear42/internal/auth"
	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/services"
)

func newSubjectCmd() *cobra.Command {
	var (
		lang      string
		force     bool
		noOpen    bool
		httpDebug bool
		pdfID     int
	)

	cmd := &cobra.Command{
		Use:   "subject <projeto>",
		Short: "Baixa e abre o PDF do subject de um projeto",
		Long: `Resolve o projeto (prioriza a sua inscrição em /me), obtém o PDF do subject
no CDN público da Intra e abre no visualizador padrão.

A API pública NÃO expõe attachments/sessions do PDF (HTTP 403). Por isso o
comando usa, nesta ordem:

  1. --pdf-id <n> (ex.: 189890 em cdn.intra.42.fr/pdf/pdf/189890/…)
  2. cache local em ~/.local/share/42cli/subjects/index.json
  3. descoberta na página HTML do projeto na Intra (quando acessível)

Na primeira vez, o mais fiável é passar o id uma vez; fica gravado no índice.

Exemplos:
  lightyear subject push_swap --pdf-id 189890
  lightyear subject push_swap
  lightyear subject 42next-push_swap --lang fr --no-open`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			paths, err := config.ResolvePaths()
			if err != nil {
				return err
			}
			dir := filepath.Join(paths.DataDir, "subjects")

			deps, cleanup, err := newDeps(cmd.Context(), depsOptions{HTTPDebug: httpDebug})
			if err != nil {
				return err
			}
			defer cleanup()

			fmt.Fprintf(os.Stderr, "Buscando subject de %q…\n", query)

			ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Minute)
			defer cancel()

			res, err := deps.Subjects.EnsureSubject(ctx, services.SubjectOptions{
				Query: query,
				Dir:   dir,
				Lang:  lang,
				Force: force,
				PDFID: pdfID,
			})
			if err != nil {
				if errors.Is(err, services.ErrSubjectPDFUnknown) && res != nil && res.Project.Slug != "" {
					intra := "https://projects.intra.42.fr/projects/" + res.Project.Slug
					fmt.Fprintf(os.Stderr, "\nNão foi possível obter o PDF pela API (attachments 403).\n")
					fmt.Fprintf(os.Stderr, "1) Abra o projeto na Intra e copie o id do CDN:\n   %s\n", intra)
					fmt.Fprintf(os.Stderr, "2) Rode: lightyear subject %s --pdf-id <id>\n", query)
					fmt.Fprintf(os.Stderr, "   (ex.: push_swap → --pdf-id 189890)\n")
					_ = auth.OpenBrowser(intra)
				}
				return err
			}

			if res.Cached {
				fmt.Fprintf(os.Stderr, "Usando PDF em cache: %s\n", res.Path)
			} else {
				fmt.Fprintf(os.Stderr, "Subject salvo em %s\n", res.Path)
			}

			if noOpen {
				fmt.Println(res.Path)
				return nil
			}
			if err := auth.OpenBrowser(res.Path); err != nil {
				fmt.Fprintf(os.Stderr, "aviso: não foi possível abrir o PDF: %v\n", err)
				fmt.Println(res.Path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "en", "idioma do subject (en, fr, …)")
	cmd.Flags().BoolVar(&force, "force", false, "baixa de novo mesmo com PDF em cache")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "só baixa/imprime o caminho, sem abrir o PDF")
	cmd.Flags().BoolVar(&httpDebug, "http-debug", false, "imprime método, URL e status de cada pedido HTTP (stderr)")
	cmd.Flags().IntVar(&pdfID, "pdf-id", 0, "id numérico do PDF no CDN (cdn.intra.42.fr/pdf/pdf/<id>/…)")
	return cmd
}
