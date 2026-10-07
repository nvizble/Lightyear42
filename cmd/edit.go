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
		Use:   "edit <arquivo>...",
		Short: "Editor embutido (experimental)",
		Long: `Abre o editor embutido do lightyear, com edição modal estilo Vim e
cores (Tree-sitter) e erros do language server (clangd, gopls, pyright,
rust-analyzer) para C, C++, Go, Python e Rust — K mostra a assinatura, gd
vai à definição, ]d e [d andam entre os erros e o INSERT completa enquanto
você digita (ou com Ctrl-n), Tab aceita. Text objects (diw, ci", da(, yi{),
f/F/t/T com ; e ,, busca com / ? n N * (:noh apaga os destaques), . repete
a última mudança e macros com qa…q e @a. Vários arquivos viram buffers
(:e abre outro, :bn/:bp/:b N trocam, :ls lista, :bd fecha, :wa salva todos;
gd em outro arquivo abre e Ctrl-o volta) e janelas (:sp, :vsp, Ctrl-w w
troca, :close, :only). Com o language server: grn renomeia (:Rename), grr
lista as referências, gra as correções e :Format formata (C só com
.clang-format no projeto, para não quebrar a norminette):
i/a/o/I/A/O entram no INSERT, esc volta ao NORMAL; movimentos hjkl, w b e,
0 ^ $, gg G; operador d com qualquer movimento (dw, d$, dG, dd) e contadores
(3j, 3dd, 2d3w); y copia (yy, yw, Y), c muda (cw, cc, C) e p/P colam o
que foi copiado ou apagado (dd, x, D também guardam); x apaga, u e Ctrl-r
desfazem/refazem; v e V selecionam (caracteres ou linhas, também arrastando
o mouse), o troca a ponta e d, y, c e p agem na seleção; :w salva, :wq salva
e sai, :q! sai sem salvar. Ctrl-S e Ctrl-Q também salvam e saem.
Use --plain para o editor sem modos.`,
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ed, err := editor.Open(args[0])
			if err != nil {
				return err
			}
			model := editorview.NewVim(ed)
			if plain {
				model = editorview.New(ed)
			}
			// More files open in buffers; the first one shows.
			for _, path := range append(args[1:], args[0]) {
				if model, err = model.Open(path); err != nil {
					return err
				}
			}
			model = model.WithLSP()
			defer model.Close()
			_, err = tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithContext(cmd.Context())).Run()
			return err
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "editor sem modos (sem os comandos estilo Vim)")
	return cmd
}
