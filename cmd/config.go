package cmd

import (
	"fmt"

	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Gerencia a configuração da CLI",
		Long: `Exibe e inspeciona a configuração local da 42 CLI.

A configuração é carregada de:
  1. Arquivo YAML em $XDG_CONFIG_HOME/42cli/config.yaml
  2. Variáveis de ambiente com prefixo FORTYTWO_
  3. Valores padrão internos`,
	}

	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigShowCmd())

	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Mostra o caminho do arquivo de configuração",
		Long:  "Imprime o caminho absoluto do arquivo config.yaml usado pela CLI.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := config.ResolvePaths()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), paths.ConfigFile)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Mostra a configuração efetiva",
		Long: `Exibe a configuração efetiva após mesclar arquivo, ambiente e defaults.

O client_secret é mascarado por segurança.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := config.ResolvePaths()
			if err != nil {
				return err
			}

			secret := "(vazio)"
			if rootCfg.ClientSecret != "" {
				secret = "********"
			}

			clientID := rootCfg.ClientID
			if clientID == "" {
				clientID = "(vazio)"
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "config_file:   %s\n", paths.ConfigFile)
			fmt.Fprintf(out, "cache_dir:     %s\n", paths.CacheDir)
			fmt.Fprintf(out, "data_dir:      %s\n", paths.DataDir)
			fmt.Fprintf(out, "api_base_url:  %s\n", rootCfg.APIBaseURL)
			fmt.Fprintf(out, "redirect_uri:  %s\n", rootCfg.RedirectURI)
			fmt.Fprintf(out, "client_id:     %s\n", clientID)
			fmt.Fprintf(out, "client_secret: %s\n", secret)
			return nil
		},
	}
}
