package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joaodiniz/42cli/internal/models"
	"github.com/joaodiniz/42cli/internal/services"
)

// snapshotTimeout bounds each dashboard refresh round-trip.
const snapshotTimeout = 30 * time.Second

// occupancyBarWidth is the width of each cluster occupancy bar.
const occupancyBarWidth = 24

// SnapshotProvider fetches a fresh dashboard snapshot.
// Implemented by *services.DashboardService.
type SnapshotProvider interface {
	Snapshot(ctx context.Context, campusID int) (*services.DashboardSnapshot, error)
}

// DashboardOptions configures the live dashboard.
type DashboardOptions struct {
	// CampusID selects the campus; 0 means the user's primary campus.
	CampusID int
	// Interval between automatic refreshes.
	Interval time.Duration
	// Layout is the optional physical cluster layout (config campus_layout).
	Layout map[int]ClusterGrid
}

// snapshotMsg carries the result of one refresh.
type snapshotMsg struct {
	snap *services.DashboardSnapshot
	err  error
}

// tickMsg fires when it is time to refresh automatically.
type tickMsg time.Time

// DashboardModel is the Bubble Tea model of the live dashboard.
type DashboardModel struct {
	provider SnapshotProvider
	opts     DashboardOptions

	snap    *services.DashboardSnapshot
	err     error
	loading bool
	width   int
}

// NewDashboard builds the dashboard model.
func NewDashboard(provider SnapshotProvider, opts DashboardOptions) DashboardModel {
	if opts.Interval <= 0 {
		opts.Interval = time.Minute
	}
	return DashboardModel{provider: provider, opts: opts, loading: true}
}

// Init starts the first fetch and the refresh timer.
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(m.fetch(), m.tick())
}

// Update handles keys, ticks and fetch results.
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "r":
			if m.loading {
				return m, nil
			}
			m.loading = true
			return m, m.fetch()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tickMsg:
		cmds := []tea.Cmd{m.tick()}
		if !m.loading {
			m.loading = true
			cmds = append(cmds, m.fetch())
		}
		return m, tea.Batch(cmds...)

	case snapshotMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.snap = msg.snap
		}
	}
	return m, nil
}

// View renders the dashboard.
func (m DashboardModel) View() string {
	if m.snap == nil {
		if m.err != nil {
			return styleFail.Render("Erro ao carregar o dashboard: "+m.err.Error()) +
				"\n" + m.footer()
		}
		return styleLabel.Render("Carregando dashboard…")
	}

	sections := []string{
		m.header(),
		renderOccupancy(m.snap.Locations, m.opts.Layout),
		m.friendsSection(),
	}
	if m.err != nil {
		sections = append(sections, styleFail.Render("Falha ao atualizar: "+m.err.Error()+" (mostrando dados anteriores)"))
	}
	sections = append(sections, m.footer())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// fetch runs one snapshot refresh off the UI loop.
func (m DashboardModel) fetch() tea.Cmd {
	provider, campusID := m.provider, m.opts.CampusID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), snapshotTimeout)
		defer cancel()
		snap, err := provider.Snapshot(ctx, campusID)
		return snapshotMsg{snap: snap, err: err}
	}
}

// tick schedules the next automatic refresh.
func (m DashboardModel) tick() tea.Cmd {
	return tea.Tick(m.opts.Interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// header shows who you are, the campus and when data was last refreshed.
func (m DashboardModel) header() string {
	snap := m.snap
	var b strings.Builder

	b.WriteString(styleTitle.Render(snap.CampusName))
	b.WriteString(styleLabel.Render(fmt.Sprintf(" — %d online", len(snap.Locations))))
	b.WriteString("\n")

	b.WriteString(styleValue.Render(snap.Me.Login))
	if cursus := snap.Me.MainCursus(); cursus != nil {
		b.WriteString(styleLevel.Render(fmt.Sprintf("  Level %.2f", cursus.Level)))
	}
	b.WriteString(styleLabel.Render(fmt.Sprintf("  ·  %d ₳  ·  %d pontos de avaliação",
		snap.Me.Wallet, snap.Me.CorrectionPoint)))
	b.WriteString("\n")

	status := "atualizado às " + snap.TakenAt.Local().Format("15:04:05")
	if m.loading {
		status += "  ·  atualizando…"
	}
	b.WriteString(styleLabel.Render(status))

	return styleCard.Render(b.String())
}

// friendsSection renders friends online, or a short hint when the list is empty.
func (m DashboardModel) friendsSection() string {
	if len(m.snap.Friends) == 0 {
		return styleLabel.Render("Sem amigos na lista. Adicione com `42 friends add <login>`.")
	}
	return RenderFriendsOnline(m.snap.FriendsOnline, len(m.snap.Friends))
}

// footer lists the key bindings.
func (m DashboardModel) footer() string {
	return styleLabel.Render("r atualizar  ·  q sair")
}

// renderOccupancy draws one occupancy bar per cluster. Capacity comes from
// layout when configured; otherwise only the online count is shown.
func renderOccupancy(locations []models.Location, layout map[int]ClusterGrid) string {
	online := map[int]int{}
	maxCluster := 0
	for _, loc := range locations {
		st, ok := parseHost(loc.Host)
		if !ok {
			continue
		}
		online[st.cluster]++
		maxCluster = max(maxCluster, st.cluster)
	}
	for cluster := range layout {
		maxCluster = max(maxCluster, cluster)
	}

	if maxCluster == 0 {
		return styleCard.Render(styleLabel.Render("Nenhum posto mapeado em clusters."))
	}

	clusters := make([]int, 0, maxCluster)
	for cluster := 1; cluster <= maxCluster; cluster++ {
		clusters = append(clusters, cluster)
	}
	sort.Ints(clusters)

	var b strings.Builder
	b.WriteString(styleTitle.Render("Ocupação por cluster"))
	for _, cluster := range clusters {
		count := online[cluster]
		capacity := 0
		if grid, ok := layout[cluster]; ok {
			capacity = grid.Capacity()
		}
		b.WriteString("\n")
		b.WriteString(occupancyLine(cluster, count, capacity))
	}
	return styleCard.Render(b.String())
}

// occupancyLine renders "Cluster N  ████░░  57/72  79%".
// capacity == 0 means unknown (no layout configured).
func occupancyLine(cluster, count, capacity int) string {
	label := fmt.Sprintf("Cluster %-2d ", cluster)
	if capacity <= 0 {
		return styleValue.Render(label) + styleLabel.Render(fmt.Sprintf("%d online", count))
	}

	fraction := float64(count) / float64(capacity)
	if fraction > 1 {
		fraction = 1
	}
	filled := int(fraction*float64(occupancyBarWidth) + 0.5)
	bar := styleGood.Render(strings.Repeat("█", filled)) +
		styleLabel.Render(strings.Repeat("░", occupancyBarWidth-filled))

	return styleValue.Render(label) + bar +
		styleValue.Render(fmt.Sprintf(" %3d/%d", count, capacity)) +
		styleLabel.Render(fmt.Sprintf(" %3.0f%%", fraction*100))
}
