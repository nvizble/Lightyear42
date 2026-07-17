package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
	"github.com/joaodiniz/42cli/internal/repository"
)

// Search result bounds enforced by the service.
const (
	searchLimitDefault = 10
	searchLimitMax     = 100
)

// ErrEmptyQuery indicates a blank login or search term.
var ErrEmptyQuery = errors.New("informe um login ou termo de busca não vazio")

// UserService implements the user-facing business rules for profiles and search.
type UserService struct {
	users repository.Users
}

// NewUserService wires the users repository.
func NewUserService(users repository.Users) *UserService {
	return &UserService{users: users}
}

// Me returns the authenticated user's profile.
func (s *UserService) Me(ctx context.Context) (*models.User, error) {
	return s.users.Me(ctx)
}

// Profile returns the profile of the given login.
func (s *UserService) Profile(ctx context.Context, login string) (*models.User, error) {
	login = strings.TrimSpace(strings.ToLower(login))
	if login == "" {
		return nil, ErrEmptyQuery
	}
	return s.users.ByLogin(ctx, login)
}

// Projects returns the user's project enrolments, most recent first and
// in-progress work on top. login == "" means the authenticated user.
// By default only projects of the main cursus are returned (piscine days
// would flood the list); set all to include every cursus.
func (s *UserService) Projects(ctx context.Context, login string, all bool) ([]models.ProjectUser, error) {
	var user *models.User
	var err error
	if login == "" {
		user, err = s.users.Me(ctx)
	} else {
		user, err = s.Profile(ctx, login)
	}
	if err != nil {
		return nil, err
	}

	projects := user.ProjectsUsers
	if !all {
		if main := user.MainCursus(); main != nil {
			filtered := make([]models.ProjectUser, 0, len(projects))
			for _, pu := range projects {
				if pu.InCursus(main.Cursus.ID) {
					filtered = append(filtered, pu)
				}
			}
			projects = filtered
		}
	}

	sort.SliceStable(projects, func(i, j int) bool {
		a, b := projects[i], projects[j]
		aActive := a.Status != models.ProjectStatusFinished
		bActive := b.Status != models.ProjectStatusFinished
		if aActive != bActive {
			return aActive
		}
		return markedTime(a).After(markedTime(b))
	})

	return projects, nil
}

// markedTime returns the evaluation time, zero when not yet marked.
func markedTime(pu models.ProjectUser) time.Time {
	if pu.MarkedAt == nil {
		return time.Time{}
	}
	return *pu.MarkedAt
}

// UpcomingEvaluations returns the authenticated user's scheduled evaluations
// (as evaluator or evaluated), soonest first.
func (s *UserService) UpcomingEvaluations(ctx context.Context) ([]models.ScaleTeam, error) {
	return s.users.UpcomingEvaluations(ctx)
}

// Search lists users whose login starts with term, up to limit results.
// A non-positive limit falls back to the default; the maximum is capped
// to respect the API page size.
func (s *UserService) Search(ctx context.Context, term string, limit int) ([]models.UserSummary, error) {
	term = strings.TrimSpace(strings.ToLower(term))
	if term == "" {
		return nil, ErrEmptyQuery
	}

	if limit <= 0 {
		limit = searchLimitDefault
	}
	if limit > searchLimitMax {
		return nil, fmt.Errorf("limite máximo de resultados é %d", searchLimitMax)
	}

	return s.users.SearchByLoginPrefix(ctx, term, limit)
}
