package models

import "time"

// Project statuses returned by the 42 API.
const (
	ProjectStatusFinished   = "finished"
	ProjectStatusInProgress = "in_progress"
)

// ProjectUser is a user's enrolment in a project, as embedded in
// /v2/me and /v2/users/:login under projects_users.
type ProjectUser struct {
	ID        int        `json:"id"`
	FinalMark *int       `json:"final_mark"`
	Status    string     `json:"status"`
	Validated *bool      `json:"validated?"`
	MarkedAt  *time.Time `json:"marked_at"`
	CursusIDs []int      `json:"cursus_ids"`
	Project   Project    `json:"project"`
}

// Project identifies a 42 project.
type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// InCursus reports whether this enrolment belongs to the given cursus.
func (pu *ProjectUser) InCursus(cursusID int) bool {
	for _, id := range pu.CursusIDs {
		if id == cursusID {
			return true
		}
	}
	return false
}

// Passed reports whether the project was finished and validated.
func (pu *ProjectUser) Passed() bool {
	return pu.Status == ProjectStatusFinished && pu.Validated != nil && *pu.Validated
}
