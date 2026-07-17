package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/joaodiniz/42cli/internal/api"
	"github.com/joaodiniz/42cli/internal/auth"
	"github.com/joaodiniz/42cli/internal/cache"
	"github.com/joaodiniz/42cli/internal/repository"
	"github.com/joaodiniz/42cli/internal/services"
)

// appDeps bundles the services available to data commands.
type appDeps struct {
	Users  *services.UserService
	Campus *services.CampusService
	Slots  *services.SlotsService
}

// newDeps is the composition root for data commands: it wires
// config → token source → API client → cache → repositories → services.
//
// The returned cleanup closes the cache store and must be called when the
// command finishes. Requires an active session (auth.ErrNoToken otherwise).
func newDeps(ctx context.Context) (*appDeps, func(), error) {
	source, err := auth.NewTokenSource(ctx, rootCfg, auth.NewKeyringStore())
	if err != nil {
		return nil, nil, err
	}

	client := api.NewClient(rootCfg.APIBaseURL, source)

	// A broken local cache must not block API access: fall back to no caching.
	var kv repository.KVCache = repository.NoopCache{}
	cleanup := func() {}
	if dbPath, err := cacheDBPath(); err == nil {
		if store, err := cache.Open(dbPath); err == nil {
			kv = store
			cleanup = func() { _ = store.Close() }
		} else {
			fmt.Fprintln(os.Stderr, "aviso: cache local indisponível, seguindo sem cache:", err)
		}
	}

	usersRepo := repository.NewUsersRepository(client, kv)
	users := services.NewUserService(usersRepo)
	deps := &appDeps{
		Users:  users,
		Campus: services.NewCampusService(repository.NewCampusRepository(client, kv)),
		Slots:  services.NewSlotsService(repository.NewSlotsRepository(client), users),
	}
	return deps, cleanup, nil
}

// primaryCampusID resolves the authenticated user's primary campus.
func primaryCampusID(ctx context.Context, deps *appDeps) (int, string, error) {
	me, err := deps.Users.Me(ctx)
	if err != nil {
		return 0, "", err
	}
	campus := me.PrimaryCampus()
	if campus == nil {
		return 0, "", fmt.Errorf("seu perfil não tem campus associado")
	}
	return campus.ID, campus.Name, nil
}
