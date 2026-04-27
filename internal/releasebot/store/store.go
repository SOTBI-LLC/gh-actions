package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
)

type Store struct {
	mu       sync.RWMutex
	releases map[string]domain.BuildNotification
}

func New() *Store {
	return &Store{
		releases: make(map[string]domain.BuildNotification),
	}
}

func (s *Store) Create(notification domain.BuildNotification) (string, error) {
	releaseID, err := generateReleaseID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.releases[releaseID] = notification

	return releaseID, nil
}

func (s *Store) Get(releaseID string) (domain.BuildNotification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notification, ok := s.releases[releaseID]

	return notification, ok
}

func generateReleaseID() (string, error) {
	var bytes [6]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
