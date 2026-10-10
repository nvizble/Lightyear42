package models

import (
	"encoding/json"
	"testing"
)

func TestSlotScaleTeamInvisible(t *testing.T) {
	var s Slot
	if err := json.Unmarshal([]byte(`{"id":1,"scale_team":"invisible","user":"invisible"}`), &s); err != nil {
		t.Fatal(err)
	}
	if !s.Booked() {
		t.Fatal("slot com scale_team oculto deve contar como reservado")
	}
}
