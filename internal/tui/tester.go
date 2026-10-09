package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/nvizble/Lightyear42/internal/tester"
)

// RenderTestReport renders `lightyear test --run`: one line per group, the
// failures (with what went wrong) under the groups that failed. all lists
// the passing cases too.
func RenderTestReport(r tester.Report, took time.Duration, all bool) string {
	var b strings.Builder
	for _, g := range r.Groups() {
		switch failed := g.Failed(); {
		case failed > 0:
			fmt.Fprintf(&b, "%s %s\n", styleFail.Render("✗ "+g.Name), styleLabel.Render(fmt.Sprintf("%d de %d falharam", failed, len(g.Cases))))
		case g.Skipped():
			fmt.Fprintf(&b, "%s\n", styleLabel.Render("– "+g.Name+" (pulado)"))
		default:
			fmt.Fprintf(&b, "%s %s\n", styleGood.Render("✓ "+g.Name), styleLabel.Render(fmt.Sprint(len(g.Cases))))
		}
		for _, c := range g.Cases {
			if !all && !c.Status.Failed() && !(c.Status == tester.Skip && !g.Skipped()) {
				continue
			}
			mark := styleGood.Render("✓")
			switch {
			case c.Status == tester.Skip:
				mark = styleLabel.Render("–")
			case c.Status.Failed():
				mark = styleFail.Render("✗")
			}
			fmt.Fprintf(&b, "    %s %s", mark, c.Name)
			if c.Status.Failed() && c.Status != tester.KO {
				b.WriteString(styleFail.Render(" [" + string(c.Status) + "]"))
			}
			b.WriteString("\n")
			if c.Detail != "" && (c.Status.Failed() || c.Status == tester.Skip) {
				for line := range strings.SplitSeq(c.Detail, "\n") {
					fmt.Fprintf(&b, "        %s\n", styleLabel.Render(line))
				}
			}
		}
	}
	passed, failed, skipped := r.Count()
	summary := styleGood.Render(fmt.Sprintf("%d passaram", passed))
	if failed > 0 {
		summary += " · " + styleFail.Render(fmt.Sprintf("%d falharam", failed))
	}
	if skipped > 0 {
		summary += " · " + styleLabel.Render(fmt.Sprintf("%d pulados", skipped))
	}
	fmt.Fprintf(&b, "\n%s %s %s", styleTitle.Render(r.Project+":"), summary, styleLabel.Render(fmt.Sprintf("(%.1fs)", took.Seconds())))
	return b.String()
}
