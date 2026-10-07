package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvizble/Lightyear42/internal/editor"
	"github.com/nvizble/Lightyear42/internal/tui/editorview"
	"github.com/spf13/cobra"
)

// newEditCmd opens the embedded editor on a file. Hidden while the editor is
// experimental (phase 1: plain editing; see internal/editor/DESIGN.md).
func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "edit <arquivo>",
		Short:  "Editor embutido (experimental)",
		Long:   "Abre o editor embutido do lightyear. Ctrl-S salva, Ctrl-Z/Ctrl-Y desfazem/refazem, Ctrl-Q sai.",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ed, err := editor.Open(args[0])
			if err != nil {
				return err
			}
			_, err = tea.NewProgram(editorview.New(ed),
				tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithContext(cmd.Context())).Run()
			return err
		},
	}
}
