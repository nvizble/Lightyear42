package models

import (
	"bytes"
	"encoding/json"
	"time"
)

// Slot is an evaluation availability window (/v2/me/slots, /v2/slots).
// When ScaleTeam is non-nil the slot is already booked for a defense.
type Slot struct {
	ID        int            `json:"id"`
	BeginAt   *time.Time     `json:"begin_at"`
	EndAt     *time.Time     `json:"end_at"`
	ScaleTeam *SlotScaleTeam `json:"scale_team"`
	User      ScaleTeamActor `json:"user"`
}

// SlotScaleTeam is the booked evaluation linked to a slot, when present.
type SlotScaleTeam struct {
	ID int `json:"id"`
}

// Booked reports whether the slot already has a scheduled evaluation.
func (s *Slot) Booked() bool {
	return s.ScaleTeam != nil
}

// UnmarshalJSON tolerates user being the string "invisible".
func (s *Slot) UnmarshalJSON(data []byte) error {
	type plain struct {
		ID        int             `json:"id"`
		BeginAt   *time.Time      `json:"begin_at"`
		EndAt     *time.Time      `json:"end_at"`
		ScaleTeam *SlotScaleTeam  `json:"scale_team"`
		User      json.RawMessage `json:"user"`
	}
	var raw plain
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.ID = raw.ID
	s.BeginAt = raw.BeginAt
	s.EndAt = raw.EndAt
	s.ScaleTeam = raw.ScaleTeam
	s.User = ScaleTeamActor{}
	if len(raw.User) == 0 || bytes.Equal(raw.User, invisible) || bytes.Equal(raw.User, []byte("null")) {
		return nil
	}
	return json.Unmarshal(raw.User, &s.User)
}
