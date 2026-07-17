package models

import (
	"encoding/json"
	"testing"
)

func TestScaleTeamUnmarshal_VisibleParticipants(t *testing.T) {
	t.Parallel()

	payload := `{
		"id": 1,
		"begin_at": "2026-07-18T14:00:00.000Z",
		"corrector": {"id": 10, "login": "jdiniz"},
		"correcteds": [{"id": 20, "login": "malima-m"}],
		"team": {"name": "malima-m's group", "project_id": 1331}
	}`

	var st ScaleTeam
	if err := json.Unmarshal([]byte(payload), &st); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if st.Corrector.Login != "jdiniz" {
		t.Errorf("Corrector.Login = %q, want jdiniz", st.Corrector.Login)
	}
	if len(st.Correcteds) != 1 || st.Correcteds[0].Login != "malima-m" {
		t.Errorf("Correcteds = %+v, want malima-m", st.Correcteds)
	}
	if st.Team.Name != "malima-m's group" {
		t.Errorf("Team.Name = %q", st.Team.Name)
	}
	if !st.IsCorrector("jdiniz") || st.IsCorrector("malima-m") {
		t.Error("IsCorrector deveria valer só para jdiniz")
	}
}

func TestScaleTeamUnmarshal_InvisibleParticipants(t *testing.T) {
	t.Parallel()

	payload := `{
		"id": 2,
		"corrector": "invisible",
		"correcteds": "invisible",
		"team": {"name": "secret", "project_id": 1}
	}`

	var st ScaleTeam
	if err := json.Unmarshal([]byte(payload), &st); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if st.Corrector.Login != "" {
		t.Errorf("Corrector = %+v, want zero (invisible)", st.Corrector)
	}
	if st.Correcteds != nil {
		t.Errorf("Correcteds = %+v, want nil (invisible)", st.Correcteds)
	}
	if st.IsCorrector("jdiniz") {
		t.Error("IsCorrector deveria ser false com corrector invisível")
	}
}
