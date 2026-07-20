package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/nvizble/Lightyear42/internal/models"
)

// hostPattern matches standard workstation hosts like "c1r2p3"
// (cluster 1, row 2, post 3), ignoring an optional domain suffix.
var hostPattern = regexp.MustCompile(`(?i)^c(\d+)r(\d+)p(\d+)$`)

// seat is a parsed workstation position.
type seat struct {
	cluster, row, post int
}

// parseHost extracts the seat from a location host, false when the host
// does not follow the cXrYpZ convention.
func parseHost(host string) (seat, bool) {
	host, _, _ = strings.Cut(host, ".")
	m := hostPattern.FindStringSubmatch(host)
	if m == nil {
		return seat{}, false
	}
	cluster, _ := strconv.Atoi(m[1])
	row, _ := strconv.Atoi(m[2])
	post, _ := strconv.Atoi(m[3])
	return seat{cluster: cluster, row: row, post: post}, true
}

// ClusterGrid is the drawn size of one cluster, usually from user config.
// Seats overrides the real capacity for irregular clusters (0 = rows × posts).
// NaturalPosts draws columns p1…pN (left-to-right). By default posts are
// mirrored (pN…p1) to match physical numbering on 42 campuses like São Paulo.
type ClusterGrid struct {
	Rows         int
	Posts        int
	Seats        int
	NaturalPosts bool
}

// Capacity returns the number of real seats in the cluster.
func (g ClusterGrid) Capacity() int {
	if g.Seats > 0 {
		return g.Seats
	}
	return g.Rows * g.Posts
}

// RenderCampusMap renders active sessions grouped by cluster as seat maps.
//
// The API only exposes active sessions — there is no public endpoint with
// the physical campus layout. Grid sizes come from layout when provided
// (config.yaml campus_layout); otherwise clusters 1..max(observed) are drawn
// as a uniform grid of the largest row × post seen across the campus.
func RenderCampusMap(campusName string, locations []models.Location, layout map[int]ClusterGrid) string {
	if len(locations) == 0 {
		return styleLabel.Render("Ninguém online no campus agora.")
	}

	// occupants[cluster][row][post] = login
	occupants := map[int]map[int]map[int]string{}
	var unmapped []models.Location
	maxCluster, maxRow, maxPost := 0, 0, 0

	for _, loc := range locations {
		st, ok := parseHost(loc.Host)
		if !ok {
			unmapped = append(unmapped, loc)
			continue
		}
		if occupants[st.cluster] == nil {
			occupants[st.cluster] = map[int]map[int]string{}
		}
		if occupants[st.cluster][st.row] == nil {
			occupants[st.cluster][st.row] = map[int]string{}
		}
		occupants[st.cluster][st.row][st.post] = loc.User.Login

		maxCluster = max(maxCluster, st.cluster)
		maxRow = max(maxRow, st.row)
		maxPost = max(maxPost, st.post)
	}

	for cluster := range layout {
		maxCluster = max(maxCluster, cluster)
	}

	var sections []string
	title := fmt.Sprintf("%s — %d online", campusName, len(locations))
	sections = append(sections, styleTitle.Render(title))

	for cluster := 1; cluster <= maxCluster; cluster++ {
		rows, posts := maxRow, maxPost
		reverse := true // default: physical mirror (pN … p1)
		if grid, ok := layout[cluster]; ok {
			// Never hide an occupied seat that falls outside the configured grid.
			rows, posts = grid.Rows, grid.Posts
			reverse = !grid.NaturalPosts
			for row, occupied := range occupants[cluster] {
				rows = max(rows, row)
				for post := range occupied {
					posts = max(posts, post)
				}
			}
		}
		sections = append(sections, renderCluster(cluster, occupants[cluster], rows, posts, reverse))
	}

	if len(unmapped) > 0 {
		sections = append(sections, renderUnmapped(unmapped))
	}

	return strings.Join(sections, "\n\n")
}

// renderCluster draws one cluster as a uniform rows × posts grid.
// rows may be nil for a cluster with nobody online.
// When reversePosts is true, columns are drawn pN … p1 (physical mirror).
func renderCluster(cluster int, rows map[int]map[int]string, maxRow, maxPost int, reversePosts bool) string {
	online := 0
	for _, posts := range rows {
		online += len(posts)
	}

	postOrder := make([]int, 0, maxPost)
	if reversePosts {
		for post := maxPost; post >= 1; post-- {
			postOrder = append(postOrder, post)
		}
	} else {
		for post := 1; post <= maxPost; post++ {
			postOrder = append(postOrder, post)
		}
	}

	headers := make([]string, 0, maxPost+1)
	headers = append(headers, "")
	for _, post := range postOrder {
		headers = append(headers, fmt.Sprintf("p%d", post))
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorMuted)).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow || col == 0 {
				return styleTableHeader.Padding(0, 1)
			}
			return styleTableCell
		})

	for row := 1; row <= maxRow; row++ {
		cells := make([]string, 0, maxPost+1)
		cells = append(cells, fmt.Sprintf("r%d", row))
		for _, post := range postOrder {
			if login, ok := rows[row][post]; ok {
				cells = append(cells, styleGood.Render(login))
			} else {
				cells = append(cells, styleLabel.Render("·"))
			}
		}
		t.Row(cells...)
	}

	header := styleTitle.Render(fmt.Sprintf("Cluster %d", cluster)) +
		styleLabel.Render(fmt.Sprintf(" — %d online", online))
	return header + "\n" + t.Render()
}

// renderUnmapped lists sessions whose host doesn't follow the seat convention.
func renderUnmapped(locations []models.Location) string {
	var b strings.Builder
	b.WriteString(styleLabel.Render("Outros postos:"))
	for _, loc := range locations {
		b.WriteString("\n  ")
		b.WriteString(styleGood.Render(loc.User.Login))
		b.WriteString(styleLabel.Render(" @ " + loc.Host))
	}
	return b.String()
}
