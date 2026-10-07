package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
)

// fakeUsers implements repository.Users for tests.
type fakeUsers struct {
	me             *models.User
	byLogin        map[string]*models.User
	summaries      []models.UserSummary
	evaluations    []models.ScaleTeam
	asCorrector    []models.ScaleTeam
	asCorrectorErr error
	lastPrefix     string
	lastLimit      int
}

func (f *fakeUsers) Me(context.Context) (*models.User, error) {
	return f.me, nil
}

func (f *fakeUsers) ByLogin(_ context.Context, login string) (*models.User, error) {
	user, ok := f.byLogin[login]
	if !ok {
		return nil, errors.New("not found: " + login)
	}
	return user, nil
}

func (f *fakeUsers) SearchByLoginPrefix(_ context.Context, prefix string, limit int) ([]models.UserSummary, error) {
	f.lastPrefix = prefix
	f.lastLimit = limit
	return f.summaries, nil
}

func (f *fakeUsers) UpcomingEvaluations(context.Context) ([]models.ScaleTeam, error) {
	return f.evaluations, nil
}

func (f *fakeUsers) UnfilledAsCorrector(context.Context) ([]models.ScaleTeam, error) {
	if f.asCorrectorErr != nil {
		return nil, f.asCorrectorErr
	}
	return f.asCorrector, nil
}

func TestUserService_Profile(t *testing.T) {
	t.Parallel()

	repo := &fakeUsers{byLogin: map[string]*models.User{
		"jdiniz": {Login: "jdiniz"},
	}}
	svc := NewUserService(repo)

	tests := []struct {
		name    string
		login   string
		wantErr error
	}{
		{"found", "jdiniz", nil},
		{"normalizes case and spaces", "  JDiniz ", nil},
		{"empty login", "   ", ErrEmptyQuery},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			user, err := svc.Profile(context.Background(), tt.login)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Profile: %v", err)
			}
			if user.Login != "jdiniz" {
				t.Errorf("Login = %q, want jdiniz", user.Login)
			}
		})
	}
}

func TestUserService_Projects(t *testing.T) {
	t.Parallel()

	yes := true
	now := time.Now()
	older := now.Add(-30 * 24 * time.Hour)

	me := &models.User{
		Login: "jdiniz",
		CursusUsers: []models.CursusUser{
			{Level: 5, Cursus: models.Cursus{ID: 21, Name: "42cursus", Kind: "main"}},
			{Level: 8, Cursus: models.Cursus{ID: 9, Name: "C Piscine", Kind: "piscine"}},
		},
		ProjectsUsers: []models.ProjectUser{
			{
				Status: models.ProjectStatusFinished, Validated: &yes, MarkedAt: &older,
				CursusIDs: []int{21}, Project: models.Project{Name: "libft"},
			},
			{
				Status:    models.ProjectStatusInProgress,
				CursusIDs: []int{21}, Project: models.Project{Name: "get_next_line"},
			},
			{
				Status: models.ProjectStatusFinished, Validated: &yes, MarkedAt: &now,
				CursusIDs: []int{9}, Project: models.Project{Name: "Shell 00"},
			},
		},
	}

	svc := NewUserService(&fakeUsers{me: me})

	t.Run("default filters to main cursus, in-progress first", func(t *testing.T) {
		t.Parallel()

		projects, err := svc.Projects(context.Background(), "", false)
		if err != nil {
			t.Fatalf("Projects: %v", err)
		}
		if len(projects) != 2 {
			t.Fatalf("len = %d, want 2 (piscine filtered out)", len(projects))
		}
		if projects[0].Project.Name != "get_next_line" {
			t.Errorf("first = %q, want in-progress get_next_line", projects[0].Project.Name)
		}
	})

	t.Run("all includes piscine", func(t *testing.T) {
		t.Parallel()

		projects, err := svc.Projects(context.Background(), "", true)
		if err != nil {
			t.Fatalf("Projects: %v", err)
		}
		if len(projects) != 3 {
			t.Fatalf("len = %d, want 3", len(projects))
		}
	})
}

func TestUserService_Search(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		term       string
		limit      int
		wantErr    bool
		wantPrefix string
		wantLimit  int
	}{
		{"default limit", "jdi", 0, false, "jdi", 10},
		{"explicit limit", "JDI ", 25, false, "jdi", 25},
		{"empty term", "  ", 10, true, "", 0},
		{"limit above max", "jdi", 101, true, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeUsers{summaries: []models.UserSummary{{Login: "jdiniz"}}}
			svc := NewUserService(repo)

			_, err := svc.Search(context.Background(), tt.term, tt.limit)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Search: %v", err)
			}
			if repo.lastPrefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", repo.lastPrefix, tt.wantPrefix)
			}
			if repo.lastLimit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", repo.lastLimit, tt.wantLimit)
			}
		})
	}
}

func TestUserService_OpenableEvaluation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC)
	past := now.Add(-30 * time.Minute)
	future := now.Add(30 * time.Minute)

	t.Run("prioriza a que já começou", func(t *testing.T) {
		t.Parallel()
		svc := NewUserService(&fakeUsers{asCorrector: []models.ScaleTeam{
			{ID: 1, BeginAt: &future, Team: models.EvaluationTeam{Name: "soon"}},
			{ID: 2, BeginAt: &past, Team: models.EvaluationTeam{Name: "now"}},
		}})
		got, err := svc.OpenableEvaluation(context.Background(), now)
		if err != nil {
			t.Fatalf("OpenableEvaluation: %v", err)
		}
		if got.ID != 2 {
			t.Fatalf("ID = %d, want 2 (já começou)", got.ID)
		}
	})

	t.Run("sem iniciada, pega a próxima", func(t *testing.T) {
		t.Parallel()
		svc := NewUserService(&fakeUsers{asCorrector: []models.ScaleTeam{
			{ID: 3, BeginAt: &future, Team: models.EvaluationTeam{Name: "next"}},
		}})
		got, err := svc.OpenableEvaluation(context.Background(), now)
		if err != nil {
			t.Fatalf("OpenableEvaluation: %v", err)
		}
		if got.ID != 3 {
			t.Fatalf("ID = %d, want 3", got.ID)
		}
		if got.HasStarted(now) {
			t.Fatal("próxima ainda não deveria ter começado")
		}
	})

	t.Run("vazio", func(t *testing.T) {
		t.Parallel()
		svc := NewUserService(&fakeUsers{})
		_, err := svc.OpenableEvaluation(context.Background(), now)
		if !errors.Is(err, ErrNoOpenableEvaluation) {
			t.Fatalf("err = %v, want ErrNoOpenableEvaluation", err)
		}
	})
}
