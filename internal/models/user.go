// Package models defines domain types shared across layers.
package models

import "time"

// User is a 42 student/staff profile as returned by /v2/me and /v2/users/:login.
// Only the fields relevant to the CLI are mapped.
type User struct {
	ID              int           `json:"id"`
	Login           string        `json:"login"`
	Email           string        `json:"email"`
	Displayname     string        `json:"displayname"`
	Image           Image         `json:"image"`
	Wallet          int           `json:"wallet"`
	CorrectionPoint int           `json:"correction_point"`
	Location        string        `json:"location"`
	Staff           bool          `json:"staff?"`
	CursusUsers     []CursusUser  `json:"cursus_users"`
	Campus          []Campus      `json:"campus"`
	CampusUsers     []CampusUser  `json:"campus_users"`
	ProjectsUsers   []ProjectUser `json:"projects_users"`
}

// CampusUser links a user to a campus, flagging the primary one.
type CampusUser struct {
	CampusID  int  `json:"campus_id"`
	IsPrimary bool `json:"is_primary"`
}

// Image holds the profile picture URLs.
type Image struct {
	Link string `json:"link"`
}

// CursusUser is the enrolment of a user in a cursus, carrying level and grade.
type CursusUser struct {
	Level   float64    `json:"level"`
	Grade   string     `json:"grade"`
	BeginAt *time.Time `json:"begin_at"`
	EndAt   *time.Time `json:"end_at"`
	Cursus  Cursus     `json:"cursus"`
}

// Cursus identifies a study track (e.g. "42cursus", "C Piscine").
type Cursus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	// Kind classifies the cursus: "main" (42cursus), "piscine", "test", ...
	Kind string `json:"kind"`
}

// IsMain reports whether this enrolment belongs to a main cursus (e.g. 42cursus).
func (cu *CursusUser) IsMain() bool {
	return cu.Cursus.Kind == "main"
}

// Active reports whether the enrolment is ongoing (no end date, or one in the future).
func (cu *CursusUser) Active(now time.Time) bool {
	return cu.EndAt == nil || cu.EndAt.After(now)
}

// Campus is a 42 campus.
type Campus struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Country  string `json:"country"`
	TimeZone string `json:"time_zone"`
}

// UserSummary is the compact user object returned by list endpoints (/v2/users).
type UserSummary struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	Displayname string `json:"displayname"`
	Location    string `json:"location"`
	Image       Image  `json:"image"`
}

// PrimaryCampus returns the user's primary campus (transfer students belong
// to several). Falls back to the first campus; nil when the user has none.
func (u *User) PrimaryCampus() *Campus {
	for _, cu := range u.CampusUsers {
		if !cu.IsPrimary {
			continue
		}
		for i := range u.Campus {
			if u.Campus[i].ID == cu.CampusID {
				return &u.Campus[i]
			}
		}
	}
	if len(u.Campus) > 0 {
		return &u.Campus[0]
	}
	return nil
}

// MainCursus returns the user's primary enrolment, mirroring how the Intra
// picks the profile cursus. Preference order: main-kind cursus (42cursus)
// over piscines, active enrolments over finished ones, then the most
// recently started. Returns nil when the user has none.
func (u *User) MainCursus() *CursusUser {
	now := time.Now()
	var best *CursusUser
	for i := range u.CursusUsers {
		candidate := &u.CursusUsers[i]
		if best == nil || moreRelevant(candidate, best, now) {
			best = candidate
		}
	}
	return best
}

// moreRelevant reports whether a should be preferred over b as primary cursus.
func moreRelevant(a, b *CursusUser, now time.Time) bool {
	if a.IsMain() != b.IsMain() {
		return a.IsMain()
	}
	if a.Active(now) != b.Active(now) {
		return a.Active(now)
	}
	return beginTime(a).After(beginTime(b))
}

// beginTime returns the enrolment start, zero when unknown.
func beginTime(cu *CursusUser) time.Time {
	if cu.BeginAt == nil {
		return time.Time{}
	}
	return *cu.BeginAt
}
