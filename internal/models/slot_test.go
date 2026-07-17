package models

import (
	"encoding/json"
	"testing"
)

func TestSlotUnmarshal_InvisibleUser(t *testing.T) {
	t.Parallel()

	payload := `{
		"id": 27,
		"begin_at": "2017-11-24T20:15:00.000Z",
		"end_at": "2017-11-24T20:30:00.000Z",
		"scale_team": null,
		"user": "invisible"
	}`

	var slot Slot
	if err := json.Unmarshal([]byte(payload), &slot); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if slot.ID != 27 {
		t.Errorf("ID = %d, want 27", slot.ID)
	}
	if slot.User.Login != "" {
		t.Errorf("User = %+v, want zero (invisible)", slot.User)
	}
	if slot.Booked() {
		t.Error("slot livre não deveria estar Booked")
	}
}

func TestSlotUnmarshal_Booked(t *testing.T) {
	t.Parallel()

	payload := `{
		"id": 99,
		"scale_team": {"id": 7},
		"user": {"id": 1, "login": "jdiniz"}
	}`

	var slot Slot
	if err := json.Unmarshal([]byte(payload), &slot); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !slot.Booked() || slot.ScaleTeam.ID != 7 {
		t.Errorf("ScaleTeam = %+v, want id 7", slot.ScaleTeam)
	}
	if slot.User.Login != "jdiniz" {
		t.Errorf("User.Login = %q", slot.User.Login)
	}
}
