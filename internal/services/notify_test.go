package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/notify"
)

// fakeSource feeds the notify service a fixed schedule.
type fakeSource struct {
	me       *models.User
	meErr    error
	evals    []models.ScaleTeam
	evalsErr error
}

func (f *fakeSource) Me(context.Context) (*models.User, error) {
	if f.meErr != nil {
		return nil, f.meErr
	}
	return f.me, nil
}

func (f *fakeSource) UpcomingEvaluations(context.Context) ([]models.ScaleTeam, error) {
	return f.evals, f.evalsErr
}

// fakeSender records notifications and can fail on demand.
type fakeSender struct {
	sent []notify.Notification
	err  error
}

func (f *fakeSender) Send(_ context.Context, n notify.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, n)
	return nil
}

// memStore is an in-memory notify.Store.
type memStore struct {
	state notify.State
	saves int
}

func (m *memStore) Load() (notify.State, error) { return m.state, nil }

func (m *memStore) Save(s notify.State) error {
	m.saves++
	m.state = notify.State{Seeded: true, Evaluations: s.Evaluations}
	return nil
}

// scaleTeam builds a scheduled evaluation for the tests.
func scaleTeam(id int, begin time.Time, corrector string, correcteds ...string) models.ScaleTeam {
	actors := make(models.ScaleTeamActors, 0, len(correcteds))
	for _, login := range correcteds {
		actors = append(actors, models.ScaleTeamActor{Login: login})
	}
	return models.ScaleTeam{
		ID:         id,
		BeginAt:    &begin,
		Corrector:  models.ScaleTeamActor{Login: corrector},
		Correcteds: actors,
		Team:       models.EvaluationTeam{Name: "libft"},
	}
}

func newTestService(evals []models.ScaleTeam) (*NotifyService, *fakeSender, *memStore) {
	source := &fakeSource{me: &models.User{Login: "joaodini"}, evals: evals}
	sender := &fakeSender{}
	store := &memStore{state: notify.State{Evaluations: map[int]time.Time{}}}
	return NewNotifyService(source, sender, store), sender, store
}

func TestNotifyServiceFirstRunSeedsBaseline(t *testing.T) {
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	svc, sender, store := newTestService([]models.ScaleTeam{
		scaleTeam(1, now.Add(2*time.Hour), "joaodini", "alice"),
		scaleTeam(2, now.Add(3*time.Hour), "bob", "joaodini"),
	})

	report, err := svc.Check(context.Background(), now)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if !report.Baseline {
		t.Error("Baseline = false, want true on the first check")
	}
	if len(sender.sent) != 0 {
		t.Errorf("sent %d notifications, want 0 on the first check", len(sender.sent))
	}
	if report.Scheduled != 2 {
		t.Errorf("Scheduled = %d, want 2", report.Scheduled)
	}
	if !store.state.Has(1) || !store.state.Has(2) {
		t.Errorf("state = %v, want both evaluations recorded", store.state.Evaluations)
	}
}

func TestNotifyServiceNotifiesOnlyNew(t *testing.T) {
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	svc, sender, store := newTestService([]models.ScaleTeam{
		scaleTeam(1, now.Add(2*time.Hour), "joaodini", "alice"),
	})
	// Pretend a previous check already ran and recorded evaluation 1.
	store.state = notify.State{Seeded: true, Evaluations: map[int]time.Time{1: now.Add(2 * time.Hour)}}

	report, err := svc.Check(context.Background(), now)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(sender.sent) != 0 || len(report.Notified) != 0 {
		t.Fatalf("notified %d, want 0 for an unchanged schedule", len(report.Notified))
	}

	// A new booking appears.
	svc.source.(*fakeSource).evals = append(svc.source.(*fakeSource).evals,
		scaleTeam(2, now.Add(5*time.Hour), "bob", "joaodini"))

	report, err = svc.Check(context.Background(), now)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(report.Notified) != 1 || report.Notified[0].ID != 2 {
		t.Fatalf("Notified = %+v, want only evaluation 2", report.Notified)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d notifications, want 1", len(sender.sent))
	}
	if !strings.Contains(sender.sent[0].Title, "bob") {
		t.Errorf("title = %q, want the corrector login", sender.sent[0].Title)
	}

	// And it must not be announced twice.
	report, err = svc.Check(context.Background(), now)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(report.Notified) != 0 {
		t.Errorf("Notified = %+v, want nothing on a repeated check", report.Notified)
	}
}

func TestNotifyServiceFailedSendIsRetried(t *testing.T) {
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	svc, sender, store := newTestService([]models.ScaleTeam{
		scaleTeam(7, now.Add(time.Hour), "joaodini", "alice"),
	})
	store.state = notify.State{Seeded: true, Evaluations: map[int]time.Time{}}
	sender.err = errors.New("ntfy fora do ar")

	_, err := svc.Check(context.Background(), now)
	if err == nil {
		t.Fatal("Check() = nil error, want the send failure")
	}
	if store.state.Has(7) {
		t.Error("state recorded evaluation 7 despite the failed push")
	}

	// Next check succeeds and the notification finally goes out.
	sender.err = nil
	report, err := svc.Check(context.Background(), now)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(report.Notified) != 1 {
		t.Fatalf("Notified = %+v, want the retried evaluation", report.Notified)
	}
	if !store.state.Has(7) {
		t.Error("state did not record evaluation 7 after a successful push")
	}
}

func TestNotifyServicePrunesOldEntries(t *testing.T) {
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	svc, _, store := newTestService(nil)
	store.state = notify.State{Seeded: true, Evaluations: map[int]time.Time{
		1: now.Add(-stateRetention - time.Hour), // long past: drop
		2: now.Add(-time.Hour),                  // started recently: keep
		3: now.Add(time.Hour),                   // still upcoming: keep
	}}

	if _, err := svc.Check(context.Background(), now); err != nil {
		t.Fatalf("Check: %v", err)
	}

	if store.state.Has(1) {
		t.Error("Has(1) = true, want the stale entry pruned")
	}
	if !store.state.Has(2) || !store.state.Has(3) {
		t.Errorf("state = %v, want entries 2 and 3 kept", store.state.Evaluations)
	}
}

func TestNotifyServicePropagatesSourceErrors(t *testing.T) {
	svc, sender, _ := newTestService(nil)
	svc.source.(*fakeSource).evalsErr = errors.New("api fora do ar")

	if _, err := svc.Check(context.Background(), time.Now()); err == nil {
		t.Fatal("Check() = nil error, want the API failure")
	}
	if len(sender.sent) != 0 {
		t.Error("sent notifications despite the API failure")
	}
}

func TestEvaluationNotification(t *testing.T) {
	begin := time.Date(2026, 9, 14, 10, 30, 0, 0, time.UTC).Local()

	tests := []struct {
		name        string
		team        models.ScaleTeam
		wantTitle   string
		wantInBody  string
		wantClicked bool
	}{
		{
			name:        "como avaliador",
			team:        scaleTeam(1, begin, "joaodini", "alice"),
			wantTitle:   "Você vai avaliar libft",
			wantInBody:  "alice",
			wantClicked: true,
		},
		{
			name:       "como avaliado",
			team:       scaleTeam(2, begin, "bob", "joaodini"),
			wantTitle:  "bob vai te avaliar",
			wantInBody: "libft",
		},
		{
			name:       "avaliador escondido pela intra",
			team:       scaleTeam(3, begin, "", "joaodini"),
			wantTitle:  "Alguém vai te avaliar",
			wantInBody: "libft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluationNotification(tt.team, "joaodini")

			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if !strings.Contains(got.Message, tt.wantInBody) {
				t.Errorf("Message = %q, want it to mention %q", got.Message, tt.wantInBody)
			}
			if !strings.Contains(got.Message, begin.Local().Format("02/01")) {
				t.Errorf("Message = %q, want the local start date", got.Message)
			}
			if tt.wantClicked && got.Click == "" {
				t.Error("Click = empty, want the fill URL for the corrector")
			}
			if !tt.wantClicked && got.Click != "" {
				t.Errorf("Click = %q, want empty when not the corrector", got.Click)
			}
		})
	}
}

func TestEvaluationNotificationWithoutBeginAt(t *testing.T) {
	st := models.ScaleTeam{ID: 9, Corrector: models.ScaleTeamActor{Login: "joaodini"}}

	got := EvaluationNotification(st, "joaodini")
	if !strings.Contains(got.Message, "horário a confirmar") {
		t.Errorf("Message = %q, want the missing-time fallback", got.Message)
	}
	if !strings.Contains(got.Title, "projeto não informado") {
		t.Errorf("Title = %q, want the missing-team fallback", got.Title)
	}
	if !strings.Contains(got.Message, "avaliado não informado") {
		t.Errorf("Message = %q, want the hidden-participant fallback", got.Message)
	}
}
