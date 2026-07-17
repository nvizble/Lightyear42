package repository

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

// slotsFakeAPI records Get/Post/Delete for slots tests.
type slotsFakeAPI struct {
	getPath    string
	getQuery   url.Values
	postPath   string
	postBody   []byte
	deletePath string
	list       []models.Slot
	created    []models.Slot
	err        error
}

func (f *slotsFakeAPI) Get(_ context.Context, path string, query url.Values, out any) error {
	f.getPath = path
	f.getQuery = query
	if f.err != nil {
		return f.err
	}
	target, ok := out.(*[]models.Slot)
	if !ok {
		return errUnexpectedOut
	}
	*target = f.list
	return nil
}

func (f *slotsFakeAPI) Post(_ context.Context, path string, body any, out any) error {
	f.postPath = path
	f.postBody, _ = json.Marshal(body)
	if f.err != nil {
		return f.err
	}
	if target, ok := out.(*[]models.Slot); ok {
		*target = f.created
	}
	return nil
}

func (f *slotsFakeAPI) Delete(_ context.Context, path string) error {
	f.deletePath = path
	return f.err
}

var errUnexpectedOut = errString("unexpected out type")

type errString string

func (e errString) Error() string { return string(e) }

func TestSlotsRepository_ListMine(t *testing.T) {
	t.Parallel()

	api := &slotsFakeAPI{list: []models.Slot{{ID: 1}, {ID: 2}}}
	repo := NewSlotsRepository(api)

	slots, err := repo.ListMine(context.Background())
	if err != nil {
		t.Fatalf("ListMine: %v", err)
	}
	if api.getPath != "/me/slots" {
		t.Errorf("path = %q", api.getPath)
	}
	if api.getQuery.Get("filter[future]") != "true" {
		t.Errorf("filter[future] = %q", api.getQuery.Get("filter[future]"))
	}
	if len(slots) != 2 {
		t.Errorf("len = %d, want 2", len(slots))
	}
}

func TestSlotsRepository_Create(t *testing.T) {
	t.Parallel()

	begin := time.Date(2026, 7, 18, 17, 0, 0, 0, time.UTC)
	end := begin.Add(30 * time.Minute)
	api := &slotsFakeAPI{created: []models.Slot{{ID: 10}, {ID: 11}}}
	repo := NewSlotsRepository(api)

	created, err := repo.Create(context.Background(), 42, begin, end)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if api.postPath != "/slots" {
		t.Errorf("path = %q", api.postPath)
	}
	body := string(api.postBody)
	for _, want := range []string{`"user_id":42`, `"begin_at":"2026-07-18T17:00:00Z"`, `"end_at":"2026-07-18T17:30:00Z"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %s: %s", want, body)
		}
	}
	if len(created) != 2 {
		t.Errorf("created len = %d", len(created))
	}
}

func TestSlotsRepository_Delete(t *testing.T) {
	t.Parallel()

	api := &slotsFakeAPI{}
	repo := NewSlotsRepository(api)
	if err := repo.Delete(context.Background(), 99); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if api.deletePath != "/slots/99" {
		t.Errorf("path = %q", api.deletePath)
	}
}
