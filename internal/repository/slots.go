package repository

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

// APIClient is the HTTP surface used by repositories that mutate state.
// Implemented by *api.Client. APIGetter remains for read-only repos.
type APIClient interface {
	APIGetter
	Post(ctx context.Context, path string, body any, out any) error
	Delete(ctx context.Context, path string) error
}

// Slots manages evaluation availability windows for the authenticated user.
type Slots interface {
	// ListMine returns the token owner's future slots, soonest first.
	ListMine(ctx context.Context) ([]models.Slot, error)
	// Create opens availability from begin to end for userID.
	// The API may return several 15-minute slots for longer windows.
	Create(ctx context.Context, userID int, begin, end time.Time) ([]models.Slot, error)
	// Delete closes the slot with the given id.
	Delete(ctx context.Context, id int) error
}

// SlotsRepository implements Slots over the API client.
// Slots are not cached: they change with every open/close.
type SlotsRepository struct {
	api APIClient
}

// NewSlotsRepository wires the API client.
func NewSlotsRepository(client APIClient) *SlotsRepository {
	return &SlotsRepository{api: client}
}

// ListMine lists future slots of the resource owner.
func (r *SlotsRepository) ListMine(ctx context.Context) ([]models.Slot, error) {
	query := url.Values{
		"filter[future]": {"true"},
		"sort":           {"begin_at"},
		"page[size]":     {"100"},
	}

	var slots []models.Slot
	if err := r.api.Get(ctx, "/me/slots", query, &slots); err != nil {
		return nil, err
	}
	return slots, nil
}

// createSlotBody is the JSON payload for POST /v2/slots.
type createSlotBody struct {
	Slot createSlotFields `json:"slot"`
}

type createSlotFields struct {
	UserID  int    `json:"user_id"`
	BeginAt string `json:"begin_at"`
	EndAt   string `json:"end_at"`
}

// Create opens one or more slots covering [begin, end].
func (r *SlotsRepository) Create(ctx context.Context, userID int, begin, end time.Time) ([]models.Slot, error) {
	body := createSlotBody{Slot: createSlotFields{
		UserID:  userID,
		BeginAt: begin.UTC().Format(time.RFC3339),
		EndAt:   end.UTC().Format(time.RFC3339),
	}}

	var created []models.Slot
	if err := r.api.Post(ctx, "/slots", body, &created); err != nil {
		return nil, err
	}
	return created, nil
}

// Delete removes a slot by id.
func (r *SlotsRepository) Delete(ctx context.Context, id int) error {
	if id < 1 {
		return fmt.Errorf("id de slot inválido: %d", id)
	}
	return r.api.Delete(ctx, "/slots/"+strconv.Itoa(id))
}
