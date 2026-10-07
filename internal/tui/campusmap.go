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

// clusterView is one cluster ready to draw: grid size, column order and who
// sits where (occupants[row][post] = login; nil for an empty cluster).
type clusterView struct {
	cluster, rows, posts int
	reverse              bool
	occupants            map[int]map[int]string
}

// postOrder lists the columns left to right: pN … p1 when mirrored.
func (c clusterView) postOrder() []int {
	order := make([]int, 0, c.posts)
	for i := 1; i <= c.posts; i++ {
		if c.reverse {
			order = append(order, c.posts-i+1)
		} else {
			order = append(order, i)
		}
	}
	return order
}

func (c clusterView) online() int {
	n := 0
	for _, posts := range c.occupants {
		n += len(posts)
	}
	return n
}

// campusClusters groups sessions by cluster. Grid sizes come from layout when
// provided (config.yaml campus_layout), never hiding an occupied seat;
// otherwise clusters 1..max(observed) share the largest row × post seen.
// Sessions on hosts outside the cXrYpZ convention are returned apart.
func campusClusters(locations []models.Location, layout map[int]ClusterGrid) ([]clusterView, []models.Location) {
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

	views := make([]clusterView, 0, maxCluster)
	for cluster := 1; cluster <= maxCluster; cluster++ {
		v := clusterView{cluster: cluster, rows: maxRow, posts: maxPost, reverse: true, occupants: occupants[cluster]}
		if grid, ok := layout[cluster]; ok {
			v.rows, v.posts, v.reverse = grid.Rows, grid.Posts, !grid.NaturalPosts
			for row, occupied := range occupants[cluster] {
				v.rows = max(v.rows, row)
				for post := range occupied {
					v.posts = max(v.posts, post)
				}
			}
		}
		views = append(views, v)
	}
	return views, unmapped
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

	clusters, unmapped := campusClusters(locations, layout)
	sections := []string{styleTitle.Render(fmt.Sprintf("%s — %d online", campusName, len(locations)))}
	for _, c := range clusters {
		sections = append(sections, renderCluster(c))
	}
	if len(unmapped) > 0 {
		sections = append(sections, renderUnmapped(unmapped))
	}
	return strings.Join(sections, "\n\n")
}

// renderCluster draws one cluster as a table with the login at each seat.
func renderCluster(c clusterView) string {
	order := c.postOrder()
	headers := make([]string, 0, len(order)+1)
	headers = append(headers, "")
	for _, post := range order {
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

	for row := 1; row <= c.rows; row++ {
		cells := make([]string, 0, len(order)+1)
		cells = append(cells, fmt.Sprintf("r%d", row))
		for _, post := range order {
			if login, ok := c.occupants[row][post]; ok {
				cells = append(cells, styleGood.Render(login))
			} else {
				cells = append(cells, styleLabel.Render("·"))
			}
		}
		t.Row(cells...)
	}

	header := styleTitle.Render(fmt.Sprintf("Cluster %d", c.cluster)) +
		styleLabel.Render(fmt.Sprintf(" — %d online", c.online()))
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
