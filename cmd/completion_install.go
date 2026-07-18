package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/nvizble/Lightyear42/internal/completion"
	"github.com/spf13/cobra"
)

func attachCompletionInstall(root *cobra.Command) {
	root.InitDefaultCompletionCmd()
	for _, c := range root.Commands() {
		if c.Name() == "completion" {
			c.AddCommand(newCompletionInstallCmd())
			return
		}
	}
}

func newCompletionInstallCmd() *cobra.Command {
	var shell string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Instala o autocomplete no shell do utilizador",
		Long: `Gera o script de completion e configura o rc do shell (zsh/bash/fish).

Por defeito deteta $SHELL. Também é chamado automaticamente por lightyear setup.

Exemplos:
  lightyear completion install
  lightyear completion install --shell zsh`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := installShellCompletion(cmd.Root(), shell)
			if err != nil {
				return err
			}
			printCompletionInstall(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVar(&shell, "shell", "", "shell alvo (zsh, bash, fish); default: $SHELL")
	return cmd
}

func completionGenerator(root *cobra.Command) completion.Generator {
	return func(shell string, w io.Writer) error {
		switch shell {
		case "zsh":
			return root.GenZshCompletion(w)
		case "bash":
			return root.GenBashCompletionV2(w, true)
		case "fish":
			return root.GenFishCompletion(w, true)
		default:
			return fmt.Errorf("shell desconhecido: %s", shell)
		}
	}
}

func installShellCompletion(root *cobra.Command, shell string) (*completion.Result, error) {
	if shell == "" {
		shell = completion.DetectShell()
	}
	return completion.Install(shell, "", completionGenerator(root))
}

func printCompletionInstall(out io.Writer, res *completion.Result) {
	fmt.Fprintf(out, "Autocomplete (%s) instalado:\n", res.Shell)
	fmt.Fprintf(out, "  script: %s\n", res.ScriptPath)
	if res.RCPath != "" {
		if res.RCUpdated {
			fmt.Fprintf(out, "  rc:     %s (atualizado)\n", res.RCPath)
		} else {
			fmt.Fprintf(out, "  rc:     %s (já configurado)\n", res.RCPath)
		}
	}
	if hint := strings.TrimSpace(res.ReloadHint); hint != "" {
		fmt.Fprintf(out, "Para ativar: %s\n", hint)
	}
}
