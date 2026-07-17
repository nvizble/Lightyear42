package models

import (
	"testing"
	"time"
)

func timePtr(t time.Time) *time.Time { return &t }

func TestUser_MainCursus(t *testing.T) {
	t.Parallel()

	past := timePtr(time.Now().Add(-365 * 24 * time.Hour))
	older := timePtr(time.Now().Add(-2 * 365 * 24 * time.Hour))
	ended := timePtr(time.Now().Add(-300 * 24 * time.Hour))

	piscine := CursusUser{
		Level:   8.2, // nível alto: não pode vencer só por isso
		BeginAt: older,
		EndAt:   ended,
		Cursus:  Cursus{Name: "C Piscine", Kind: "piscine"},
	}
	mainCursus := CursusUser{
		Level:   3.1,
		BeginAt: past,
		Cursus:  Cursus{Name: "42cursus", Kind: "main"},
	}
	oldMain := CursusUser{
		Level:   21.0,
		BeginAt: older,
		EndAt:   ended,
		Cursus:  Cursus{Name: "42", Kind: "main"},
	}

	tests := []struct {
		name string
		user User
		want string
	}{
		{
			name: "main cursus wins over higher-level piscine",
			user: User{CursusUsers: []CursusUser{piscine, mainCursus}},
			want: "42cursus",
		},
		{
			name: "order of enrolments does not matter",
			user: User{CursusUsers: []CursusUser{mainCursus, piscine}},
			want: "42cursus",
		},
		{
			name: "active main wins over finished main",
			user: User{CursusUsers: []CursusUser{oldMain, mainCursus}},
			want: "42cursus",
		},
		{
			name: "pisciner falls back to piscine",
			user: User{CursusUsers: []CursusUser{piscine}},
			want: "C Piscine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.user.MainCursus()
			if got == nil {
				t.Fatal("MainCursus = nil")
			}
			if got.Cursus.Name != tt.want {
				t.Errorf("MainCursus = %q, want %q", got.Cursus.Name, tt.want)
			}
		})
	}
}

func TestUser_MainCursus_Empty(t *testing.T) {
	t.Parallel()

	user := User{}
	if got := user.MainCursus(); got != nil {
		t.Fatalf("MainCursus = %v, want nil", got)
	}
}

func TestUser_PrimaryCampus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		user User
		want string
	}{
		{
			name: "is_primary wins over order",
			user: User{
				Campus: []Campus{{ID: 1, Name: "Paris"}, {ID: 30, Name: "São-Paulo"}},
				CampusUsers: []CampusUser{
					{CampusID: 1, IsPrimary: false},
					{CampusID: 30, IsPrimary: true},
				},
			},
			want: "São-Paulo",
		},
		{
			name: "falls back to first campus without campus_users",
			user: User{Campus: []Campus{{ID: 1, Name: "Paris"}}},
			want: "Paris",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.user.PrimaryCampus()
			if got == nil {
				t.Fatal("PrimaryCampus = nil")
			}
			if got.Name != tt.want {
				t.Errorf("PrimaryCampus = %q, want %q", got.Name, tt.want)
			}
		})
	}
}

func TestUser_PrimaryCampus_Empty(t *testing.T) {
	t.Parallel()

	user := User{}
	if got := user.PrimaryCampus(); got != nil {
		t.Fatalf("PrimaryCampus = %v, want nil", got)
	}
}

func TestProjectUser_Passed(t *testing.T) {
	t.Parallel()

	yes, no := true, false
	tests := []struct {
		name string
		pu   ProjectUser
		want bool
	}{
		{"validated", ProjectUser{Status: ProjectStatusFinished, Validated: &yes}, true},
		{"failed", ProjectUser{Status: ProjectStatusFinished, Validated: &no}, false},
		{"in progress", ProjectUser{Status: ProjectStatusInProgress}, false},
	}

	for _, tt := range tests {
		if got := tt.pu.Passed(); got != tt.want {
			t.Errorf("%s: Passed = %v, want %v", tt.name, got, tt.want)
		}
	}
}
