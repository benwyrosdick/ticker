package snaptrade

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

// Store persists per-user SnapTrade secrets to disk (via afero, so it is
// testable with an in-memory filesystem). The secret is returned by SnapTrade
// at user registration and is required for all subsequent requests.
type Store struct {
	fs       afero.Fs
	filePath string
}

type storeFile struct {
	Users map[string]storeUser `json:"users"`
}

type storeUser struct {
	UserSecret string `json:"user_secret"`
}

// NewStore returns a Store backed by fs, writing to <dataHome>/ticker/snaptrade.json.
func NewStore(fs afero.Fs, dataHome string) *Store {
	return &Store{
		fs:       fs,
		filePath: filepath.Join(dataHome, "ticker", "snaptrade.json"),
	}
}

// GetUserSecret returns the persisted secret for userID. ok is false when no
// secret has been stored yet (i.e. the user has not connected).
func (s *Store) GetUserSecret(userID string) (string, bool, error) {
	file, err := s.read()
	if err != nil {
		return "", false, err
	}

	user, ok := file.Users[userID]
	if !ok || user.UserSecret == "" {
		return "", false, nil
	}

	return user.UserSecret, true, nil
}

// SaveUserSecret persists the secret for userID, preserving any other users.
func (s *Store) SaveUserSecret(userID, userSecret string) error {
	file, err := s.read()
	if err != nil {
		return err
	}

	file.Users[userID] = storeUser{UserSecret: userSecret}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	if err := s.fs.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return fmt.Errorf("snaptrade: create data dir: %w", err)
	}

	return afero.WriteFile(s.fs, s.filePath, data, 0o600)
}

func (s *Store) read() (storeFile, error) {
	exists, err := afero.Exists(s.fs, s.filePath)
	if err != nil {
		return storeFile{}, err
	}

	if !exists {
		return storeFile{Users: map[string]storeUser{}}, nil
	}

	data, err := afero.ReadFile(s.fs, s.filePath)
	if err != nil {
		return storeFile{}, err
	}

	var file storeFile
	if err := json.Unmarshal(data, &file); err != nil {
		return storeFile{}, fmt.Errorf("snaptrade: parse %s: %w", s.filePath, err)
	}

	if file.Users == nil {
		file.Users = map[string]storeUser{}
	}

	return file, nil
}
