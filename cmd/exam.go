package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/spf13/cobra"
)

// newExamService wires the embedded catalog, the cc grader, the session file
// (data dir) and the workspace (~/lightyear-exam).
func newExamService() (*services.ExamService, error) {
	catalog, err := exam.Load()
	if err != nil {
		return nil, err
	}
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	store := exam.NewSessionFile(filepath.Join(paths.DataDir, "exam-session.json"))
	return services.NewExamService(catalog, exam.NewGrader(), store, filepath.Join(home, "lightyear-exam")), nil
}

func newExamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exam",
		Short: "Simulador de provas da 42 (exam rank), com correção local",
		Long: `Simula as provas da 42 no seu computador, offline.

O enunciado de cada exercício fica em ~/lightyear-exam/subjects/<exercício>/
(em inglês e português). Escreva sua solução em ~/lightyear-exam/rendu/<exercício>/
e rode "lightyear exam grademe": seu código é compilado com cc -Wall -Wextra -Werror
e comparado com uma solução de referência. Passou, vai para o próximo nível;
falhou, o trace fica em ~/lightyear-exam/traces/.

Fluxo:
  lightyear exam start            prova cronometrada (rank 02 por padrão)
  lightyear exam practice <ex>    treina um exercício, sem tempo
  lightyear exam grademe          corrige o exercício atual
  lightyear exam status           mostra exercício, nota e tempo
  lightyear exam finish           encerra a sessão (seu código fica guardado)`,
	}

	cmd.AddCommand(newExamStartCmd())
	cmd.AddCommand(newExamPracticeCmd())
	cmd.AddCommand(newExamGrademeCmd())
	cmd.AddCommand(newExamStatusCmd())
	cmd.AddCommand(newExamFinishCmd())
	cmd.AddCommand(newExamListCmd())

	return cmd
}

func newExamStartCmd() *cobra.Command {
	var (
		rank     string
		duration time.Duration
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Começa uma prova cronometrada",
		Long: `Começa uma prova no nível 1 do rank escolhido, com um exercício sorteado.
Uma entrega anterior em ~/lightyear-exam/rendu é movida para ~/lightyear-exam/history.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newExamService()
			if err != nil {
				return err
			}
			now := time.Now()
			sess, err := svc.Start(now, rank, duration)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderExamSession(sess, now))
			return nil
		},
	}

	cmd.Flags().StringVar(&rank, "rank", "02", "rank da prova (veja `lightyear exam list`)")
	cmd.Flags().DurationVar(&duration, "time", 3*time.Hour, "duração da prova (ex.: 3h, 90m)")

	return cmd
}

func newExamPracticeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "practice <exercício>",
		Short: "Treina um exercício específico, sem tempo",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			catalog, err := exam.Load()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			names := make([]string, 0, len(catalog))
			for _, ex := range catalog {
				names = append(names, ex.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newExamService()
			if err != nil {
				return err
			}
			now := time.Now()
			sess, err := svc.Practice(now, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderExamSession(sess, now))
			return nil
		},
	}
}

func newExamGrademeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "grademe",
		Short: "Corrige o exercício atual",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newExamService()
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Corrigindo…")
			now := time.Now()
			report, err := svc.Grade(cmd.Context(), now)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderGradeReport(report, now))
			return nil
		},
	}
}

func newExamStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Mostra o exercício, a nota e o tempo restante",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newExamService()
			if err != nil {
				return err
			}
			sess, err := svc.Status()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderExamSession(sess, time.Now()))
			return nil
		},
	}
}

func newExamFinishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "finish",
		Short: "Encerra a prova em andamento",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newExamService()
			if err != nil {
				return err
			}
			sess, err := svc.Finish()
			if errors.Is(err, services.ErrNoExamSession) {
				fmt.Fprintln(cmd.OutOrStdout(), "Nenhuma prova em andamento.")
				return nil
			}
			if err != nil {
				return err
			}
			if sess.Mode == exam.ModeExam {
				fmt.Fprintf(cmd.OutOrStdout(), "Prova encerrada com %d/100. Seu código continua em %s.\n",
					sess.Score, filepath.Join(sess.Workspace, "rendu"))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Prática encerrada.")
			return nil
		},
	}
}

func newExamListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista os exercícios disponíveis por rank e nível",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			catalog, err := exam.Load()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), tui.RenderExamCatalog(catalog))
			return nil
		},
	}
}
