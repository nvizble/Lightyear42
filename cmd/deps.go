package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nvizble/Lightyear42/internal/api"
	"github.com/nvizble/Lightyear42/internal/auth"
	"github.com/nvizble/Lightyear42/internal/cache"
	"github.com/nvizble/Lightyear42/internal/repository"
	"github.com/nvizble/Lightyear42/internal/services"
)

// appDeps bundles the services available to data commands.
type appDeps struct {
	Users    *services.UserService
	Campus   *services.CampusService
	Slots    *services.SlotsService
	Subjects *services.SubjectService
}

// depsOptions customizes the composition root (debug HTTP, etc.).
type depsOptions struct {
	HTTPDebug bool
}

// newDeps is the composition root for data commands: it wires
// config → token source → API client → cache → repositories → services.
//
// The returned cleanup closes the cache store and must be called when the
// command finishes. Requires an active session (auth.ErrNoToken otherwise).
func newDeps(ctx context.Context, opts ...depsOptions) (*appDeps, func(), error) {
	var opt depsOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	source, err := auth.NewTokenSource(ctx, rootCfg, auth.NewKeyringStore())
	if err != nil {
		return nil, nil, err
	}

	var apiOpts []api.Option
	if opt.HTTPDebug || httpDebugEnvEnabled() {
		apiOpts = append(apiOpts, api.WithDebugLog(os.Stderr))
	}
	client := api.NewClient(rootCfg.APIBaseURL, source, apiOpts...)

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
	projectsRepo := repository.NewProjectsRepository(client, kv)
	deps := &appDeps{
		Users:    users,
		Campus:   services.NewCampusService(repository.NewCampusRepository(client, kv)),
		Slots:    services.NewSlotsService(repository.NewSlotsRepository(client), users),
		Subjects: services.NewSubjectService(projectsRepo, users, client),
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

func httpDebugEnvEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FORTYTWO_HTTP_DEBUG")))
	return v == "1" || v == "true" || v == "yes"
}
