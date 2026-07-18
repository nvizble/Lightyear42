package repository

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nvizble/Lightyear42/internal/models"
)

const (
	projectTTL     = 30 * time.Minute
	attachmentTTL  = 30 * time.Minute
	projectsPageSz = 30
)

// Projects reads pedagogic projects and their attachments from the 42 API.
type Projects interface {
	// BySlugOrName finds a project by slug (exact) or name (case-insensitive).
	BySlugOrName(ctx context.Context, query string) (*models.Project, error)
	// Attachments lists attachments of a project.
	Attachments(ctx context.Context, projectID int) ([]models.Attachment, error)
	// SessionAttachments lists attachments of a project session.
	SessionAttachments(ctx context.Context, sessionID int) ([]models.Attachment, error)
	// Sessions lists project sessions for a project.
	Sessions(ctx context.Context, projectID int) ([]models.ProjectSession, error)
}

// ProjectsRepository implements Projects with read-through caching.
type ProjectsRepository struct {
	api   APIGetter
	cache KVCache
}

// NewProjectsRepository wires the API client and cache.
func NewProjectsRepository(client APIGetter, cache KVCache) *ProjectsRepository {
	return &ProjectsRepository{api: client, cache: cache}
}

// BySlugOrName tries GET /projects/:slug first, then a name filter search.
func (r *ProjectsRepository) BySlugOrName(ctx context.Context, query string) (*models.Project, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("nome do projeto obrigatório")
	}

	slugKey := strings.ToLower(query)
	key := cacheKey("project", "slug", slugKey)
	return fetchCached[*models.Project](ctx, r.cache, key, projectTTL, func(ctx context.Context) (*models.Project, error) {
		var bySlug models.Project
		if err := r.api.Get(ctx, "/projects/"+url.PathEscape(query), nil, &bySlug); err == nil && bySlug.ID != 0 {
			return &bySlug, nil
		}

		q := url.Values{
			"filter[name]": {query},
			"page[size]":   {strconv.Itoa(projectsPageSz)},
		}
		var list []models.Project
		if err := r.api.Get(ctx, "/projects", q, &list); err != nil {
			return nil, err
		}
		if p := matchProject(list, query); p != nil {
			return p, nil
		}

		// Broader search: slug filter
		q = url.Values{
			"filter[slug]": {slugKey},
			"page[size]":   {strconv.Itoa(projectsPageSz)},
		}
		list = nil
		if err := r.api.Get(ctx, "/projects", q, &list); err != nil {
			return nil, err
		}
		if p := matchProject(list, query); p != nil {
			return p, nil
		}
		return nil, fmt.Errorf("projeto %q não encontrado", query)
	})
}

func matchProject(list []models.Project, query string) *models.Project {
	lower := strings.ToLower(query)
	for i := range list {
		if strings.EqualFold(list[i].Slug, query) || strings.EqualFold(list[i].Name, query) {
			p := list[i]
			return &p
		}
	}
	for i := range list {
		if strings.Contains(strings.ToLower(list[i].Slug), lower) ||
			strings.Contains(strings.ToLower(list[i].Name), lower) {
			p := list[i]
			return &p
		}
	}
	if len(list) == 1 {
		p := list[0]
		return &p
	}
	return nil
}

// Attachments returns project-level attachments.
func (r *ProjectsRepository) Attachments(ctx context.Context, projectID int) ([]models.Attachment, error) {
	id := strconv.Itoa(projectID)
	key := cacheKey("project", id, "attachments")
	return fetchCached[[]models.Attachment](ctx, r.cache, key, attachmentTTL, func(ctx context.Context) ([]models.Attachment, error) {
		var out []models.Attachment
		if err := r.api.Get(ctx, "/projects/"+id+"/attachments", nil, &out); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// Sessions returns all sessions of a project.
func (r *ProjectsRepository) Sessions(ctx context.Context, projectID int) ([]models.ProjectSession, error) {
	id := strconv.Itoa(projectID)
	key := cacheKey("project", id, "sessions")
	return fetchCached[[]models.ProjectSession](ctx, r.cache, key, projectTTL, func(ctx context.Context) ([]models.ProjectSession, error) {
		var out []models.ProjectSession
		if err := r.api.Get(ctx, "/projects/"+id+"/project_sessions", nil, &out); err != nil {
			return nil, err
		}
		return out, nil
	})
}

// SessionAttachments returns attachments of one project session.
func (r *ProjectsRepository) SessionAttachments(ctx context.Context, sessionID int) ([]models.Attachment, error) {
	id := strconv.Itoa(sessionID)
	key := cacheKey("project_session", id, "attachments")
	return fetchCached[[]models.Attachment](ctx, r.cache, key, attachmentTTL, func(ctx context.Context) ([]models.Attachment, error) {
		var out []models.Attachment
		if err := r.api.Get(ctx, "/project_sessions/"+id+"/attachments", nil, &out); err != nil {
			return nil, err
		}
		return out, nil
	})
}
