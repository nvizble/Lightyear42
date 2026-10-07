package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/auth"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/nvizble/Lightyear42/internal/subjects"
	"github.com/nvizble/Lightyear42/internal/tui"
)

// runApp opens the full-screen app (`lightyear` with no arguments). Each tab
// reuses the same services and renderers as the matching subcommand.
func runApp(ctx context.Context) error {
	examSvc, err := newExamService()
	if err != nil {
		return err
	}

	opts := tui.AppOptions{Exam: examSvc}
	deps, cleanup, err := newDeps(ctx)
	if err != nil {
		// Logged out (or no OAuth app configured): the API tabs explain how to
		// fix it and the offline Exam tab still works.
		opts.Tabs = appTabs(nil)
		opts.Unavailable = err.Error() + "\n\nPrimeiro uso: `lightyear setup` e depois `lightyear login`."
	} else {
		defer cleanup()
		opts.Tabs = appTabs(deps)
	}

	program := tea.NewProgram(tui.NewApp(opts, time.Now()),
		tea.WithAltScreen(), tea.WithMouseAllMotion(), tea.WithContext(ctx))
	_, err = program.Run()
	return err
}

// appTabs builds the API-backed tabs; with nil deps they are listed but
// unavailable.
func appTabs(deps *appDeps) []tui.AppTab {
	home, evals, projects := tui.AppTab{Title: "Início"}, tui.AppTab{Title: "Avaliações"}, tui.AppTab{Title: "Projetos"}
	subjectsTab, campus, slots := tui.AppTab{Title: "Subjects"}, tui.AppTab{Title: "Campus"}, tui.AppTab{Title: "Slots"}
	if deps == nil {
		return []tui.AppTab{home, evals, projects, subjectsTab, campus, slots}
	}

	dashboard := services.NewDashboardService(deps.Users, deps.Campus, newFriendsService(), deps.Slots)
	layout := campusLayout()

	home.Load = func(ctx context.Context) (string, error) {
		snap, err := dashboard.Snapshot(ctx, 0)
		if err != nil {
			return "", err
		}
		return tui.RenderHome(snap, layout, time.Now()), nil
	}
	evals.Load = func(ctx context.Context) (string, error) {
		me, err := deps.Users.Me(ctx)
		if err != nil {
			return "", err
		}
		evaluations, err := deps.Users.UpcomingEvaluations(ctx)
		if err != nil {
			return "", err
		}
		return tui.RenderEvaluationRows(evaluations, me.Login, time.Now(), 0), nil
	}
	projects.Load = func(ctx context.Context) (string, error) {
		mine, err := deps.Users.Projects(ctx, "", false)
		if err != nil {
			return "", err
		}
		return tui.RenderProjects(mine), nil
	}
	subjectsTab.LoadView = func(ctx context.Context) (tui.AppView, error) {
		mine, err := deps.Users.Projects(ctx, "", false)
		if err != nil {
			return tui.AppView{}, err
		}
		catalog := make([]string, 0, len(subjects.Embedded()))
		for slug := range subjects.Embedded() {
			catalog = append(catalog, slug)
		}
		return tui.SubjectsView(mine, catalog), nil
	}
	// Clicking (or picking in the search) downloads the PDF, like
	// `lightyear subject`, and opens it.
	subjectsTab.Activate = func(ctx context.Context, h tui.Hotspot) (string, error) {
		dir, err := subjectsDir()
		if err != nil {
			return "", err
		}
		res, err := deps.Subjects.EnsureSubject(ctx, services.SubjectOptions{Query: h.ID, Dir: dir, Lang: "en"})
		if errors.Is(err, services.ErrSubjectPDFUnknown) {
			return "", fmt.Errorf("sem o PDF de %s no catálogo: rode lightyear subject set-id %s <id>", h.Label, h.ID)
		}
		if err != nil {
			return "", err
		}
		if err := auth.OpenBrowser(res.Path); err != nil {
			return "PDF salvo em " + res.Path, nil
		}
		return "Subject aberto: " + res.Path, nil
	}
	campus.LoadView = func(ctx context.Context) (tui.AppView, error) {
		me, err := deps.Users.Me(ctx)
		if err != nil {
			return tui.AppView{}, err
		}
		id, name, err := primaryCampusID(ctx, deps)
		if err != nil {
			return tui.AppView{}, err
		}
		locations, err := deps.Campus.Online(ctx, id)
		if err != nil {
			return tui.AppView{}, err
		}
		friends, err := newFriendsService().List()
		if err != nil {
			return tui.AppView{}, err
		}
		return tui.CampusSeatsView(name, locations, layout, friends, me.Login), nil
	}
	slots.Load = func(ctx context.Context) (string, error) {
		list, err := deps.Slots.List(ctx)
		if err != nil {
			return "", err
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			tui.RenderSlotsCalendar(list, "", time.Now()), "", tui.RenderSlots(list)), nil
	}
	return []tui.AppTab{home, evals, projects, subjectsTab, campus, slots}
}
