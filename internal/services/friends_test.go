package services

import (
	"errors"
	"testing"

	"github.com/joaodiniz/42cli/internal/models"
)

type fakeFriendsStore struct {
	friends []string
}

func (f *fakeFriendsStore) Load() ([]string, error) {
	return append([]string(nil), f.friends...), nil
}

func (f *fakeFriendsStore) Save(friends []string) error {
	f.friends = friends
	return nil
}

func TestFriendsService_AddRemoveList(t *testing.T) {
	t.Parallel()

	store := &fakeFriendsStore{}
	svc := NewFriendsService(store)

	added, err := svc.Add("  Malima-M ")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !added {
		t.Fatal("Add = false, want true")
	}

	// Duplicate (normalized) must not be added twice.
	added, err = svc.Add("malima-m")
	if err != nil {
		t.Fatalf("Add (dup): %v", err)
	}
	if added {
		t.Fatal("Add duplicate = true, want false")
	}

	if _, err := svc.Add("jdiniz"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	friends, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(friends) != 2 || friends[0] != "jdiniz" || friends[1] != "malima-m" {
		t.Fatalf("List = %v, want sorted [jdiniz malima-m]", friends)
	}

	removed, err := svc.Remove("JDINIZ")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if !removed {
		t.Fatal("Remove = false, want true")
	}

	removed, err = svc.Remove("ghost")
	if err != nil {
		t.Fatalf("Remove (absent): %v", err)
	}
	if removed {
		t.Fatal("Remove absent = true, want false")
	}
}

func TestFriendsService_AddEmpty(t *testing.T) {
	t.Parallel()

	svc := NewFriendsService(&fakeFriendsStore{})
	if _, err := svc.Add("   "); !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("err = %v, want ErrEmptyQuery", err)
	}
}

func TestFilterLocationsByLogin(t *testing.T) {
	t.Parallel()

	locations := []models.Location{
		{Host: "c1r1p1", User: models.UserSummary{Login: "jdiniz"}},
		{Host: "c1r1p2", User: models.UserSummary{Login: "stranger"}},
		{Host: "c2r3p4", User: models.UserSummary{Login: "malima-m"}},
	}

	filtered := FilterLocationsByLogin(locations, []string{"jdiniz", "malima-m"})
	if len(filtered) != 2 {
		t.Fatalf("len = %d, want 2", len(filtered))
	}
	if filtered[0].User.Login != "jdiniz" || filtered[1].User.Login != "malima-m" {
		t.Errorf("filtered = %v", filtered)
	}

	if got := FilterLocationsByLogin(locations, nil); len(got) != 0 {
		t.Errorf("empty friends should filter everything, got %v", got)
	}
}
