package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

type stubSlotsStore struct {
	list       []models.Slot
	created    []models.Slot
	listErr    error
	createErr  error
	deleteErr  error
	gotUser    int
	gotBegin   time.Time
	gotEnd     time.Time
	deleted    int
	deletedIDs []int
}

func (s *stubSlotsStore) ListMine(context.Context) ([]models.Slot, error) {
	return s.list, s.listErr
}

func (s *stubSlotsStore) Create(_ context.Context, userID int, begin, end time.Time) ([]models.Slot, error) {
	s.gotUser, s.gotBegin, s.gotEnd = userID, begin, end
	return s.created, s.createErr
}

func (s *stubSlotsStore) Delete(_ context.Context, id int) error {
	s.deleted = id
	s.deletedIDs = append(s.deletedIDs, id)
	return s.deleteErr
}

type stubMeID struct {
	user *models.User
	err  error
}

func (s stubMeID) Me(context.Context) (*models.User, error) { return s.user, s.err }

func TestSlotsService_Open_FromDuration(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local)
	store := &stubSlotsStore{created: []models.Slot{{ID: 1}}}
	svc := NewSlotsService(store, stubMeID{user: &models.User{ID: 42}})
	svc.now = func() time.Time { return fixed }

	created, err := svc.Open(context.Background(), OpenRequest{
		From:     "2026-07-18 14:00",
		Duration: "1h",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %+v", created)
	}
	if store.gotUser != 42 {
		t.Errorf("userID = %d", store.gotUser)
	}
	wantBegin := time.Date(2026, 7, 18, 14, 0, 0, 0, time.Local)
	if !store.gotBegin.Equal(wantBegin) || !store.gotEnd.Equal(wantBegin.Add(time.Hour)) {
		t.Errorf("window = %v..%v", store.gotBegin, store.gotEnd)
	}
}

func TestSlotsService_Open_FromTo(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local)
	store := &stubSlotsStore{created: []models.Slot{{ID: 1}}}
	svc := NewSlotsService(store, stubMeID{user: &models.User{ID: 7}})
	svc.now = func() time.Time { return fixed }

	_, err := svc.Open(context.Background(), OpenRequest{
		From: "2026-07-18 14:00",
		To:   "2026-07-18 15:00",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	wantEnd := time.Date(2026, 7, 18, 15, 0, 0, 0, time.Local)
	if !store.gotEnd.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", store.gotEnd, wantEnd)
	}
}

func TestSlotsService_Open_DurationOnly_Earliest(t *testing.T) {
	t.Parallel()

	// 12:01 + 30m = 12:31 → round up to 12:45
	fixed := time.Date(2026, 7, 17, 12, 1, 0, 0, time.Local)
	store := &stubSlotsStore{created: []models.Slot{{ID: 1}}}
	svc := NewSlotsService(store, stubMeID{user: &models.User{ID: 42}})
	svc.now = func() time.Time { return fixed }

	_, err := svc.Open(context.Background(), OpenRequest{Duration: "30m"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	wantBegin := time.Date(2026, 7, 17, 12, 45, 0, 0, time.Local)
	if !store.gotBegin.Equal(wantBegin) {
		t.Errorf("begin = %v, want %v", store.gotBegin, wantBegin)
	}
	if !store.gotEnd.Equal(wantBegin.Add(30 * time.Minute)) {
		t.Errorf("end = %v", store.gotEnd)
	}
}

func TestSlotsService_Open_RejectsBothOrNeither(t *testing.T) {
	t.Parallel()

	svc := NewSlotsService(&stubSlotsStore{}, stubMeID{user: &models.User{ID: 1}})
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.Local) }

	_, err := svc.Open(context.Background(), OpenRequest{From: "2026-07-18 14:00", To: "x", Duration: "1h"})
	if !errors.Is(err, ErrSlotBothBounds) {
		t.Errorf("ambos: err = %v", err)
	}
	_, err = svc.Open(context.Background(), OpenRequest{From: "2026-07-18 14:00"})
	if !errors.Is(err, ErrSlotMissingBound) {
		t.Errorf("nenhum: err = %v", err)
	}
	_, err = svc.Open(context.Background(), OpenRequest{To: "2026-07-18 15:00"})
	if !errors.Is(err, ErrSlotToNeedsFrom) {
		t.Errorf("--to sem --from: err = %v", err)
	}
}

func TestSlotsService_Open_TooSoon(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 7, 18, 14, 0, 0, 0, time.Local)
	svc := NewSlotsService(&stubSlotsStore{}, stubMeID{user: &models.User{ID: 1}})
	svc.now = func() time.Time { return fixed }

	_, err := svc.Open(context.Background(), OpenRequest{
		From:     "2026-07-18 14:10",
		Duration: "30m",
	})
	if err == nil {
		t.Fatal("deveria rejeitar slot com menos de 30m de antecedência")
	}
}

func TestSlotsService_Close(t *testing.T) {
	t.Parallel()

	store := &stubSlotsStore{list: []models.Slot{
		{ID: 10},
		{ID: 11, ScaleTeam: &models.SlotScaleTeam{ID: 1}},
	}}
	svc := NewSlotsService(store, stubMeID{})

	if err := svc.Close(context.Background(), 10); err != nil {
		t.Fatalf("Close livre: %v", err)
	}
	if store.deleted != 10 {
		t.Errorf("deleted = %d", store.deleted)
	}

	if err := svc.Close(context.Background(), 11); err == nil {
		t.Fatal("deveria recusar slot agendado")
	}
	if err := svc.Close(context.Background(), 999); err == nil {
		t.Fatal("deveria recusar id inexistente")
	}
}

func TestSlotsService_CloseAll(t *testing.T) {
	t.Parallel()

	store := &stubSlotsStore{list: []models.Slot{
		{ID: 10},
		{ID: 11, ScaleTeam: &models.SlotScaleTeam{ID: 1}},
		{ID: 12},
	}}
	svc := NewSlotsService(store, stubMeID{})

	closed, skipped, err := svc.CloseAll(context.Background())
	if err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
	if closed != 2 || skipped != 1 {
		t.Errorf("closed=%d skipped=%d, want 2 e 1", closed, skipped)
	}
	if len(store.deletedIDs) != 2 || store.deletedIDs[0] != 10 || store.deletedIDs[1] != 12 {
		t.Errorf("deletedIDs = %v, want [10 12]", store.deletedIDs)
	}
}
