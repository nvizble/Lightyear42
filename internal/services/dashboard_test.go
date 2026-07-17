package services

import (
	"context"
	"errors"
	"testing"

	"github.com/joaodiniz/42cli/internal/models"
)

type stubMeReader struct {
	user *models.User
	err  error
}

func (s stubMeReader) Me(context.Context) (*models.User, error) { return s.user, s.err }

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
		stubMeReader{user: dashboardUser()},
		online,
		stubFriendsLister{friends: []string{"malima-m"}},
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
	if snap.TakenAt.IsZero() {
		t.Error("TakenAt não deveria ser zero")
	}
}

func TestDashboardSnapshot_ExplicitCampusID(t *testing.T) {
	t.Parallel()

	online := &stubOnlineLister{}
	svc := NewDashboardService(stubMeReader{user: dashboardUser()}, online, stubFriendsLister{})

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

	svc := NewDashboardService(stubMeReader{user: &models.User{Login: "x"}}, &stubOnlineLister{}, stubFriendsLister{})
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
		{"me", NewDashboardService(stubMeReader{err: boom}, &stubOnlineLister{}, stubFriendsLister{})},
		{"online", NewDashboardService(stubMeReader{user: dashboardUser()}, &stubOnlineLister{err: boom}, stubFriendsLister{})},
		{"friends", NewDashboardService(stubMeReader{user: dashboardUser()}, &stubOnlineLister{}, stubFriendsLister{err: boom})},
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
