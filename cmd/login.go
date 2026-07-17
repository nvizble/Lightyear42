package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/nvizble/Lightyear42/internal/auth"
	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Autentica na 42 Network via OAuth2",
		Long: `Inicia o fluxo OAuth2 da 42: abre o navegador na página de autorização
da Intra e aguarda o callback local. O token é salvo no keyring do sistema
(Keychain, Secret Service ou Windows Credential Manager) — nunca em arquivo.

Pré-requisito: rode "lightyear setup" (ou defina client_id/client_secret
no config.yaml / FORTYTWO_CLIENT_ID e FORTYTWO_CLIENT_SECRET).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rootCfg.ClientID == "" || rootCfg.ClientSecret == "" {
				return fmt.Errorf(
					"client_id/client_secret não configurados.\n\n"+
						"Rode: lightyear setup\n"+
						"(ou configure %s / FORTYTWO_CLIENT_ID e FORTYTWO_CLIENT_SECRET)",
					configFileHint())
			}

			out := cmd.OutOrStdout()
			flow := auth.NewFlow(rootCfg, auth.FlowOptions{
				OnAuthURL: func(url string) {
					fmt.Fprintf(out, "Abrindo o navegador para autorizar o lightyear...\n\n")
					fmt.Fprintf(out, "Se o navegador não abrir, acesse:\n  %s\n\n", url)
					fmt.Fprintln(out, "Aguardando autorização...")
				},
			})
			svc := services.NewAuthService(flow, auth.NewKeyringStore())

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			token, err := svc.Login(ctx)
			if err != nil {
				return err
			}

			fmt.Fprintln(out)
			fmt.Fprintln(out, "Login concluído. Token salvo com segurança no keyring do sistema.")
			if !token.Expiry.IsZero() {
				fmt.Fprintf(out, "Access token expira em %s (renovação automática).\n",
					time.Until(token.Expiry).Round(time.Second))
			}
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 3*time.Minute,
		"tempo máximo de espera pela autorização no navegador")

	return cmd
}

// configFileHint returns the config file path for error messages,
// falling back to a generic hint if paths cannot be resolved.
func configFileHint() string {
	paths, err := config.ResolvePaths()
	if err != nil {
		return "~/.config/42cli/config.yaml"
	}
	return paths.ConfigFile
}
