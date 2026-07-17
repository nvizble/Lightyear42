// Package repository abstracts access to the 42 Intra API, adding a
// read-through cache with per-resource TTLs on top of the HTTP client.
package repository

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/joaodiniz/42cli/internal/models"
)

// Cache TTLs per resource. "Me" changes often (wallet, location),
// other profiles are safe to keep longer.
const (
	meTTL     = 2 * time.Minute
	userTTL   = 10 * time.Minute
	searchTTL = 2 * time.Minute
)

// APIGetter is the minimal API client surface the repositories need.
// Implemented by *api.Client.
type APIGetter interface {
	Get(ctx context.Context, path string, query url.Values, out any) error
}

// Users reads user profiles from the 42 API.
type Users interface {
	// Me returns the profile of the authenticated user.
	Me(ctx context.Context) (*models.User, error)
	// ByLogin returns the profile of the given login.
	ByLogin(ctx context.Context, login string) (*models.User, error)
	// SearchByLoginPrefix lists users whose login starts with prefix.
	SearchByLoginPrefix(ctx context.Context, prefix string, limit int) ([]models.UserSummary, error)
}

// UsersRepository implements Users over the API client with read-through caching.
type UsersRepository struct {
	api   APIGetter
	cache KVCache
}

// NewUsersRepository wires the API client and cache.
// Pass a NoopCache to disable caching.
func NewUsersRepository(client APIGetter, cache KVCache) *UsersRepository {
	return &UsersRepository{api: client, cache: cache}
}

// Me returns the authenticated user's profile.
func (r *UsersRepository) Me(ctx context.Context) (*models.User, error) {
	return fetchCached[*models.User](ctx, r.cache, cacheKey("users", "me"), meTTL, func(ctx context.Context) (*models.User, error) {
		var user models.User
		if err := r.api.Get(ctx, "/me", nil, &user); err != nil {
			return nil, err
		}
		return &user, nil
	})
}

// ByLogin returns the profile of another user by login.
func (r *UsersRepository) ByLogin(ctx context.Context, login string) (*models.User, error) {
	key := cacheKey("users", "login", login)
	return fetchCached[*models.User](ctx, r.cache, key, userTTL, func(ctx context.Context) (*models.User, error) {
		var user models.User
		if err := r.api.Get(ctx, "/users/"+url.PathEscape(login), nil, &user); err != nil {
			return nil, err
		}
		return &user, nil
	})
}

// searchUpperPad is appended to the prefix to form the upper bound of the
// login range. Logins only contain [a-z0-9-], so a run of "z" longer than any
// login sorts after every login with the prefix — in byte order and in the
// linguistic collation used by the Intra database (which ignores hyphens,
// making a "~" bound sort *before* logins like "malima-m").
const searchUpperPad = 20

// SearchByLoginPrefix lists users whose login starts with prefix, sorted by
// login, up to limit results.
//
// The 42 API has no public fuzzy search; the community-standard approach is a
// range filter on login covering every login that starts with the prefix.
// Results are additionally filtered client-side because collation quirks may
// let near-matches (e.g. hyphen variations) slip into the range.
func (r *UsersRepository) SearchByLoginPrefix(ctx context.Context, prefix string, limit int) ([]models.UserSummary, error) {
	key := cacheKey("users", "search", prefix, strconv.Itoa(limit))
	return fetchCached[[]models.UserSummary](ctx, r.cache, key, searchTTL, func(ctx context.Context) ([]models.UserSummary, error) {
		upper := prefix + strings.Repeat("z", searchUpperPad)
		query := url.Values{
			"range[login]": {prefix + "," + upper},
			"sort":         {"login"},
			"page[size]":   {strconv.Itoa(limit)},
		}

		var users []models.UserSummary
		if err := r.api.Get(ctx, "/users", query, &users); err != nil {
			return nil, err
		}

		matches := users[:0]
		for _, user := range users {
			if strings.HasPrefix(user.Login, prefix) {
				matches = append(matches, user)
			}
		}
		return matches, nil
	})
}
