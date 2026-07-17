package models

import "time"

// Location is an active session at a campus workstation
// (/v2/campus/:id/locations).
type Location struct {
	ID      int         `json:"id"`
	Host    string      `json:"host"`
	BeginAt *time.Time  `json:"begin_at"`
	User    UserSummary `json:"user"`
}
