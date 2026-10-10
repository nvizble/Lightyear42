package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/tester"
	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/nvizble/Lightyear42/internal/tui/editorview"
	"github.com/nvizble/Lightyear42/internal/tui/testview"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	var (
		run  bool
		all  bool
		only []string
	)
	cmd := &cobra.Command{
		Use:   "test <projeto>",
		Short: "Testa o seu projeto da 42 (libft…) com centenas de casos",
		Long: `Abre o editor do lightyear na pasta do projeto, com os arquivos numa árvore
à esquerda e um botão ▶ Rodar testes em cima. Rode na raiz do projeto.

Os testes seguem o subject. Projetos em C: README, norminette (se instalada),
as regras do Makefile (sem relink, flags), funções proibidas e variáveis
globais e, para cada função, vários casos — inclusive os de borda. Módulos de
Python: arquivos entregues, flake8, mypy, type hints, funções autorizadas e o
comportamento de cada exercício, com os exemplos do subject e casos de borda. Cada caso roda isolado:
segfault, loop infinito, vazamento de memória, double free, escrita além do
malloc e malloc sem proteção (cada malloc falhando, um de cada vez) aparecem
como falha daquele caso. Nada é compilado na sua pasta: o lightyear usa uma cópia.

No editor:
  Space e   árvore de arquivos (enter abre, h/l fecham e abrem pastas)
  Space r   roda os testes (salva os arquivos antes)
  Space t   resultados (enter num erro abre o código dele)
O mouse funciona em tudo: clique no botão, nos arquivos e nos erros.

O born2beroot é diferente: rode dentro da VM avaliada, como root (sudo). Ele
checa a máquina (LVM criptografado, SSH na 4242 sem root, UFW, hostname,
grupos, política de senha, sudo e o monitoring.sh, conferindo cada valor que
ele mostra) e imprime no terminal.

Projetos: ` + strings.Join(projectNames(), ", ") + `.`,
		Example: `  cd ~/libft && lightyear test libft
  cd ~/ft_printf && lightyear test ft_printf
  cd ~/gnl && lightyear test get_next_line
  cd ~/python00 && lightyear test python-module-00
  sudo lightyear test born2beroot             # dentro da VM avaliada
  lightyear test libft --run                  # sem editor, só o resultado
  lightyear test libft --run --only ft_split  # só uma função`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return projectNames(), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := tester.Lookup(args[0])
			if err != nil {
				return err
			}
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			if err := project.CheckRoot(root); err != nil {
				return err
			}
			opts := tester.DefaultOptions()
			opts.Only = only
			if run || project.Machine {
				return runTests(cmd, project, root, opts, all)
			}
			return openTestView(cmd, project, root, opts)
		},
	}
	cmd.Flags().BoolVar(&run, "run", false, "roda os testes direto no terminal, sem abrir o editor (sai com erro se algum falhar)")
	cmd.Flags().BoolVar(&all, "all", false, "com --run, lista também os casos que passaram")
	cmd.Flags().StringSliceVar(&only, "only", nil, "testa só essas funções (ex.: --only ft_split,ft_itoa)")
	return cmd
}

// firstFile is what the editor shows first: the first marker (the
// Makefile), or the first file in it when it is a folder (ex0/).
func firstFile(root string, project tester.Project) string {
	first := filepath.Join(root, project.Markers[0])
	entries, err := os.ReadDir(first)
	if err != nil {
		return first
	}
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			return filepath.Join(first, e.Name())
		}
	}
	return filepath.Join(root, "README.md")
}

func projectNames() []string {
	var names []string
	for _, p := range tester.Projects() {
		names = append(names, p.Name)
	}
	return names
}

// errTestsFailed makes `lightyear test --run` exit non-zero, for scripts.
var errTestsFailed = errors.New("alguns testes falharam")

func runTests(cmd *cobra.Command, project tester.Project, root string, opts tester.Options, all bool) error {
	opts.Progress = func(step string) { fmt.Fprintf(os.Stderr, "… %s\n", step) }
	start := time.Now()
	rep, err := project.Run(cmd.Context(), root, opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), tui.RenderTestReport(rep, time.Since(start), all))
	if _, failed, _ := rep.Count(); failed > 0 {
		cmd.SilenceUsage = true
		return errTestsFailed
	}
	return nil
}

func openTestView(cmd *cobra.Command, project tester.Project, root string, opts tester.Options) error {
	ed, err := editor.Open(firstFile(root, project))
	if err != nil {
		return err
	}
	scheme, _ := config.Colorscheme()
	view := editorview.NewVim(ed).WithLSP().WithColorscheme(scheme).OnColorscheme(func(name string) { _ = config.SaveColorscheme(name) })
	model := testview.New(view, project, root, opts)
	defer model.Close()
	_, err = tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion(), tea.WithContext(cmd.Context())).Run()
	return err
}
