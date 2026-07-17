package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
	"github.com/joaodiniz/42cli/internal/repository"
)

// Slot timing constraints from the Intra (campus may tighten the minimum).
const (
	slotMinDuration = 30 * time.Minute
	slotMinLeadTime = 30 * time.Minute
	slotMaxHorizon  = 14 * 24 * time.Hour
)

var (
	// ErrSlotBothBounds means both --to and --duration were set.
	ErrSlotBothBounds = errors.New("informe --to ou --duration, não os dois")
	// ErrSlotMissingBound means neither --to nor --duration was set.
	ErrSlotMissingBound = errors.New("informe --to ou --duration")
	// ErrSlotToNeedsFrom means --to was set without --from.
	ErrSlotToNeedsFrom = errors.New("--to exige --from (ou use só --duration para o mais cedo possível)")
)

// slotGrid is the Intra's slot alignment (15 minutes).
const slotGrid = 15 * time.Minute

// slotsStore lists and mutates slots. Implemented by *repository.SlotsRepository.
type slotsStore interface {
	ListMine(ctx context.Context) ([]models.Slot, error)
	Create(ctx context.Context, userID int, begin, end time.Time) ([]models.Slot, error)
	Delete(ctx context.Context, id int) error
}

// meIDReader returns the authenticated profile (for user_id on create).
type meIDReader interface {
	Me(ctx context.Context) (*models.User, error)
}

// SlotsService implements list/open/close for evaluation slots.
type SlotsService struct {
	slots slotsStore
	users meIDReader
	now   func() time.Time
}

// NewSlotsService wires the slots repository and user profile reader.
func NewSlotsService(slots slotsStore, users meIDReader) *SlotsService {
	return &SlotsService{slots: slots, users: users, now: time.Now}
}

// Ensure the concrete repository satisfies slotsStore at compile time.
var _ slotsStore = (*repository.SlotsRepository)(nil)

// List returns future slots, soonest first (API already sorts).
func (s *SlotsService) List(ctx context.Context) ([]models.Slot, error) {
	return s.slots.ListMine(ctx)
}

// OpenRequest is the input for opening slots from CLI flags.
type OpenRequest struct {
	From     string // local datetime; optional when Duration alone is set
	To       string // local datetime; mutually exclusive with Duration
	Duration string // e.g. "30m", "1h"; alone → starts at earliest allowed time
}

// Open creates availability for [from, to], resolving to/duration.
func (s *SlotsService) Open(ctx context.Context, req OpenRequest) ([]models.Slot, error) {
	begin, end, err := resolveSlotWindow(req, s.now())
	if err != nil {
		return nil, err
	}
	if err := validateSlotWindow(begin, end, s.now()); err != nil {
		return nil, err
	}

	me, err := s.users.Me(ctx)
	if err != nil {
		return nil, err
	}

	return s.slots.Create(ctx, me.ID, begin, end)
}

// Close deletes an open (unbooked) future slot owned by the user.
func (s *SlotsService) Close(ctx context.Context, id int) error {
	if id < 1 {
		return fmt.Errorf("informe um id de slot válido")
	}

	slots, err := s.slots.ListMine(ctx)
	if err != nil {
		return err
	}

	var found *models.Slot
	for i := range slots {
		if slots[i].ID == id {
			found = &slots[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("slot %d não encontrado entre os seus slots futuros", id)
	}
	if found.Booked() {
		return fmt.Errorf("slot %d já está agendado para uma avaliação e não pode ser fechado", id)
	}

	return s.slots.Delete(ctx, id)
}

// CloseAll deletes every future free (unbooked) slot. Booked ones are skipped.
func (s *SlotsService) CloseAll(ctx context.Context) (closed, skipped int, err error) {
	slots, err := s.slots.ListMine(ctx)
	if err != nil {
		return 0, 0, err
	}

	for _, slot := range slots {
		if slot.Booked() {
			skipped++
			continue
		}
		if err := s.slots.Delete(ctx, slot.ID); err != nil {
			return closed, skipped, fmt.Errorf("fechar slot %d: %w", slot.ID, err)
		}
		closed++
	}
	return closed, skipped, nil
}

// resolveSlotWindow parses open flags into [begin, end].
//
// Allowed combinations:
//   - --duration alone → begin = earliest allowed (now+30m, rounded up to 15m)
//   - --from + --duration
//   - --from + --to
func resolveSlotWindow(req OpenRequest, now time.Time) (time.Time, time.Time, error) {
	from := strings.TrimSpace(req.From)
	to := strings.TrimSpace(req.To)
	duration := strings.TrimSpace(req.Duration)

	if to != "" && duration != "" {
		return time.Time{}, time.Time{}, ErrSlotBothBounds
	}
	if to == "" && duration == "" {
		return time.Time{}, time.Time{}, ErrSlotMissingBound
	}
	if to != "" && from == "" {
		return time.Time{}, time.Time{}, ErrSlotToNeedsFrom
	}

	var begin time.Time
	var err error
	if from == "" {
		// Duration-only: start ASAP.
		begin = earliestSlotBegin(now)
	} else {
		begin, err = parseLocalDateTime(from, now)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("--from: %w", err)
		}
	}

	var end time.Time
	if to != "" {
		end, err = parseLocalDateTime(to, now)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("--to: %w", err)
		}
	} else {
		d, err := time.ParseDuration(duration)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("--duration: use formatos como 30m ou 1h: %w", err)
		}
		if d <= 0 {
			return time.Time{}, time.Time{}, fmt.Errorf("--duration deve ser positiva")
		}
		end = begin.Add(d)
	}

	return begin, end, nil
}

// earliestSlotBegin is now + min lead time, rounded up to the 15-minute grid.
func earliestSlotBegin(now time.Time) time.Time {
	return roundUpToSlotGrid(now.Add(slotMinLeadTime))
}

// roundUpToSlotGrid aligns t to the next (or current) 15-minute boundary.
func roundUpToSlotGrid(t time.Time) time.Time {
	t = t.Truncate(time.Minute)
	mins := t.Hour()*60 + t.Minute()
	rem := mins % int(slotGrid/time.Minute)
	if rem == 0 {
		return t
	}
	return t.Add(time.Duration(int(slotGrid/time.Minute)-rem) * time.Minute)
}

// validateSlotWindow enforces Intra timing rules.
func validateSlotWindow(begin, end, now time.Time) error {
	if !end.After(begin) {
		return fmt.Errorf("o fim do slot deve ser depois do início")
	}
	if end.Sub(begin) < slotMinDuration {
		return fmt.Errorf("duração mínima do slot é %s", slotMinDuration)
	}
	if begin.Before(now.Add(slotMinLeadTime)) {
		return fmt.Errorf("o slot deve começar pelo menos %s no futuro", slotMinLeadTime)
	}
	if begin.After(now.Add(slotMaxHorizon)) {
		return fmt.Errorf("o slot deve começar no máximo daqui a 2 semanas")
	}
	return nil
}

// localDateTimeLayouts are accepted --from / --to formats (local zone).
var localDateTimeLayouts = []string{
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	time.RFC3339,
}

// parseLocalDateTime parses s in the local timezone (or as RFC3339 as-is).
func parseLocalDateTime(s string, now time.Time) (time.Time, error) {
	loc := now.Location()
	for _, layout := range localDateTimeLayouts {
		if layout == time.RFC3339 {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
			continue
		}
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("use o formato \"YYYY-MM-DD HH:MM\" (hora local)")
}
