package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/notify"
)

// stateRetention keeps an evaluation in the "already notified" set for a
// while after it starts. The API only lists future scale teams, so without a
// grace period an evaluation that just began would look new again if it ever
// reappeared in the listing.
const stateRetention = 24 * time.Hour

// evaluationTags are the emoji shown alongside the push notification.
var evaluationTags = []string{"calendar"}

// EvaluationsSource provides the data NotifyService watches.
// Implemented by *UserService.
type EvaluationsSource interface {
	Me(ctx context.Context) (*models.User, error)
	UpcomingEvaluations(ctx context.Context) ([]models.ScaleTeam, error)
}

// NotifyService detects newly scheduled evaluations and pushes them to the
// user's phone, keeping track of what was already announced.
type NotifyService struct {
	source EvaluationsSource
	sender notify.Sender
	store  notify.Store
}

// NewNotifyService wires the evaluations source, the push sender and the
// state store.
func NewNotifyService(source EvaluationsSource, sender notify.Sender, store notify.Store) *NotifyService {
	return &NotifyService{source: source, sender: sender, store: store}
}

// NotifyReport summarizes one check.
type NotifyReport struct {
	// Baseline is true on the very first check, when the current schedule was
	// recorded without notifying (otherwise every existing evaluation would
	// arrive as a push at once).
	Baseline bool
	// Notified lists the evaluations pushed in this check.
	Notified []models.ScaleTeam
	// Scheduled is how many evaluations are on the schedule right now.
	Scheduled int
}

// Check fetches the schedule, notifies about evaluations never seen before
// and persists the updated state.
//
// An evaluation is only marked as seen once its notification was delivered,
// so a failed push is retried on the next check.
func (s *NotifyService) Check(ctx context.Context, now time.Time) (NotifyReport, error) {
	me, err := s.source.Me(ctx)
	if err != nil {
		return NotifyReport{}, err
	}

	evaluations, err := s.source.UpcomingEvaluations(ctx)
	if err != nil {
		return NotifyReport{}, err
	}

	state, err := s.store.Load()
	if err != nil {
		return NotifyReport{}, err
	}

	report := NotifyReport{Baseline: !state.Seeded, Scheduled: len(evaluations)}
	next := retained(state, now)

	for _, st := range evaluations {
		if st.ID <= 0 {
			continue
		}
		if state.Has(st.ID) || report.Baseline {
			next[st.ID] = startOrNow(st, now)
			continue
		}
		if err := s.sender.Send(ctx, EvaluationNotification(st, me.Login)); err != nil {
			// Leave it out of the state so the next check tries again.
			err = fmt.Errorf("notificar avaliação #%d: %w", st.ID, err)
			return report, errors.Join(err, s.store.Save(notify.State{Evaluations: next}))
		}
		next[st.ID] = startOrNow(st, now)
		report.Notified = append(report.Notified, st)
	}

	if err := s.store.Save(notify.State{Evaluations: next}); err != nil {
		return report, err
	}
	return report, nil
}

// retained copies the entries that must survive this check: evaluations
// already notified whose start is still within the retention window.
func retained(state notify.State, now time.Time) map[int]time.Time {
	next := make(map[int]time.Time, len(state.Evaluations))
	for id, begin := range state.Evaluations {
		if now.Sub(begin) < stateRetention {
			next[id] = begin
		}
	}
	return next
}

// startOrNow is the timestamp recorded for an evaluation, falling back to the
// check time when the API omits begin_at.
func startOrNow(st models.ScaleTeam, now time.Time) time.Time {
	if st.BeginAt == nil {
		return now
	}
	return *st.BeginAt
}

// EvaluationNotification phrases one evaluation as a push notification from
// the point of view of meLogin.
func EvaluationNotification(st models.ScaleTeam, meLogin string) notify.Notification {
	when := "horário a confirmar"
	if st.BeginAt != nil {
		when = st.BeginAt.Local().Format("02/01 às 15:04")
	}

	msg := notify.Notification{
		Tags:     evaluationTags,
		Priority: notify.PriorityHigh,
	}

	if st.IsCorrector(meLogin) {
		msg.Title = "Você vai avaliar " + teamName(st)
		msg.Message = fmt.Sprintf("%s — %s", when, correctedLogins(st))
		msg.Click = st.FillURL()
		return msg
	}

	corrector := st.Corrector.Login
	if corrector == "" {
		corrector = "Alguém"
	}
	msg.Title = corrector + " vai te avaliar"
	msg.Message = fmt.Sprintf("%s — %s", when, teamName(st))
	return msg
}

// teamName is the evaluated team, with a fallback when the API hides it.
func teamName(st models.ScaleTeam) string {
	if name := strings.TrimSpace(st.Team.Name); name != "" {
		return name
	}
	return "projeto não informado"
}

// correctedLogins lists who is being evaluated, or a placeholder when the
// Intra hides the participants.
func correctedLogins(st models.ScaleTeam) string {
	logins := make([]string, 0, len(st.Correcteds))
	for _, actor := range st.Correcteds {
		if actor.Login != "" {
			logins = append(logins, actor.Login)
		}
	}
	if len(logins) == 0 {
		return "avaliado não informado"
	}
	return strings.Join(logins, ", ")
}
