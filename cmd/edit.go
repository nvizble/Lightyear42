package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/tui/editorview"
	"github.com/spf13/cobra"
)

// newEditCmd opens the embedded editor on a file. Hidden while the editor is
// experimental (see internal/editor/DESIGN.md).
func newEditCmd() *cobra.Command {
	var plain bool
	cmd := &cobra.Command{
		Use:   "edit <arquivo>",
		Short: "Editor embutido (experimental)",
		Long: `Abre o editor embutido do lightyear, com edição modal estilo Vim:
i/a/o/I/A/O entram no INSERT, esc volta ao NORMAL; movimentos hjkl, w b e,
0 ^ $, gg G; operador d com qualquer movimento (dw, d$, dG, dd) e contadores
(3j, 3dd, 2d3w); x apaga, u e Ctrl-r desfazem/refazem; v e V selecionam
(caracteres ou linhas, também arrastando o mouse), o troca a ponta e d apaga
a seleção; :w salva, :wq salva e sai, :q! sai sem salvar. Ctrl-S e Ctrl-Q
também salvam e saem.
Use --plain para o editor sem modos.`,
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ed, err := editor.Open(args[0])
			if err != nil {
				return err
			}
			model := editorview.NewVim(ed)
			if plain {
				model = editorview.New(ed)
			}
			_, err = tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithContext(cmd.Context())).Run()
			return err
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "editor sem modos (sem os comandos estilo Vim)")
	return cmd
}
