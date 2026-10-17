package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LocalStateData struct {
	LastState          string            `json:"last_state"`
	LastConnectedAt    time.Time         `json:"last_connected_at"`
	LastControllerURL  string            `json:"last_controller_url"`
	ActiveJobIDs       []string          `json:"active_job_ids"`
	InstalledModels    map[string]string `json:"installed_models"` // model_id -> version/checksum
	RecoveryMetadata   map[string]string `json:"recovery_metadata"`
}

type StateStore struct {
	mu       sync.RWMutex
	filePath string
	Data     LocalStateData
}

func NewStateStore(stateDir string) (*StateStore, error) {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, err
	}

	fp := filepath.Join(stateDir, "node_state.json")
	store := &StateStore{
		filePath: fp,
		Data: LocalStateData{
			InstalledModels:  make(map[string]string),
			RecoveryMetadata: make(map[string]string),
		},
	}

	_ = store.load()
	return store, nil
}

func (s *StateStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.Data)
}

func (s *StateStore) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.Data, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

func (s *StateStore) UpdateState(stateStr string, controllerURL string) error {
	s.mu.Lock()
	s.Data.LastState = stateStr
	s.Data.LastControllerURL = controllerURL
	if stateStr == "READY" || stateStr == "BUSY" {
		s.Data.LastConnectedAt = time.Now()
	}
	s.mu.Unlock()
	return s.Save()
}
