package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/notify"
	"github.com/nvizble/Lightyear42/internal/services"
)

type fakeSchedule struct{ evals []models.ScaleTeam }

func (f *fakeSchedule) Me(context.Context) (*models.User, error) {
	return &models.User{Login: "marvin"}, nil
}

func (f *fakeSchedule) UpcomingEvaluations(context.Context) ([]models.ScaleTeam, error) {
	return f.evals, nil
}

type fakePhone struct{ sent []string }

func (f *fakePhone) Send(_ context.Context, n notify.Notification) error {
	f.sent = append(f.sent, n.Title)
	return nil
}

func TestAppNotify(t *testing.T) {
	begin := time.Now().Add(2 * time.Hour)
	schedule, phone := &fakeSchedule{}, &fakePhone{}
	check := appNotify(services.NewNotifyService(schedule, phone, notify.NewFileStore(t.TempDir()+"/state.json")))

	if status, err := check(context.Background()); status != "" || err != nil {
		t.Fatalf("a primeira checagem só registra a linha de base: %q %v", status, err)
	}
	schedule.evals = []models.ScaleTeam{{ID: 7, BeginAt: &begin, Team: models.EvaluationTeam{Name: "ft_printf"}}}
	status, err := check(context.Background())
	if err != nil || len(phone.sent) != 1 || !strings.HasPrefix(status, "avisado no celular: ") || !strings.Contains(status, "ft_printf") {
		t.Fatalf("avaliação nova: %q %v, enviados %v", status, err, phone.sent)
	}
	if status, _ := check(context.Background()); status != "" || len(phone.sent) != 1 {
		t.Fatalf("a mesma avaliação não é avisada de novo: %q", status)
	}
}
