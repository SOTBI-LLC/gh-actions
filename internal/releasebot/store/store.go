package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/SOTBI-LLC/gh-actions/internal/releasebot/domain"
)

type releaseEntry struct {
	notification  domain.BuildNotification
	devDeployed   bool
	prodDeployed  bool
}

type Store struct {
	mu       sync.RWMutex
	releases map[string]releaseEntry
}

func New() *Store {
	return &Store{
		releases: make(map[string]releaseEntry),
	}
}

func (s *Store) Create(notification domain.BuildNotification) (string, error) {
	releaseID, err := generateReleaseID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.releases[releaseID] = releaseEntry{notification: notification}

	return releaseID, nil
}

func (s *Store) Get(releaseID string) (domain.BuildNotification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.releases[releaseID]

	return entry.notification, ok
}

// MarkDeployed records a successful workflow dispatch for the given environment.
func (s *Store) MarkDeployed(releaseID, environment string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.releases[releaseID]
	if !ok {
		return false
	}

	switch environment {
	case domain.EnvironmentDev:
		entry.devDeployed = true
	case domain.EnvironmentProd:
		entry.prodDeployed = true
	default:
		return false
	}

	s.releases[releaseID] = entry

	return true
}

// DeploymentStatus reports which environments have already been deployed for a release.
func (s *Store) DeploymentStatus(releaseID string) (devDone, prodDone, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, found := s.releases[releaseID]
	if !found {
		return false, false, false
	}

	return entry.devDeployed, entry.prodDeployed, true
}

func generateReleaseID() (string, error) {
	var bytes [6]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
