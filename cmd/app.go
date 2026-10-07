package cmd

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvizble/Lightyear42/internal/services"
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
		tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithContext(ctx))
	_, err = program.Run()
	return err
}

// appTabs builds the API-backed tabs; with nil deps they are listed but
// unavailable.
func appTabs(deps *appDeps) []tui.AppTab {
	tabs := []tui.AppTab{
		{Title: "Início"}, {Title: "Avaliações"}, {Title: "Projetos"}, {Title: "Campus"}, {Title: "Slots"},
	}
	if deps == nil {
		return tabs
	}

	dashboard := services.NewDashboardService(deps.Users, deps.Campus, newFriendsService(), deps.Slots)
	layout := campusLayout()

	tabs[0].Load = func(ctx context.Context) (string, error) {
		snap, err := dashboard.Snapshot(ctx, 0)
		if err != nil {
			return "", err
		}
		return tui.RenderHome(snap, layout, time.Now()), nil
	}
	tabs[1].Load = func(ctx context.Context) (string, error) {
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
	tabs[2].Load = func(ctx context.Context) (string, error) {
		projects, err := deps.Users.Projects(ctx, "", false)
		if err != nil {
			return "", err
		}
		return tui.RenderProjects(projects), nil
	}
	tabs[3].Load = func(ctx context.Context) (string, error) {
		me, err := deps.Users.Me(ctx)
		if err != nil {
			return "", err
		}
		id, name, err := primaryCampusID(ctx, deps)
		if err != nil {
			return "", err
		}
		locations, err := deps.Campus.Online(ctx, id)
		if err != nil {
			return "", err
		}
		friends, err := newFriendsService().List()
		if err != nil {
			return "", err
		}
		return tui.RenderCampusSeats(name, locations, layout, friends, me.Login), nil
	}
	tabs[4].Load = func(ctx context.Context) (string, error) {
		slots, err := deps.Slots.List(ctx)
		if err != nil {
			return "", err
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			tui.RenderSlotsCalendar(slots, "", time.Now()), "", tui.RenderSlots(slots)), nil
	}
	return tabs
}
