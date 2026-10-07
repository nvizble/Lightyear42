package editorview

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/syntax"
	"github.com/nvizble/Lightyear42/internal/vim"
)

// Color schemes for the code: :colorscheme dracula (or :colo) switches,
// :colorscheme alone lists them. They color the syntax classes only; the
// background stays the terminal's.

// scheme colors the code by syntax.Class (Plain stays as is).
type scheme struct {
	name   string
	styles [syntax.Type + 1]lipgloss.Style
}

// palette builds a scheme from its colors, in class order: keyword, string,
// comment (italic), number, function, type.
func palette(name string, keyword, str, comment, number, function, typ lipgloss.TerminalColor) scheme {
	s := scheme{name: name}
	s.styles[syntax.Keyword] = lipgloss.NewStyle().Foreground(keyword)
	s.styles[syntax.String] = lipgloss.NewStyle().Foreground(str)
	s.styles[syntax.Comment] = lipgloss.NewStyle().Foreground(comment).Italic(true)
	s.styles[syntax.Number] = lipgloss.NewStyle().Foreground(number)
	s.styles[syntax.Function] = lipgloss.NewStyle().Foreground(function)
	s.styles[syntax.Type] = lipgloss.NewStyle().Foreground(typ)
	return s
}

func hex(c string) lipgloss.Color { return lipgloss.Color(c) }

// schemes are the color schemes, the default first.
var schemes = []scheme{
	palette("lightyear", colorVisual, lipgloss.AdaptiveColor{Light: "28", Dark: "114"}, colorMuted,
		lipgloss.AdaptiveColor{Light: "130", Dark: "215"}, lipgloss.AdaptiveColor{Light: "25", Dark: "75"}, lipgloss.AdaptiveColor{Light: "30", Dark: "80"}),
	palette("gruvbox", hex("#fb4934"), hex("#b8bb26"), hex("#928374"), hex("#d3869b"), hex("#8ec07c"), hex("#fabd2f")),
	palette("dracula", hex("#ff79c6"), hex("#f1fa8c"), hex("#6272a4"), hex("#bd93f9"), hex("#50fa7b"), hex("#8be9fd")),
	palette("nord", hex("#81a1c1"), hex("#a3be8c"), hex("#616e88"), hex("#b48ead"), hex("#88c0d0"), hex("#8fbcbb")),
	palette("tokyonight", hex("#bb9af7"), hex("#9ece6a"), hex("#565f89"), hex("#ff9e64"), hex("#7aa2f7"), hex("#2ac3de")),
	palette("monokai", hex("#f92672"), hex("#e6db74"), hex("#75715e"), hex("#ae81ff"), hex("#a6e22e"), hex("#66d9ef")),
	palette("catppuccin", hex("#cba6f7"), hex("#a6e3a1"), hex("#6c7086"), hex("#fab387"), hex("#89b4fa"), hex("#f9e2af")),
	palette("onedark", hex("#c678dd"), hex("#98c379"), hex("#5c6370"), hex("#d19a66"), hex("#61afef"), hex("#e5c07b")),
	palette("solarized", hex("#859900"), hex("#2aa198"), lipgloss.AdaptiveColor{Light: "#93a1a1", Dark: "#586e75"}, hex("#d33682"), hex("#268bd2"), hex("#b58900")),
	palette("github-light", hex("#cf222e"), hex("#0a3069"), hex("#6e7781"), hex("#0550ae"), hex("#8250df"), hex("#953800")),
	{name: "off"}, // no colors, like a plain vim
}

// ColorSchemes are the names of the color schemes, the default first.
func ColorSchemes() []string {
	names := make([]string, len(schemes))
	for i, s := range schemes {
		names[i] = s.name
	}
	return names
}

func findScheme(name string) *scheme {
	for i := range schemes {
		if schemes[i].name == name {
			return &schemes[i]
		}
	}
	return nil
}

// WithColorscheme colors the code with the named scheme (the default when
// the name is unknown or empty).
func (m Model) WithColorscheme(name string) Model {
	if s := findScheme(name); s != nil {
		m.ses.scheme = s
	}
	return m
}

// Colorscheme is the name of the scheme in use.
func (m Model) Colorscheme() string { return m.ses.scheme.name }

// OnColorscheme calls save when :colorscheme picks a scheme, to keep it for
// next time.
func (m Model) OnColorscheme(save func(name string)) Model {
	m.ses.onScheme = save
	return m
}

// colorscheme is :colorscheme [name].
func (m Model) colorscheme(name string) vim.Result {
	if name == "" {
		names := ColorSchemes()
		for i, n := range names {
			if n == m.ses.scheme.name {
				names[i] = "[" + n + "]"
			}
		}
		return vim.Result{Message: "temas: " + strings.Join(names, " ") + " (:colorscheme nome)"}
	}
	s := findScheme(name)
	if s == nil {
		return vim.Result{Message: "E185: não existe o tema " + name + " (:colorscheme lista os temas)", Err: true}
	}
	m.ses.scheme = s
	if m.ses.onScheme != nil {
		m.ses.onScheme(s.name)
	}
	return vim.Result{Message: "tema: " + s.name}
}
