package repository

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"testing"

	"github.com/nvizble/Lightyear42/internal/models"
)

// fakeCampusAPI serves paginated locations.
type fakeCampusAPI struct {
	locations []models.Location
	calls     int
}

func (f *fakeCampusAPI) Get(_ context.Context, path string, query url.Values, out any) error {
	f.calls++

	target, ok := out.(*[]models.Location)
	if !ok {
		return fmt.Errorf("unexpected out type for %s", path)
	}

	page, _ := strconv.Atoi(query.Get("page[number]"))
	size, _ := strconv.Atoi(query.Get("page[size]"))
	start := (page - 1) * size
	if start >= len(f.locations) {
		*target = nil
		return nil
	}
	*target = f.locations[start:min(start+size, len(f.locations))]
	return nil
}

func makeLocations(n int) []models.Location {
	locations := make([]models.Location, n)
	for i := range locations {
		locations[i] = models.Location{
			ID:   i + 1,
			Host: fmt.Sprintf("c1r1p%d", i+1),
			User: models.UserSummary{Login: fmt.Sprintf("user%d", i+1)},
		}
	}
	return locations
}

func TestCampusRepository_ActiveLocations_Paginates(t *testing.T) {
	t.Parallel()

	api := &fakeCampusAPI{locations: makeLocations(230)}
	repo := NewCampusRepository(api, newMemCache())

	locations, err := repo.ActiveLocations(context.Background(), 30)
	if err != nil {
		t.Fatalf("ActiveLocations: %v", err)
	}
	if len(locations) != 230 {
		t.Fatalf("len = %d, want 230", len(locations))
	}
	// 3 pages: 100 + 100 + 30 (a última, incompleta, encerra o loop).
	if api.calls != 3 {
		t.Errorf("API calls = %d, want 3", api.calls)
	}

	// Cached on the second read.
	if _, err := repo.ActiveLocations(context.Background(), 30); err != nil {
		t.Fatalf("ActiveLocations (cached): %v", err)
	}
	if api.calls != 3 {
		t.Errorf("API calls after cache hit = %d, want 3", api.calls)
	}
}
