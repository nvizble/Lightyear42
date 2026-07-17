package services

import (
	"context"
	"errors"
	"testing"

	"github.com/joaodiniz/42cli/internal/models"
)

type stubMeReader struct {
	user        *models.User
	err         error
	evaluations []models.ScaleTeam
	evalErr     error
}

func (s stubMeReader) Me(context.Context) (*models.User, error) { return s.user, s.err }

func (s stubMeReader) UpcomingEvaluations(context.Context) ([]models.ScaleTeam, error) {
	return s.evaluations, s.evalErr
}

type stubOnlineLister struct {
	locations []models.Location
	err       error
	gotCampus int
}

func (s *stubOnlineLister) Online(_ context.Context, campusID int) ([]models.Location, error) {
	s.gotCampus = campusID
	return s.locations, s.err
}

type stubFriendsLister struct {
	friends []string
	err     error
}

func (s stubFriendsLister) List() ([]string, error) { return s.friends, s.err }

type stubSlotsLister struct {
	slots []models.Slot
	err   error
}

func (s stubSlotsLister) List(context.Context) ([]models.Slot, error) { return s.slots, s.err }

func dashboardUser() *models.User {
	return &models.User{
		Login:       "jdiniz",
		Campus:      []models.Campus{{ID: 28, Name: "São-Paulo"}},
		CampusUsers: []models.CampusUser{{CampusID: 28, IsPrimary: true}},
	}
}

func TestDashboardSnapshot_PrimaryCampusAndFriends(t *testing.T) {
	t.Parallel()

	locations := []models.Location{
		{Host: "c1r1p1", User: models.UserSummary{Login: "malima-m"}},
		{Host: "c1r1p2", User: models.UserSummary{Login: "other"}},
	}
	online := &stubOnlineLister{locations: locations}
	svc := NewDashboardService(
		stubMeReader{
			user:        dashboardUser(),
			evaluations: []models.ScaleTeam{{ID: 7, Team: models.EvaluationTeam{Name: "libft group"}}},
		},
		online,
		stubFriendsLister{friends: []string{"malima-m"}},
		stubSlotsLister{slots: []models.Slot{{ID: 99}}},
	)

	snap, err := svc.Snapshot(context.Background(), 0)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if online.gotCampus != 28 {
		t.Errorf("campus consultado = %d, want 28 (primário)", online.gotCampus)
	}
	if snap.CampusName != "São-Paulo" {
		t.Errorf("CampusName = %q, want São-Paulo", snap.CampusName)
	}
	if len(snap.Locations) != 2 {
		t.Errorf("len(Locations) = %d, want 2", len(snap.Locations))
	}
	if len(snap.FriendsOnline) != 1 || snap.FriendsOnline[0].User.Login != "malima-m" {
		t.Errorf("FriendsOnline = %+v, want só malima-m", snap.FriendsOnline)
	}
	if len(snap.Evaluations) != 1 || snap.Evaluations[0].ID != 7 {
		t.Errorf("Evaluations = %+v, want a avaliação 7", snap.Evaluations)
	}
	if len(snap.Slots) != 1 || snap.Slots[0].ID != 99 {
		t.Errorf("Slots = %+v, want id 99", snap.Slots)
	}
	if snap.TakenAt.IsZero() {
		t.Error("TakenAt não deveria ser zero")
	}
}

func TestDashboardSnapshot_SlotsSoftFail(t *testing.T) {
	t.Parallel()

	svc := NewDashboardService(
		stubMeReader{user: dashboardUser()},
		&stubOnlineLister{},
		stubFriendsLister{},
		stubSlotsLister{err: errors.New("scope projects")},
	)

	snap, err := svc.Snapshot(context.Background(), 0)
	if err != nil {
		t.Fatalf("Snapshot não deveria falhar por slots: %v", err)
	}
	if snap.SlotsErr == "" {
		t.Fatal("SlotsErr deveria estar preenchido")
	}
	if len(snap.Slots) != 0 {
		t.Errorf("Slots = %+v, want vazio", snap.Slots)
	}
}

func TestDashboardSnapshot_ExplicitCampusID(t *testing.T) {
	t.Parallel()

	online := &stubOnlineLister{}
	svc := NewDashboardService(stubMeReader{user: dashboardUser()}, online, stubFriendsLister{}, nil)

	snap, err := svc.Snapshot(context.Background(), 42)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if online.gotCampus != 42 {
		t.Errorf("campus consultado = %d, want 42", online.gotCampus)
	}
	if snap.CampusName != "Campus 42" {
		t.Errorf("CampusName = %q, want Campus 42", snap.CampusName)
	}
}

func TestDashboardSnapshot_NoCampus(t *testing.T) {
	t.Parallel()

	svc := NewDashboardService(stubMeReader{user: &models.User{Login: "x"}}, &stubOnlineLister{}, stubFriendsLister{}, nil)
	if _, err := svc.Snapshot(context.Background(), 0); err == nil {
		t.Fatal("Snapshot deveria falhar sem campus primário")
	}
}

func TestDashboardSnapshot_PropagatesErrors(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")

	tests := []struct {
		name string
		svc  *DashboardService
	}{
		{"me", NewDashboardService(stubMeReader{err: boom}, &stubOnlineLister{}, stubFriendsLister{}, nil)},
		{"online", NewDashboardService(stubMeReader{user: dashboardUser()}, &stubOnlineLister{err: boom}, stubFriendsLister{}, nil)},
		{"friends", NewDashboardService(stubMeReader{user: dashboardUser()}, &stubOnlineLister{}, stubFriendsLister{err: boom}, nil)},
		{"evaluations", NewDashboardService(stubMeReader{user: dashboardUser(), evalErr: boom}, &stubOnlineLister{}, stubFriendsLister{}, nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tt.svc.Snapshot(context.Background(), 0); !errors.Is(err, boom) {
				t.Fatalf("err = %v, want boom", err)
			}
		})
	}
}
