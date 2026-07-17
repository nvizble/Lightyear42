package services

import (
	"context"
	"fmt"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

// selfReader reads the authenticated user's own data (profile and scheduled
// evaluations). Implemented by *UserService.
type selfReader interface {
	Me(ctx context.Context) (*models.User, error)
	UpcomingEvaluations(ctx context.Context) ([]models.ScaleTeam, error)
}

// onlineLister lists active sessions at a campus. Implemented by *CampusService.
type onlineLister interface {
	Online(ctx context.Context, campusID int) ([]models.Location, error)
}

// friendsLister lists the local friends. Implemented by *FriendsService.
type friendsLister interface {
	List() ([]string, error)
}

// slotsLister lists the user's future evaluation slots. Implemented by *SlotsService.
type slotsLister interface {
	List(ctx context.Context) ([]models.Slot, error)
}

// DashboardSnapshot is one refresh of everything the dashboard shows.
type DashboardSnapshot struct {
	Me            *models.User
	CampusID      int
	CampusName    string
	Locations     []models.Location
	Friends       []string
	FriendsOnline []models.Location
	Evaluations   []models.ScaleTeam
	Slots         []models.Slot
	// SlotsErr is set when slots could not be loaded (e.g. missing projects
	// scope). The rest of the dashboard still renders.
	SlotsErr string
	TakenAt  time.Time
}

// DashboardService aggregates profile, campus presence, friends, evaluations
// and slots into a single snapshot for the live dashboard.
type DashboardService struct {
	users   selfReader
	campus  onlineLister
	friends friendsLister
	slots   slotsLister
	now     func() time.Time
}

// NewDashboardService wires the services the dashboard reads from.
// slots may be nil; in that case the slots panel stays empty.
func NewDashboardService(users selfReader, campus onlineLister, friends friendsLister, slots slotsLister) *DashboardService {
	return &DashboardService{users: users, campus: campus, friends: friends, slots: slots, now: time.Now}
}

// Snapshot fetches a fresh view of the dashboard data. campusID == 0 means
// the authenticated user's primary campus.
func (s *DashboardService) Snapshot(ctx context.Context, campusID int) (*DashboardSnapshot, error) {
	me, err := s.users.Me(ctx)
	if err != nil {
		return nil, err
	}

	campusName := fmt.Sprintf("Campus %d", campusID)
	if campusID == 0 {
		primary := me.PrimaryCampus()
		if primary == nil {
			return nil, fmt.Errorf("seu perfil não tem campus associado; use --id")
		}
		campusID, campusName = primary.ID, primary.Name
	} else if primary := me.PrimaryCampus(); primary != nil && primary.ID == campusID {
		campusName = primary.Name
	}

	locations, err := s.campus.Online(ctx, campusID)
	if err != nil {
		return nil, err
	}

	friends, err := s.friends.List()
	if err != nil {
		return nil, err
	}

	evaluations, err := s.users.UpcomingEvaluations(ctx)
	if err != nil {
		return nil, err
	}

	snap := &DashboardSnapshot{
		Me:            me,
		CampusID:      campusID,
		CampusName:    campusName,
		Locations:     locations,
		Friends:       friends,
		FriendsOnline: FilterLocationsByLogin(locations, friends),
		Evaluations:   evaluations,
		TakenAt:       s.now(),
	}

	// Slots need the projects scope; failure must not take down the dashboard.
	if s.slots != nil {
		slots, err := s.slots.List(ctx)
		if err != nil {
			snap.SlotsErr = err.Error()
		} else {
			snap.Slots = slots
		}
	}

	return snap, nil
}
