package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestAppTabBarShowsUpdate(t *testing.T) {
	for _, tt := range []struct {
		width   int
		update  string
		want    string
		without string
	}{
		{120, "1.3.0-canary.36", "↑ 1.3.0-canary.36 disponível: lightyear update", ""},
		{120, "", "", "disponível"},
		{30, "1.3.0-canary.36", "", "disponível"}, // no room: nothing breaks the tabs
	} {
		m := NewApp(AppOptions{Exam: &fakeExam{}, Unavailable: "faça login", UpdateAvailable: tt.update}, appNow)
		next, _ := m.Update(tea.WindowSizeMsg{Width: tt.width, Height: 30})
		bar := ansi.Strip(next.(AppModel).tabBar())
		if tt.want != "" && !strings.Contains(bar, tt.want) {
			t.Errorf("largura %d: a barra %q não mostra %q", tt.width, bar, tt.want)
		}
		if tt.without != "" && strings.Contains(bar, tt.without) {
			t.Errorf("largura %d: a barra %q não deveria mostrar %q", tt.width, bar, tt.without)
		}
		if ansi.StringWidth(bar) > max(tt.width, ansi.StringWidth(ansi.Strip(next.(AppModel).tabBarTabsOnly()))) {
			t.Errorf("largura %d: a barra passou da tela: %q", tt.width, bar)
		}
	}
}

func TestRenderUpdateNotice(t *testing.T) {
	got := ansi.Strip(RenderUpdateNotice("1.3.0-canary.35", "1.3.0-canary.36"))
	for _, want := range []string{"Nova versão do lightyear disponível", "1.3.0-canary.35 → 1.3.0-canary.36", "lightyear update"} {
		if !strings.Contains(got, want) {
			t.Errorf("o aviso %q não tem %q", got, want)
		}
	}
}
