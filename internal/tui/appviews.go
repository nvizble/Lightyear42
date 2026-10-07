package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/exam"
	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/services"
)

// Views of the full-screen app. The CLI commands keep their detailed
// renderers (tables with logins, fill URLs); the app favors a compact,
// glanceable layout.

// homeBarWidth is the width of the occupancy bars on the app's home tab.
const homeBarWidth = 36

// seatGlyph draws one workstation in the campus grid.
const seatGlyph = "██"

var (
	colorSeatEmpty = lipgloss.AdaptiveColor{Light: "252", Dark: "237"}
	styleSeatEmpty = lipgloss.NewStyle().Foreground(colorSeatEmpty)
	styleSeatOn    = lipgloss.NewStyle().Foreground(colorGood)
	styleSeatFr    = lipgloss.NewStyle().Foreground(colorAccent)
	styleSeatMe    = lipgloss.NewStyle().Foreground(colorPrimary)
	styleBold      = lipgloss.NewStyle().Bold(true)
)

// RenderHome renders the app's home tab: who you are, cluster occupancy,
// your next evaluations and which friends are online.
func RenderHome(snap *services.DashboardSnapshot, layout map[int]ClusterGrid, now time.Time) string {
	sections := []string{profileCard(snap)}

	occupancy := styleTitle.Render("Ocupação por cluster") +
		styleLabel.Render("  ·  atualizado às "+snap.TakenAt.Local().Format("15:04"))
	if lines := occupancyLines(snap.Locations, layout, homeBarWidth); len(lines) > 0 {
		occupancy += "\n" + strings.Join(lines, "\n")
	} else {
		occupancy += "\n" + styleLabel.Render("Nenhum posto mapeado em clusters.")
	}
	sections = append(sections, occupancy)

	sections = append(sections, RenderEvaluationRows(snap.Evaluations, snap.Me.Login, now, 3))

	friends := styleTitle.Render("Amigos online")
	switch {
	case len(snap.Friends) == 0:
		friends += "\n" + styleLabel.Render("Sua lista está vazia. Adicione com `lightyear friends add <login>`.")
	case len(snap.FriendsOnline) == 0:
		friends += "\n" + styleLabel.Render(fmt.Sprintf("Nenhum dos seus %d amigos está online.", len(snap.Friends)))
	default:
		friends += "\n" + friendSeats(snap.FriendsOnline)
	}
	sections = append(sections, friends)

	return strings.Join(sections, "\n\n")
}

// profileCard is the home header: campus, people online and your stats.
func profileCard(snap *services.DashboardSnapshot) string {
	me := snap.Me
	line := styleValue.Render(me.Login)
	if cursus := me.MainCursus(); cursus != nil {
		line += styleLevel.Render(fmt.Sprintf("  Level %.2f", cursus.Level))
	}
	line += styleLabel.Render(fmt.Sprintf("  ·  %d ₳  ·  %d pontos de avaliação", me.Wallet, me.CorrectionPoint))
	title := styleTitle.Render(snap.CampusName) + styleLabel.Render(fmt.Sprintf(" — %d online", len(snap.Locations)))
	return styleCard.Render(title + "\n" + line)
}

// friendSeats lists friends online with their seats: "tlima c1r2p3 · …".
func friendSeats(online []models.Location) string {
	parts := make([]string, 0, len(online))
	for _, loc := range online {
		parts = append(parts, styleSeatFr.Render(loc.User.Login)+" "+styleLabel.Render(loc.Host))
	}
	return strings.Join(parts, styleLabel.Render("  ·  "))
}

// RenderEvaluationRows lists upcoming evaluations one per line, in aligned
// columns: "hoje 14:30   você avalia   ft_printf   em 1h18   com bcosta".
// limit <= 0 shows every entry.
func RenderEvaluationRows(evaluations []models.ScaleTeam, meLogin string, now time.Time, limit int) string {
	title := styleTitle.Render("Próximas avaliações")
	if len(evaluations) == 0 {
		return title + "\n" + styleLabel.Render("Nenhuma avaliação agendada.")
	}

	shown := evaluations
	if limit > 0 && len(shown) > limit {
		shown = shown[:limit]
	}

	// Cells are plain text padded to the column width, then styled.
	type row struct{ when, role, project, eta, who string }
	rows := make([]row, 0, len(shown))
	width := row{}
	widen := func(w *string, s string) {
		if len([]rune(s)) > len([]rune(*w)) {
			*w = s
		}
	}
	for _, st := range shown {
		r := row{when: evaluationDay(st.BeginAt, now), project: st.Team.Name, eta: evaluationETA(st.BeginAt, now)}
		if st.IsCorrector(meLogin) {
			r.role = "você avalia"
			if logins := actorLogins(st.Correcteds); logins != "" {
				r.who = "com " + logins
			}
		} else {
			r.role = "você é avaliado"
			if st.Corrector.Login != "" {
				r.who = "por " + st.Corrector.Login
			}
		}
		rows = append(rows, r)
		widen(&width.when, r.when)
		widen(&width.role, r.role)
		widen(&width.project, r.project)
		widen(&width.eta, r.eta)
	}

	pad := func(s, w string) string { return s + strings.Repeat(" ", len([]rune(w))-len([]rune(s))) }
	var b strings.Builder
	b.WriteString(title)
	for _, r := range rows {
		role := styleSeatFr.Render(pad(r.role, width.role))
		if r.role != "você avalia" {
			role = styleGood.Render(pad(r.role, width.role))
		}
		b.WriteString("\n")
		b.WriteString(styleValue.Render(pad(r.when, width.when)) + "   " + role + "   " +
			styleValue.Render(pad(r.project, width.project)) + "   " + styleLabel.Render(pad(r.eta, width.eta)))
		if r.who != "" {
			b.WriteString("   " + styleLabel.Render(r.who))
		}
	}
	if hidden := len(evaluations) - len(shown); hidden > 0 {
		b.WriteString("\n" + styleLabel.Render(fmt.Sprintf("… e mais %d", hidden)))
	}
	return b.String()
}

// evaluationDay says when, relative to now: "hoje 14:30", "amanhã 10:15",
// "qui 09/10 09:00" within a week, "21/10 09:00" beyond.
func evaluationDay(beginAt *time.Time, now time.Time) string {
	if beginAt == nil {
		return "a confirmar"
	}
	t, n := beginAt.Local(), now.Local()
	day := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, x.Location()) }
	days := int(day(t).Sub(day(n)).Hours() / 24)
	switch {
	case days == 0:
		return "hoje " + t.Format("15:04")
	case days == 1:
		return "amanhã " + t.Format("15:04")
	case days > 1 && days < 7:
		return weekdays[t.Weekday()] + " " + t.Format("02/01 15:04")
	}
	return t.Format("02/01 15:04")
}

var weekdays = [...]string{"dom", "seg", "ter", "qua", "qui", "sex", "sáb"}

// evaluationETA is the countdown: "em 25min", "em 1h18", "em 21h", "em 3d".
func evaluationETA(beginAt *time.Time, now time.Time) string {
	if beginAt == nil {
		return ""
	}
	d := beginAt.Sub(now)
	switch {
	case d < 0:
		return "em andamento"
	case d < time.Hour:
		return fmt.Sprintf("em %dmin", int(d.Minutes()))
	case d < 10*time.Hour:
		return fmt.Sprintf("em %dh%02d", int(d.Hours()), int(d.Minutes())%60)
	case d < 48*time.Hour:
		return fmt.Sprintf("em %dh", int(d.Hours()))
	}
	return fmt.Sprintf("em %dd", int(d.Hours()/24))
}

func actorLogins(actors models.ScaleTeamActors) string {
	logins := make([]string, 0, len(actors))
	for _, a := range actors {
		if a.Login != "" {
			logins = append(logins, a.Login)
		}
	}
	return strings.Join(logins, ", ")
}

// Hotspot is a region of a tab's content that reacts to the mouse: hovering
// or clicking it highlights the region and shows Info in the footer.
// Line and Col are 0-based display cells within the content.
type Hotspot struct {
	Line, Col, Width int
	Info             string
}

// AppView is a tab's rendered content plus its interactive regions.
type AppView struct {
	Content  string
	Hotspots []Hotspot
}

// RenderCampusSeats draws each cluster as a compact grid of seats: online,
// friends and you in different colors, empty seats dimmed. Friends online
// are listed with their seats right under the header.
func RenderCampusSeats(campusName string, locations []models.Location, layout map[int]ClusterGrid, friends []string, me string) string {
	return CampusSeatsView(campusName, locations, layout, friends, me).Content
}

// CampusSeatsView is RenderCampusSeats plus one hotspot per occupied seat,
// telling who sits there.
func CampusSeatsView(campusName string, locations []models.Location, layout map[int]ClusterGrid, friends []string, me string) AppView {
	if len(locations) == 0 {
		return AppView{Content: styleLabel.Render("Ninguém online no campus agora.")}
	}

	isFriend := make(map[string]bool, len(friends))
	for _, f := range friends {
		isFriend[strings.ToLower(f)] = true
	}
	since := make(map[string]*time.Time, len(locations))
	for _, loc := range locations {
		if st, ok := parseHost(loc.Host); ok {
			since[seatHost(st.cluster, st.row, st.post)] = loc.BeginAt
		}
	}

	lines := []string{
		styleTitle.Render(campusName) + styleLabel.Render(fmt.Sprintf(" — %d online", len(locations))) +
			"     " + styleSeatOn.Render(seatGlyph) + styleLabel.Render(" online   ") +
			styleSeatFr.Render(seatGlyph) + styleLabel.Render(" amigos   ") +
			styleSeatMe.Render(seatGlyph) + styleLabel.Render(" você") +
			styleLabel.Render("   ·   passe o mouse num posto para ver quem está lá"),
	}

	var online []models.Location
	for _, loc := range locations {
		if isFriend[strings.ToLower(loc.User.Login)] {
			online = append(online, loc)
		}
	}
	sort.Slice(online, func(i, j int) bool { return online[i].User.Login < online[j].User.Login })
	if len(online) > 0 {
		lines = append(lines, styleLabel.Render("Amigos online  ")+friendSeats(online))
	}

	var spots []Hotspot
	clusters, unmapped := campusClusters(locations, layout)
	for _, c := range clusters {
		lines = append(lines, "", styleTitle.Render(fmt.Sprintf("Cluster %d", c.cluster))+
			styleLabel.Render(fmt.Sprintf(" · %d online", c.online())))
		order := c.postOrder()
		for row := 1; row <= c.rows; row++ {
			var b strings.Builder
			for i, post := range order {
				if i > 0 {
					b.WriteString(" ")
				}
				login, ok := c.occupants[row][post]
				if !ok {
					b.WriteString(styleSeatEmpty.Render(seatGlyph))
					continue
				}
				host, tag := seatHost(c.cluster, row, post), ""
				switch {
				case me != "" && strings.EqualFold(login, me):
					b.WriteString(styleSeatMe.Render(seatGlyph))
					tag = " (você)"
				case isFriend[strings.ToLower(login)]:
					b.WriteString(styleSeatFr.Render(seatGlyph))
					tag = " (amigo)"
				default:
					b.WriteString(styleSeatOn.Render(seatGlyph))
				}
				info := host + " · " + login + tag
				if at := since[host]; at != nil {
					info += " · online desde " + at.Local().Format("15:04")
				}
				spots = append(spots, Hotspot{Line: len(lines), Col: i * (lipgloss.Width(seatGlyph) + 1), Width: lipgloss.Width(seatGlyph), Info: info})
			}
			lines = append(lines, b.String())
		}
	}
	if len(unmapped) > 0 {
		lines = append(lines, "", renderUnmapped(unmapped))
	}
	return AppView{Content: strings.Join(lines, "\n"), Hotspots: spots}
}

func seatHost(cluster, row, post int) string { return fmt.Sprintf("c%dr%dp%d", cluster, row, post) }

// examCard draws the session card. The app shows the essentials; the
// detailed version (CLI) adds attempts and the subject folder. hint, when
// set, closes the card.
func examCard(s exam.Session, now time.Time, detailed bool, hint string) string {
	title := fmt.Sprintf("Exam Rank %s", s.Rank)
	if s.Mode == exam.ModePractice {
		title = "Prática"
	}

	rows := [][2]string{
		{"Exercício", styleLevel.Render(s.Exercise) + styleLabel.Render(fmt.Sprintf("  (nível %d)", s.Level))},
	}
	if s.Mode == exam.ModeExam {
		rows = append(rows,
			[2]string{"Nota", fmt.Sprintf("%d/100", s.Score)},
			[2]string{"Tempo", renderRemaining(s, now)},
		)
	}
	if detailed {
		rows = append(rows,
			[2]string{"Tentativas", fmt.Sprint(s.Attempts)},
			[2]string{"Subject", filepath.Join(s.Workspace, "subjects", s.Exercise) + string(filepath.Separator)},
		)
	}
	rows = append(rows, [2]string{"Entrega", filepath.Join(s.Workspace, "rendu", s.Exercise) + string(filepath.Separator)})

	var b strings.Builder
	b.WriteString(styleTitle.Render(title))
	for _, r := range rows {
		b.WriteString("\n")
		b.WriteString(styleLabel.Render(fmt.Sprintf("%-11s", r[0])))
		b.WriteString(r[1])
	}
	if hint != "" {
		b.WriteString("\n\n" + styleLabel.Render(hint))
	}
	return styleCard.Render(b.String())
}

// RenderGradeResult renders a grading outcome: a big SUCCESS/FAILURE banner,
// what was tested (and why it failed) and, in an exam, the level
// progression and the score. The next exercise's card is not included.
func RenderGradeResult(r services.GradeReport) string {
	var b strings.Builder
	passedTests := 0
	for _, t := range r.Tests {
		if t.Passed {
			passedTests++
		}
	}

	if r.Passed {
		b.WriteString(styleGood.Bold(true).Render(banner("✓", "SUCCESS")))
		detail := r.Graded
		if n := len(r.Tests); n > 0 {
			detail += fmt.Sprintf(" · %d/%d testes", n, n)
		}
		b.WriteString("\n" + styleLabel.Render(detail))
	} else {
		b.WriteString(styleFail.Bold(true).Render(banner("✗", "FAILURE")))
		if len(r.Tests) == 0 {
			b.WriteString("\n" + styleLabel.Render(r.Graded))
		}
		for _, t := range r.Tests {
			mark, style := "✓", styleGood
			if !t.Passed {
				mark, style = "✗", styleFail
			}
			b.WriteString("\n" + style.Render(mark) + " " + styleLabel.Render(strings.TrimSpace(r.Graded+" "+exam.QuoteArgs(t.Args))))
		}
		b.WriteString("\n\n" + r.Trace)
		if r.TracePath != "" {
			b.WriteString("\n\n" + styleLabel.Render("Trace salvo em "+r.TracePath))
		}
	}

	switch {
	case r.Passed && r.Completed && r.Session.Mode == exam.ModePractice:
		b.WriteString("\n\nExercício concluído. Pratique outro com `lightyear exam practice <exercício>`.")
	case len(r.Levels) > 0:
		b.WriteString("\n\n" + levelBadges(r) + "\n" + scoreLine(r))
	}
	return b.String()
}

// levelBadges draws one box per level of the rank, the cleared ones in green.
func levelBadges(r services.GradeReport) string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	badges := make([]string, 0, len(r.Levels)*2)
	for i, level := range r.Levels {
		done := r.Completed || level < r.Session.Level
		label := fmt.Sprintf("nível %d", level)
		style := box.BorderForeground(colorMuted).Foreground(colorMuted)
		if done {
			label += " ✓"
			style = box.BorderForeground(colorGood).Foreground(colorGood)
		}
		if i > 0 {
			badges = append(badges, " ")
		}
		badges = append(badges, style.Render(label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, badges...)
}

func scoreLine(r services.GradeReport) string {
	status := "subindo de nível…"
	switch {
	case r.Completed:
		status = "prova concluída"
	case !r.Passed:
		status = "mesmo exercício; tente de novo"
	}
	return styleBold.Render(fmt.Sprintf("%d/100", r.Session.Score)) + styleLabel.Render("  ·  "+status)
}

// banner writes word in 3-line block letters, with mark on the middle line.
func banner(mark, word string) string {
	var lines [3]strings.Builder
	for i, ch := range word {
		glyph, ok := bannerFont[ch]
		if !ok {
			continue
		}
		for l := range lines {
			if i > 0 {
				lines[l].WriteString(" ")
			}
			lines[l].WriteString(glyph[l])
		}
	}
	pad := strings.Repeat(" ", lipgloss.Width(mark)+1)
	return pad + lines[0].String() + "\n" + mark + " " + lines[1].String() + "\n" + pad + lines[2].String()
}

// bannerFont holds the block letters used by banner, built from 6-row pixel
// maps folded into 3 rows of half blocks (▀ ▄ █).
var bannerFont = func() map[rune][3]string {
	pixels := map[rune][6]string{
		'S': {".###", "#...", ".##.", "...#", "...#", "###."},
		'U': {"#..#", "#..#", "#..#", "#..#", "#..#", ".##."},
		'C': {".###", "#...", "#...", "#...", "#...", ".###"},
		'E': {"####", "#...", "###.", "#...", "#...", "####"},
		'F': {"####", "#...", "###.", "#...", "#...", "#..."},
		'A': {".##.", "#..#", "#..#", "####", "#..#", "#..#"},
		'I': {"###", ".#.", ".#.", ".#.", ".#.", "###"},
		'L': {"#...", "#...", "#...", "#...", "#...", "####"},
		'R': {"###.", "#..#", "###.", "#.#.", "#..#", "#..#"},
	}
	font := make(map[rune][3]string, len(pixels))
	for ch, p := range pixels {
		var g [3]string
		for l := 0; l < 3; l++ {
			var row strings.Builder
			for x := range p[0] {
				top, bottom := p[2*l][x] == '#', p[2*l+1][x] == '#'
				switch {
				case top && bottom:
					row.WriteString("█")
				case top:
					row.WriteString("▀")
				case bottom:
					row.WriteString("▄")
				default:
					row.WriteString(" ")
				}
			}
			g[l] = row.String()
		}
		font[ch] = g
	}
	return font
}()
