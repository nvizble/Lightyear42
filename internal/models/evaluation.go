package models

import (
	"bytes"
	"encoding/json"
	"time"
)

// ScaleTeam is a scheduled evaluation (/v2/me/scale_teams): someone
// evaluating a team, with the authenticated user on either side.
type ScaleTeam struct {
	ID         int             `json:"id"`
	BeginAt    *time.Time      `json:"begin_at"`
	Corrector  ScaleTeamActor  `json:"corrector"`
	Correcteds ScaleTeamActors `json:"correcteds"`
	Team       EvaluationTeam  `json:"team"`
}

// EvaluationTeam is the team under evaluation.
type EvaluationTeam struct {
	Name      string `json:"name"`
	ProjectID int    `json:"project_id"`
}

// ScaleTeamActor is one participant of an evaluation. A zero value means
// the Intra hid the identity ("invisible").
type ScaleTeamActor struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

// invisible is what the API returns instead of participants it hides.
var invisible = []byte(`"invisible"`)

// UnmarshalJSON tolerates the "invisible" placeholder the API uses to hide
// the participant, decoding it as a zero actor.
func (a *ScaleTeamActor) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, invisible) || bytes.Equal(data, []byte("null")) {
		*a = ScaleTeamActor{}
		return nil
	}
	type plain ScaleTeamActor
	var v plain
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*a = ScaleTeamActor(v)
	return nil
}

// ScaleTeamActors is a participant list that may also be hidden as a whole.
type ScaleTeamActors []ScaleTeamActor

// UnmarshalJSON tolerates "invisible" in place of the whole list.
func (as *ScaleTeamActors) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, invisible) || bytes.Equal(data, []byte("null")) {
		*as = nil
		return nil
	}
	var v []ScaleTeamActor
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*as = v
	return nil
}

// IsCorrector reports whether login is the evaluator of this scale team.
func (st *ScaleTeam) IsCorrector(login string) bool {
	return st.Corrector.Login != "" && st.Corrector.Login == login
}
