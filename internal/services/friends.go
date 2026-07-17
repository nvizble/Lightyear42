package services

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nvizble/Lightyear42/internal/models"
)

// FriendsStore persists the local friends list.
// Implemented by *config.FriendsFile.
type FriendsStore interface {
	Load() ([]string, error)
	Save([]string) error
}

// FriendsService manages the user's local friends list.
// Friends are a CLI-local concept: the public 42 API has no friendship data.
type FriendsService struct {
	store FriendsStore
}

// NewFriendsService wires the friends store.
func NewFriendsService(store FriendsStore) *FriendsService {
	return &FriendsService{store: store}
}

// List returns the friends sorted alphabetically.
func (s *FriendsService) List() ([]string, error) {
	friends, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	slices.Sort(friends)
	return friends, nil
}

// Add inserts a login into the list. Returns false when it was already there.
func (s *FriendsService) Add(login string) (bool, error) {
	login, err := normalizeLogin(login)
	if err != nil {
		return false, err
	}

	friends, err := s.store.Load()
	if err != nil {
		return false, err
	}
	if slices.Contains(friends, login) {
		return false, nil
	}

	friends = append(friends, login)
	slices.Sort(friends)
	return true, s.store.Save(friends)
}

// Remove deletes a login from the list. Returns false when it was absent.
func (s *FriendsService) Remove(login string) (bool, error) {
	login, err := normalizeLogin(login)
	if err != nil {
		return false, err
	}

	friends, err := s.store.Load()
	if err != nil {
		return false, err
	}

	index := slices.Index(friends, login)
	if index < 0 {
		return false, nil
	}

	friends = slices.Delete(friends, index, index+1)
	return true, s.store.Save(friends)
}

// normalizeLogin lowercases and trims a login, rejecting blanks.
func normalizeLogin(login string) (string, error) {
	login = strings.TrimSpace(strings.ToLower(login))
	if login == "" {
		return "", fmt.Errorf("%w", ErrEmptyQuery)
	}
	return login, nil
}

// FilterLocationsByLogin keeps only locations whose user is in logins.
func FilterLocationsByLogin(locations []models.Location, logins []string) []models.Location {
	wanted := make(map[string]bool, len(logins))
	for _, login := range logins {
		wanted[login] = true
	}

	var filtered []models.Location
	for _, loc := range locations {
		if wanted[loc.User.Login] {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}
