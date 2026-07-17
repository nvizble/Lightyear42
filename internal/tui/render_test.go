package tui

import (
	"strings"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

func testUser() *models.User {
	return &models.User{
		Login:           "jdiniz",
		Displayname:     "João Diniz",
		Email:           "jdiniz@student.42porto.com",
		Wallet:          50,
		CorrectionPoint: 3,
		Location:        "c1r2p3",
		Campus:          []models.Campus{{Name: "Porto"}},
		CursusUsers: []models.CursusUser{
			{Level: 9.5, Grade: "Novice", Cursus: models.Cursus{Name: "C Piscine", Kind: "piscine"}},
			{Level: 8.43, Grade: "Cadet", Cursus: models.Cursus{Name: "42cursus", Kind: "main"}},
		},
	}
}

func TestRenderUser(t *testing.T) {
	t.Parallel()

	out := RenderUser(testUser())

	for _, want := range []string{
		"João Diniz", "jdiniz", "Porto", "42cursus", "Level 8.43", "50 ₳", "c1r2p3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// The main cursus must win even when the piscine has a higher level.
	if strings.Contains(out, "Level 9.50") {
		t.Error("output shows piscine level instead of main cursus")
	}
}

func TestRenderUser_Offline(t *testing.T) {
	t.Parallel()

	user := testUser()
	user.Location = ""

	if out := RenderUser(user); !strings.Contains(out, "offline") {
		t.Errorf("output missing offline marker:\n%s", out)
	}
}

func TestRenderUserList(t *testing.T) {
	t.Parallel()

	out := RenderUserList([]models.UserSummary{
		{Login: "jdiniz", Displayname: "João Diniz", Location: "c1r2p3"},
		{Login: "jdinis", Displayname: "Someone Else"},
	})

	for _, want := range []string{"LOGIN", "jdiniz", "jdinis", "c1r2p3", "-"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderUserList_Empty(t *testing.T) {
	t.Parallel()

	if out := RenderUserList(nil); !strings.Contains(out, "Nenhum usuário") {
		t.Errorf("output = %q, want empty-state message", out)
	}
}
