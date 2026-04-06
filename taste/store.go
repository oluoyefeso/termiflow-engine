package taste

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ProfileStore persists taste profiles.
// Load returns (nil, nil) if no profile exists for the given userID.
type ProfileStore interface {
	Save(ctx context.Context, profile *TasteProfile) error
	Load(ctx context.Context, userID string) (*TasteProfile, error)
	Delete(ctx context.Context, userID string) error
}

// InMemoryProfileStore is a ProfileStore backed by an in-memory map.
// Suitable for testing and short-lived sessions.
type InMemoryProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]*TasteProfile
}

// NewInMemoryProfileStore creates a new in-memory store.
func NewInMemoryProfileStore() *InMemoryProfileStore {
	return &InMemoryProfileStore{
		profiles: make(map[string]*TasteProfile),
	}
}

func (s *InMemoryProfileStore) Save(_ context.Context, profile *TasteProfile) error {
	if profile == nil {
		return fmt.Errorf("taste: cannot save nil profile")
	}
	if err := validateUserID(profile.UserID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles[profile.UserID] = copyProfile(profile)
	return nil
}

func (s *InMemoryProfileStore) Load(_ context.Context, userID string) (*TasteProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.profiles[userID]
	if !ok {
		return nil, nil
	}
	return copyProfile(p), nil
}

func (s *InMemoryProfileStore) Delete(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.profiles, userID)
	return nil
}

// FileProfileStore is a ProfileStore backed by JSON files on disk.
// One file per user, stored in the given directory.
type FileProfileStore struct {
	dir string
}

// NewFileProfileStore creates a new file-based store.
// The directory is created lazily on the first Save call.
func NewFileProfileStore(dir string) *FileProfileStore {
	return &FileProfileStore{dir: dir}
}

func (s *FileProfileStore) Save(_ context.Context, profile *TasteProfile) error {
	if profile == nil {
		return fmt.Errorf("taste: cannot save nil profile")
	}
	if err := validateUserID(profile.UserID); err != nil {
		return err
	}

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("taste: create directory: %w", err)
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("taste: marshal profile: %w", err)
	}

	path := s.profilePath(profile.UserID)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("taste: write profile: %w", err)
	}
	return nil
}

func (s *FileProfileStore) Load(_ context.Context, userID string) (*TasteProfile, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.profilePath(userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("taste: read profile: %w", err)
	}

	var profile TasteProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("taste: unmarshal profile: %w", err)
	}
	return &profile, nil
}

func (s *FileProfileStore) Delete(_ context.Context, userID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}

	err := os.Remove(s.profilePath(userID))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("taste: delete profile: %w", err)
	}
	return nil
}

func (s *FileProfileStore) profilePath(userID string) string {
	return filepath.Join(s.dir, userID+".json")
}

// validateUserID rejects userIDs that contain path separators or traversal patterns.
func validateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("taste: empty user ID")
	}
	if strings.ContainsAny(userID, "/\\") {
		return fmt.Errorf("taste: user ID contains path separator: %q", userID)
	}
	if strings.Contains(userID, "..") {
		return fmt.Errorf("taste: user ID contains path traversal: %q", userID)
	}
	if strings.ContainsRune(userID, 0) {
		return fmt.Errorf("taste: user ID contains null byte")
	}
	return nil
}

// copyProfile creates a deep copy of a TasteProfile.
func copyProfile(p *TasteProfile) *TasteProfile {
	cp := &TasteProfile{
		Version:     p.Version,
		UserID:      p.UserID,
		SignalCount:  p.SignalCount,
		LastUpdated: p.LastUpdated,
	}
	if p.Affinities != nil {
		cp.Affinities = make(map[string]float64, len(p.Affinities))
		for k, v := range p.Affinities {
			cp.Affinities[k] = v
		}
	}
	return cp
}
