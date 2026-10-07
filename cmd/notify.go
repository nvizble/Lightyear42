package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/notify"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/spf13/cobra"
)

// notifyStateFile holds the ids already announced, inside the data directory.
const notifyStateFile = "notifications.json"

// Watch interval bounds. The evaluations endpoint is cached for a minute and
// the Intra rate limit is shared with every other command, so checking more
// often than that would only burn quota.
const (
	defaultWatchInterval = 2 * time.Minute
	minWatchInterval     = time.Minute
)

func newNotifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify",
		Short: "Avisa no celular quando uma avaliação nova for agendada",
		Long: `Envia uma notificação push quando aparece uma avaliação nova na sua
agenda — como avaliador ou como avaliado.

A entrega usa o ntfy (https://ntfy.sh): instale o app no celular, assine o
tópico gerado pelo "lightyear notify setup" e pronto. O tópico funciona como
segredo — quem souber o nome recebe as suas notificações.

Fluxo típico:

  lightyear notify setup    # gera o tópico e mostra como assinar
  lightyear notify test     # confirma que chega no celular
  lightyear notify watch    # deixa rodando; avisa a cada avaliação nova

O app (só "lightyear", sem subcomando) também vigia a agenda enquanto estiver
aberto, como o watch.

Para rodar sem deixar terminal aberto, agende "lightyear notify check"
no cron (ou systemd timer) — ele checa uma vez e sai.`,
	}

	cmd.AddCommand(newNotifySetupCmd())
	cmd.AddCommand(newNotifyTestCmd())
	cmd.AddCommand(newNotifyCheckCmd())
	cmd.AddCommand(newNotifyWatchCmd())

	return cmd
}

func newNotifySetupCmd() *cobra.Command {
	var server, token string

	cmd := &cobra.Command{
		Use:   "setup [tópico]",
		Short: "Configura o envio das notificações (ntfy)",
		Long: `Guarda no config.yaml o servidor e o tópico ntfy usados para notificar.

Sem argumento, mantém o tópico já configurado ou gera um aleatório —
preferível a escolher um nome fácil, já que o tópico é o único segredo
que protege as suas notificações em servidores públicos.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := config.NewNotificationsFile()
			current, err := store.Load()
			if err != nil {
				return err
			}

			topic := current.NtfyTopic
			if len(args) == 1 {
				topic = args[0]
			}
			if topic == "" {
				if topic, err = notify.RandomTopic(); err != nil {
					return err
				}
			}

			if server == "" {
				server = current.NtfyServer
			}
			if token == "" {
				token = current.NtfyToken
			}

			sender, err := notify.NewNtfy(server, topic, token)
			if err != nil {
				return err
			}

			if err := store.Save(config.Notifications{
				NtfyServer: sender.Server(),
				NtfyTopic:  sender.Topic(),
				NtfyToken:  token,
			}); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Notificações configuradas.")
			fmt.Fprintf(out, "  servidor: %s\n", sender.Server())
			fmt.Fprintf(out, "  tópico:   %s\n", sender.Topic())
			fmt.Fprintln(out)
			fmt.Fprintln(out, "No celular:")
			fmt.Fprintln(out, "  1. instale o app ntfy (App Store / Play Store / F-Droid)")
			fmt.Fprintf(out, "  2. assine o tópico %q\n", sender.Topic())
			fmt.Fprintf(out, "     ou abra %s\n", sender.SubscribeURL())
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Depois rode: lightyear notify test")
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "servidor ntfy (padrão: "+notify.DefaultNtfyServer+")")
	cmd.Flags().StringVar(&token, "token", "", "token de acesso, para tópicos protegidos")
	return cmd
}

func newNotifyTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Envia uma notificação de teste",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sender, err := newNotifySender()
			if err != nil {
				return err
			}
			if err := sender.Send(cmd.Context(), notify.TestNotification()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Notificação de teste enviada para %s.\n", sender.SubscribeURL())
			return nil
		},
	}
}

func newNotifyCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Checa uma vez e notifica as avaliações novas",
		Long: `Busca a agenda, notifica o que ainda não foi avisado e sai.

Feito para agendadores (cron, systemd timer, launchd):

  */5 * * * * /usr/local/bin/lightyear notify check >/dev/null

Na primeira execução nada é enviado: a agenda atual vira a linha de base,
para você não receber uma rajada de avisos sobre avaliações já conhecidas.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, cleanup, err := newNotifyService(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			report, err := svc.Check(cmd.Context(), time.Now())
			printNotifyReport(cmd.OutOrStdout(), report)
			return err
		},
	}
}

func newNotifyWatchCmd() *cobra.Command {
	var interval time.Duration

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Fica monitorando e notifica cada avaliação nova",
		Long: `Checa a agenda a cada intervalo e notifica o que for novo.

Roda até você interromper (Ctrl+C). Falhas pontuais de rede ou da API são
registradas e a checagem seguinte continua normalmente.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if interval < minWatchInterval {
				return fmt.Errorf("intervalo mínimo é %s (a agenda é cacheada por 1 minuto)", minWatchInterval)
			}

			svc, cleanup, err := newNotifyService(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Monitorando avaliações a cada %s. Ctrl+C para sair.\n", interval)

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				report, err := svc.Check(ctx, time.Now())
				if ctx.Err() != nil {
					fmt.Fprintln(out, "Monitoramento encerrado.")
					return nil
				}
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "aviso: falha na checagem:", err)
				}
				printNotifyReport(out, report)

				select {
				case <-ctx.Done():
					fmt.Fprintln(out, "Monitoramento encerrado.")
					return nil
				case <-ticker.C:
				}
			}
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", defaultWatchInterval, "intervalo entre checagens (mínimo "+minWatchInterval.String()+")")
	return cmd
}

// newNotifySender builds the push sender from the loaded configuration.
func newNotifySender() (*notify.Ntfy, error) {
	cfg := rootCfg.Notifications
	return notify.NewNtfy(cfg.NtfyServer, cfg.NtfyTopic, cfg.NtfyToken)
}

// notifyStatePath returns the file tracking which evaluations were announced.
func notifyStatePath() (string, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.DataDir, notifyStateFile), nil
}

// newNotifyService wires the notification service and returns the cleanup of
// the underlying API dependencies.
func newNotifyService(ctx context.Context) (*services.NotifyService, func(), error) {
	if _, err := newNotifySender(); err != nil {
		return nil, nil, err // not set up: say so before any login error
	}
	deps, cleanup, err := newDeps(ctx)
	if err != nil {
		return nil, nil, err
	}
	svc, err := notifyServiceFor(deps)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return svc, cleanup, nil
}

// notifyServiceFor wires the notification service on existing API
// dependencies; it fails when notify isn't set up (notify.ErrNoTopic).
func notifyServiceFor(deps *appDeps) (*services.NotifyService, error) {
	sender, err := newNotifySender()
	if err != nil {
		return nil, err
	}
	statePath, err := notifyStatePath()
	if err != nil {
		return nil, err
	}
	return services.NewNotifyService(deps.Users, sender, notify.NewFileStore(statePath)), nil
}

// printNotifyReport writes a one-line summary of a check.
func printNotifyReport(out io.Writer, report services.NotifyReport) {
	switch {
	case report.Baseline:
		fmt.Fprintf(out, "Linha de base registrada: %d avaliação(ões) na agenda. Só as próximas novidades serão notificadas.\n", report.Scheduled)
	case len(report.Notified) == 0:
		fmt.Fprintf(out, "[%s] nenhuma avaliação nova (%d na agenda).\n", time.Now().Format("15:04:05"), report.Scheduled)
	default:
		for _, st := range report.Notified {
			fmt.Fprintf(out, "[%s] notificado: %s\n", time.Now().Format("15:04:05"), notifiedLine(st))
		}
	}
}

// notifiedLine describes one announced evaluation for the terminal log.
func notifiedLine(st models.ScaleTeam) string {
	when := "horário a confirmar"
	if st.BeginAt != nil {
		when = st.BeginAt.Local().Format("02/01 15:04")
	}
	name := st.Team.Name
	if name == "" {
		name = fmt.Sprintf("avaliação #%d", st.ID)
	}
	return fmt.Sprintf("%s — %s", when, name)
}
