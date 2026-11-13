package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Device struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"tokenHash,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	LastSeen  time.Time `json:"lastSeen"`
}

type Store struct {
	mu      sync.RWMutex
	path    string
	devices []Device
}

func OpenStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	store := &Store{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			store.devices = []Device{}
			return store, nil
		}
		return nil, fmt.Errorf("reading device store: %w", err)
	}

	if err := json.Unmarshal(data, &store.devices); err != nil {
		return nil, fmt.Errorf("parsing device store %s: %w", path, err)
	}

	if store.devices == nil {
		store.devices = []Device{}
	}

	return store, nil
}

func (s *Store) Pair(name string) (Device, string, error) {
	token, hash := NewToken()

	device := Device{
		ID:        newID(),
		Name:      name,
		TokenHash: hash,
		CreatedAt: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.devices = append(s.devices, device)

	if err := s.persist(s.devices); err != nil {
		s.devices = s.devices[:len(s.devices)-1]
		return Device{}, "", err
	}

	return device, token, nil
}

func (s *Store) Verify(token string) (Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := HashToken(token)

	for i, device := range s.devices {
		if subtle.ConstantTimeCompare(
			[]byte(hash),
			[]byte(device.TokenHash),
		) == 1 {
			s.devices[i].LastSeen = time.Now().UTC()
			return s.devices[i], true
		}
	}

	return Device{}, false
}

func (s *Store) List() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Device, len(s.devices))
	for i, device := range s.devices {
		list[i] = device
		list[i].TokenHash = ""
	}

	return list
}

func (s *Store) Revoke(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, device := range s.devices {
		if device.ID != id {
			continue
		}

		remaining := make([]Device, 0, len(s.devices)-1)
		remaining = append(remaining, s.devices[:i]...)
		remaining = append(remaining, s.devices[i+1:]...)

		if err := s.persist(remaining); err != nil {
			return false
		}

		s.devices = remaining
		return true
	}

	return false
}

func (s *Store) persist(devices []Device) error {
	data, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path, data, 0o600)
}

func newID() string {
	raw := make([]byte, 4)

	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}

	return hex.EncodeToString(raw)
}
