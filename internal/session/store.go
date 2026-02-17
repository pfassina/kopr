package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Store handles session state persistence.
type Store struct {
	path string
}

// NewStore creates a store that persists to the given vault directory.
func NewStore(vaultPath string) *Store {
	return &Store{
		path: filepath.Join(vaultPath, ".kopr", "state.json"),
	}
}

// Load reads the session state from disk.
func (s *Store) Load() (State, error) {
	state := Default()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return Default(), err
	}

	return state, nil
}

// Save writes the session state to disk.
func (s *Store) Save(state State) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}
