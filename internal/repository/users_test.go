package repository

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
)

// fakeAPI records calls and serves canned users per path.
type fakeAPI struct {
	calls     int
	users     map[string]*models.User
	summaries []models.UserSummary
	lastQuery url.Values
	err       error
}

func (f *fakeAPI) Get(_ context.Context, path string, query url.Values, out any) error {
	f.calls++
	f.lastQuery = query
	if f.err != nil {
		return f.err
	}

	switch target := out.(type) {
	case *models.User:
		user, ok := f.users[path]
		if !ok {
			return errors.New("unexpected path: " + path)
		}
		*target = *user
	case *[]models.UserSummary:
		*target = f.summaries
	default:
		return errors.New("unexpected out type")
	}
	return nil
}

// memCache is an in-memory KVCache for tests.
type memCache struct {
	data map[string][]byte
}

func newMemCache() *memCache {
	return &memCache{data: map[string][]byte{}}
}

func (m *memCache) Get(key string) ([]byte, bool, error) {
	value, ok := m.data[key]
	return value, ok, nil
}

func (m *memCache) Set(key string, value []byte, _ time.Duration) error {
	m.data[key] = value
	return nil
}

func TestUsersRepository_Me_CachesResult(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{users: map[string]*models.User{
		"/me": {ID: 1, Login: "jdiniz", Wallet: 50},
	}}
	repo := NewUsersRepository(api, newMemCache())

	for range 3 {
		user, err := repo.Me(context.Background())
		if err != nil {
			t.Fatalf("Me: %v", err)
		}
		if user.Login != "jdiniz" {
			t.Fatalf("Login = %q, want jdiniz", user.Login)
		}
	}

	if api.calls != 1 {
		t.Errorf("API calls = %d, want 1 (read-through cache)", api.calls)
	}
}

func TestUsersRepository_ByLogin(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{users: map[string]*models.User{
		"/users/xlogin": {ID: 2, Login: "xlogin"},
	}}
	repo := NewUsersRepository(api, newMemCache())

	user, err := repo.ByLogin(context.Background(), "xlogin")
	if err != nil {
		t.Fatalf("ByLogin: %v", err)
	}
	if user.ID != 2 {
		t.Errorf("ID = %d, want 2", user.ID)
	}
}

func TestUsersRepository_APIErrorPropagates(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	repo := NewUsersRepository(&fakeAPI{err: wantErr}, newMemCache())

	if _, err := repo.Me(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestUsersRepository_SearchByLoginPrefix(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{summaries: []models.UserSummary{
		{ID: 1, Login: "malima"},
		{ID: 2, Login: "malima-m"}, // logins com hífen devem entrar no resultado
		{ID: 3, Login: "malimb"},   // fora do prefixo: descartado no filtro local
	}}
	repo := NewUsersRepository(api, newMemCache())

	results, err := repo.SearchByLoginPrefix(context.Background(), "malima", 10)
	if err != nil {
		t.Fatalf("SearchByLoginPrefix: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2 (malima, malima-m)", len(results))
	}
	if results[0].Login != "malima" || results[1].Login != "malima-m" {
		t.Errorf("results = %q, %q; want malima, malima-m", results[0].Login, results[1].Login)
	}

	wantRange := "malima,malima" + strings.Repeat("z", searchUpperPad)
	if got := api.lastQuery.Get("range[login]"); got != wantRange {
		t.Errorf("range[login] = %q, want %q", got, wantRange)
	}
	if got := api.lastQuery.Get("page[size]"); got != "10" {
		t.Errorf("page[size] = %q, want 10", got)
	}

	// Second identical search must hit the cache.
	if _, err := repo.SearchByLoginPrefix(context.Background(), "malima", 10); err != nil {
		t.Fatalf("SearchByLoginPrefix (cached): %v", err)
	}
	if api.calls != 1 {
		t.Errorf("API calls = %d, want 1", api.calls)
	}
}

func TestUsersRepository_NoopCacheAlwaysFetches(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{users: map[string]*models.User{
		"/me": {ID: 1, Login: "jdiniz"},
	}}
	repo := NewUsersRepository(api, NoopCache{})

	for range 2 {
		if _, err := repo.Me(context.Background()); err != nil {
			t.Fatalf("Me: %v", err)
		}
	}
	if api.calls != 2 {
		t.Errorf("API calls = %d, want 2 (noop cache)", api.calls)
	}
}
