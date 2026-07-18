package cmd

import (
	"fmt"
	"os"

	"github.com/nvizble/Lightyear42/internal/repository"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/nvizble/Lightyear42/internal/update"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newUpdateCmd() *cobra.Command {
	var (
		checkOnly bool
		force     bool
		yes       bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Atualiza o lightyear a partir do GitHub Releases",
		Long: `Consulta a release mais recente em github.com/nvizble/Lightyear42
e substitui o binário atual se houver uma versão mais nova.

Não usa Go — adequado para o campus (Go antigo) e instalações via
tarball / ~/.local/bin. Pacotes .deb continuam atualizáveis via apt.

Exemplos:
  lightyear update           # baixa e instala se houver versão nova
  lightyear update --check   # só informa se há update
  lightyear update --yes     # sem confirmação interativa
  lightyear update --force   # permite atualizar builds "dev"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, updateFlags{
				CheckOnly: checkOnly,
				Force:     force,
				Yes:       yes,
			})
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "apenas verifica se há versão nova (não baixa)")
	cmd.Flags().BoolVar(&force, "force", false, "permite atualizar builds sem versão de release (ex.: dev)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "não pede confirmação antes de instalar")

	return cmd
}

type updateFlags struct {
	CheckOnly bool
	Force     bool
	Yes       bool
}

func runUpdate(cmd *cobra.Command, flags updateFlags) error {
	out := cmd.OutOrStdout()
	ctx := cmd.Context()

	installer := &update.Installer{}
	svc := services.NewUpdateService(
		repository.NewGitHubReleases(services.DefaultGitHubOwner, services.DefaultGitHubRepo, nil),
		installer,
	)

	plan, err := svc.Check(ctx, services.UpdateOptions{
		Current: Version,
		Force:   flags.Force,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Versão atual:  %s\n", plan.Current)
	fmt.Fprintf(out, "Última release: %s\n", plan.Latest)

	if !plan.Newer {
		fmt.Fprintln(out, "Já está na versão mais recente.")
		return nil
	}

	fmt.Fprintf(out, "Asset:         %s\n", plan.Asset.Name)

	if flags.CheckOnly {
		fmt.Fprintln(out, "Há uma versão mais nova disponível. Rode: lightyear update")
		return nil
	}

	target, err := installer.TargetPath()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Destino:       %s\n", target)

	if !flags.Yes && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		ok, err := promptConfirm(out, os.Stdin, fmt.Sprintf("Atualizar %s → %s? [y/N] ", plan.Current, plan.Latest))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Update cancelado.")
			return nil
		}
	}

	fmt.Fprintln(out, "Baixando e instalando…")
	if err := svc.Apply(ctx, plan); err != nil {
		return err
	}
	fmt.Fprintf(out, "Atualizado para %s. Rode: lightyear version\n", plan.Latest)
	return nil
}
