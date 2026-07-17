package services

import (
	"context"
	"testing"

	"github.com/joaodiniz/42cli/internal/models"
)

type fakeCampus struct {
	locations []models.Location
}

func (f *fakeCampus) ActiveLocations(context.Context, int) ([]models.Location, error) {
	return f.locations, nil
}

func TestCampusService_Online_SortsByHost(t *testing.T) {
	t.Parallel()

	svc := NewCampusService(&fakeCampus{locations: []models.Location{
		{Host: "c2r1p1", User: models.UserSummary{Login: "b"}},
		{Host: "c1r1p1", User: models.UserSummary{Login: "a"}},
	}})

	locations, err := svc.Online(context.Background(), 30)
	if err != nil {
		t.Fatalf("Online: %v", err)
	}
	if locations[0].Host != "c1r1p1" || locations[1].Host != "c2r1p1" {
		t.Errorf("hosts = %q, %q; want sorted c1r1p1, c2r1p1", locations[0].Host, locations[1].Host)
	}
}
