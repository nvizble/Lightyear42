package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nvizble/Lightyear42/internal/auth"
	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const intraOAuthAppURL = "https://profile.intra.42.fr/oauth/applications/new"

func newSetupCmd() *cobra.Command {
	var (
		clientID     string
		clientSecret string
		noBrowser    bool
		force        bool
		noCompletion bool
		completionSh string
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configura client_id e client_secret da aplicação OAuth na Intra",
		Long: `Assistente interativo para registrar o lightyear na Intra 42.

Mostra o passo a passo para criar a aplicação OAuth, abre o navegador
na página de criação (opcional) e grava UID/Secret no config.yaml.

No fim, configura automaticamente o autocomplete do shell ($SHELL).

Também aceita flags para uso não-interativo:
  lightyear setup --client-id UID --client-secret SECRET`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSetup(cmd, setupOptions{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				NoBrowser:    noBrowser,
				Force:        force,
				NoCompletion: noCompletion,
				Shell:        completionSh,
			})
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", "", "UID da aplicação OAuth (pula o prompt)")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Secret da aplicação OAuth (pula o prompt)")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "não abre o navegador automaticamente")
	cmd.Flags().BoolVar(&force, "force", false, "sobrescreve credenciais existentes sem perguntar")
	cmd.Flags().BoolVar(&noCompletion, "no-completion", false, "não instala autocomplete do shell")
	cmd.Flags().StringVar(&completionSh, "shell", "", "shell para autocomplete (zsh, bash, fish); default: $SHELL")

	return cmd
}

type setupOptions struct {
	ClientID     string
	ClientSecret string
	NoBrowser    bool
	Force        bool
	NoCompletion bool
	Shell        string
}

func runSetup(cmd *cobra.Command, opts setupOptions) error {
	out := cmd.OutOrStdout()
	defaults := config.Default()

	printSetupGuide(out, defaults.RedirectURI)

	if !opts.NoBrowser {
		fmt.Fprintln(out, "Abrindo a página de criação da aplicação na Intra…")
		if err := auth.OpenBrowser(intraOAuthAppURL); err != nil {
			fmt.Fprintf(out, "Não foi possível abrir o navegador (%v).\nAcesse manualmente:\n  %s\n\n", err, intraOAuthAppURL)
		} else {
			fmt.Fprintln(out)
		}
	} else {
		fmt.Fprintf(out, "Abra no navegador:\n  %s\n\n", intraOAuthAppURL)
	}

	if config.HasCredentials(rootCfg) && !opts.Force {
		hasFlags := opts.ClientID != "" || opts.ClientSecret != ""
		switch {
		case hasFlags && opts.ClientID != "" && opts.ClientSecret != "":
			return fmt.Errorf("já existem credenciais em %s; use --force para sobrescrever", configFileHint())
		case hasFlags:
			return fmt.Errorf("informe --client-id e --client-secret juntos (e --force para sobrescrever)")
		case isInteractive():
			fmt.Fprintf(out, "Já existem credenciais em %s.\n", configFileHint())
			ok, err := promptConfirm(out, os.Stdin, "Sobrescrever? [y/N] ")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(out, "Setup cancelado. Nada foi alterado.")
				return nil
			}
		default:
			return fmt.Errorf("já existem credenciais; use --force ou rode em um terminal interativo")
		}
	}

	id := strings.TrimSpace(opts.ClientID)
	secret := strings.TrimSpace(opts.ClientSecret)

	switch {
	case id != "" && secret != "":
		// non-interactive / flags
	case id != "" || secret != "":
		return fmt.Errorf("informe --client-id e --client-secret juntos")
	default:
		if !isInteractive() {
			return fmt.Errorf("terminal não-interativo: use --client-id e --client-secret")
		}
		var err error
		id, err = promptLine(out, os.Stdin, "UID (client_id): ")
		if err != nil {
			return err
		}
		secret, err = promptSecret(out, "Secret (client_secret): ")
		if err != nil {
			return err
		}
	}

	paths, err := config.SaveCredentials(id, secret)
	if err != nil {
		return err
	}

	// Refresh in-process config so a subsequent command in the same process sees it.
	rootCfg.ClientID = id
	rootCfg.ClientSecret = secret
	if rootCfg.APIBaseURL == "" {
		rootCfg.APIBaseURL = defaults.APIBaseURL
	}
	if rootCfg.RedirectURI == "" {
		rootCfg.RedirectURI = defaults.RedirectURI
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Credenciais salvas em %s (permissões 0600).\n", paths.ConfigFile)

	if !opts.NoCompletion {
		fmt.Fprintln(out)
		res, err := installShellCompletion(cmd.Root(), opts.Shell)
		if err != nil {
			fmt.Fprintf(out, "aviso: não foi possível configurar autocomplete: %v\n", err)
			fmt.Fprintln(out, "Pode tentar depois: lightyear completion install")
		} else {
			printCompletionInstall(out, res)
		}
	}

	fmt.Fprintln(out, "Próximo passo: lightyear login")
	return nil
}

func printSetupGuide(out io.Writer, redirectURI string) {
	fmt.Fprintln(out, "=== lightyear setup ===")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "1. Crie uma aplicação OAuth na Intra 42.")
	fmt.Fprintln(out, "2. Preencha:")
	fmt.Fprintln(out, "   - Name: lightyear (ou outro nome)")
	fmt.Fprintln(out, "   - Redirect URI:")
	fmt.Fprintf(out, "     %s\n", redirectURI)
	fmt.Fprintln(out, "   - Scopes: marque public e projects")
	fmt.Fprintln(out, "     (projects é necessário para slots)")
	fmt.Fprintln(out, "3. Após salvar, copie o UID e o Secret.")
	fmt.Fprintln(out)
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func promptLine(out io.Writer, in io.Reader, label string) (string, error) {
	fmt.Fprint(out, label)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("entrada vazia")
	}
	value := strings.TrimSpace(scanner.Text())
	if value == "" {
		return "", fmt.Errorf("valor obrigatório")
	}
	return value, nil
}

func promptSecret(out io.Writer, label string) (string, error) {
	fmt.Fprint(out, label)
	fd := int(os.Stdin.Fd())
	raw, err := term.ReadPassword(fd)
	fmt.Fprintln(out)
	if err != nil {
		return "", fmt.Errorf("ler secret: %w", err)
	}
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", fmt.Errorf("secret obrigatório")
	}
	return value, nil
}

func promptConfirm(out io.Writer, in io.Reader, label string) (bool, error) {
	fmt.Fprint(out, label)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes" || answer == "s" || answer == "sim", nil
}
