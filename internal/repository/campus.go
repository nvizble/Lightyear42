package repository

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

// Presence data changes by the minute.
const locationsTTL = time.Minute

// Pagination bounds for the locations listing. locationsMaxPages caps the
// walk on very large campuses so one command never burns the rate limit.
const (
	locationsPageSize = 100
	locationsMaxPages = 10
)

// Campus reads campus-scoped resources from the 42 API.
//
// Note: exam listings were considered here but dropped — every exams
// endpoint returns 403 for tokens with only the public scope.
type Campus interface {
	// ActiveLocations lists everyone currently logged in at the campus.
	ActiveLocations(ctx context.Context, campusID int) ([]models.Location, error)
}

// CampusRepository implements Campus over the API client with caching.
type CampusRepository struct {
	api   APIGetter
	cache KVCache
}

// NewCampusRepository wires the API client and cache.
func NewCampusRepository(client APIGetter, cache KVCache) *CampusRepository {
	return &CampusRepository{api: client, cache: cache}
}

// ActiveLocations walks all pages of active sessions at the campus.
func (r *CampusRepository) ActiveLocations(ctx context.Context, campusID int) ([]models.Location, error) {
	id := strconv.Itoa(campusID)
	key := cacheKey("campus", id, "locations")

	return fetchCached[[]models.Location](ctx, r.cache, key, locationsTTL, func(ctx context.Context) ([]models.Location, error) {
		var all []models.Location
		for page := 1; page <= locationsMaxPages; page++ {
			query := url.Values{
				"filter[active]": {"true"},
				"page[size]":     {strconv.Itoa(locationsPageSize)},
				"page[number]":   {strconv.Itoa(page)},
			}

			var batch []models.Location
			if err := r.api.Get(ctx, "/campus/"+id+"/locations", query, &batch); err != nil {
				return nil, err
			}
			all = append(all, batch...)

			if len(batch) < locationsPageSize {
				break
			}
		}
		return all, nil
	})
}
