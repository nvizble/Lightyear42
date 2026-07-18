package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

Requer sessão autenticada (lightyear login). Sem login, o comando recusa o acesso.

A API pública NÃO expõe attachments/sessions do PDF (HTTP 403). Por isso o
comando resolve o id do CDN nesta ordem:

  1. --pdf-id <n> (força id e grava no índice local)
  2. índice local (~/.local/share/42cli/subjects/index.json)
     — na 1ª utilização é preenchido com o catálogo embutido (~240 projetos)
  3. catálogo embutido no CLI (internal/subjects/catalog.json)
  4. descoberta na página HTML do projeto na Intra (quando acessível)

Atualizar um id sem baixar o PDF:
  lightyear subject set-id push_swap 193464

Exemplos:
  lightyear subject push_swap
  lightyear subject push_swap --pdf-id 193464
  lightyear subject set-id 42next-push_swap 193464
  lightyear subject import ./catalog.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			dir, err := subjectsDir()
			if err != nil {
				return err
			}

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
					fmt.Fprintf(os.Stderr, "2) Rode: lightyear subject set-id %s <id>\n", query)
					fmt.Fprintf(os.Stderr, "   (ex.: lightyear subject set-id push_swap 193464)\n")
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
	cmd.Flags().IntVar(&pdfID, "pdf-id", 0, "id numérico do PDF no CDN; grava no índice local")

	cmd.AddCommand(newSubjectImportCmd())
	cmd.AddCommand(newSubjectSetIDCmd())
	return cmd
}

func subjectsDir() (string, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.DataDir, "subjects"), nil
}

func newSubjectImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <ficheiro.json>",
		Short: "Importa um catálogo slug→pdf-id para o índice local",
		Long: `Faz merge de um JSON {"slug": id, ...} no índice local de subjects.
Requer lightyear login. Formato: JSON {"slug": id, ...}.

Nota: na primeira utilização de lightyear subject o catálogo embutido já é
copiado para o índice local — import só é preciso para atualizar com um JSON novo.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Gate: subjects require an authenticated session.
			deps, cleanup, err := newDeps(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			if _, err := deps.Users.Me(cmd.Context()); err != nil {
				return fmt.Errorf("%w: %v", services.ErrSubjectAuthRequired, err)
			}

			dir, err := subjectsDir()
			if err != nil {
				return err
			}
			res, err := services.ImportPDFIndex(dir, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(),
				"Índice atualizado em %s\n  novos: %d  atualizados: %d  total: %d\n",
				res.Path, res.Added, res.Updated, res.Total)
			return nil
		},
	}
}

func newSubjectSetIDCmd() *cobra.Command {
	var httpDebug bool

	cmd := &cobra.Command{
		Use:   "set-id <projeto> <pdf-id>",
		Short: "Atualiza o id CDN do subject no índice local",
		Long: `Grava ou atualiza o mapeamento slug→pdf-id no índice local, sem baixar o PDF.
Requer lightyear login.

Equivale a descobrir o id na URL do CDN
  https://cdn.intra.42.fr/pdf/pdf/<id>/en.subject.pdf
e guardá-lo para as próximas corridas de lightyear subject.

Exemplos:
  lightyear subject set-id push_swap 193464
  lightyear subject set-id 42next-minishell 123456`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			id, err := strconv.Atoi(args[1])
			if err != nil || id <= 0 {
				return fmt.Errorf("pdf-id inválido %q (use um inteiro positivo)", args[1])
			}
			dir, err := subjectsDir()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			deps, cleanup, err := newDeps(ctx, depsOptions{HTTPDebug: httpDebug})
			if err != nil {
				return err
			}
			defer cleanup()

			res, err := deps.Subjects.SetPDFID(ctx, dir, query, id)
			if err != nil {
				return err
			}
			switch res.Previous {
			case 0:
				fmt.Fprintf(cmd.OutOrStdout(),
					"Id gravado: %s → %d\n  %s\n", res.Slug, res.ID, res.Path)
			case res.ID:
				fmt.Fprintf(cmd.OutOrStdout(),
					"Id inalterado: %s → %d\n  %s\n", res.Slug, res.ID, res.Path)
			default:
				fmt.Fprintf(cmd.OutOrStdout(),
					"Id atualizado: %s → %d (antes %d)\n  %s\n",
					res.Slug, res.ID, res.Previous, res.Path)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&httpDebug, "http-debug", false, "imprime método, URL e status de cada pedido HTTP (stderr)")
	return cmd
}
