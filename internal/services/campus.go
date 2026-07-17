package services

import (
	"context"
	"sort"

	"github.com/nvizble/Lightyear42/internal/models"
	"github.com/nvizble/Lightyear42/internal/repository"
)

// CampusService implements campus-scoped features: online map and exams.
type CampusService struct {
	campus repository.Campus
}

// NewCampusService wires the campus repository.
func NewCampusService(campus repository.Campus) *CampusService {
	return &CampusService{campus: campus}
}

// Online lists everyone logged in at the campus, sorted by host for a
// stable map layout.
func (s *CampusService) Online(ctx context.Context, campusID int) ([]models.Location, error) {
	locations, err := s.campus.ActiveLocations(ctx, campusID)
	if err != nil {
		return nil, err
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Host < locations[j].Host
	})
	return locations, nil
}
