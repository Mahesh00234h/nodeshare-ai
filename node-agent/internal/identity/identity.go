package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type NodeIdentity struct {
	mu           sync.RWMutex
	NodeID       string `json:"node_id"`
	CreatedAt    string `json:"created_at"`
	IdentityPath string `json:"-"`
}

type NodeCredentials struct {
	NodeID     string `json:"node_id"`
	AuthToken  string `json:"auth_token"`
	NodeSecret string `json:"node_secret,omitempty"`
	EnrolledAt string `json:"enrolled_at"`
}

// LoadOrCreateIdentity attempts to load existing node ID from disk, or generates a fresh persistent UUID.
func LoadOrCreateIdentity(identityDir string) (*NodeIdentity, error) {
	if err := os.MkdirAll(identityDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create identity directory: %w", err)
	}

	idPath := filepath.Join(identityDir, "node_id.json")
	if data, err := os.ReadFile(idPath); err == nil {
		var id NodeIdentity
		if err := json.Unmarshal(data, &id); err == nil && id.NodeID != "" {
			id.IdentityPath = idPath
			return &id, nil
		}
	}

	// Generate new node UUID
	newUUID := uuid.New().String()
	id := &NodeIdentity{
		NodeID:       newUUID,
		CreatedAt:    uuid.New().String(), // random timestamp/marker
		IdentityPath: idPath,
	}

	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal identity: %w", err)
	}

	if err := os.WriteFile(idPath, data, 0600); err != nil {
		return nil, fmt.Errorf("failed to write identity file: %w", err)
	}

	return id, nil
}

// SaveCredentials stores enrolled node credentials securely.
func SaveCredentials(identityDir string, creds *NodeCredentials) error {
	if err := os.MkdirAll(identityDir, 0755); err != nil {
		return err
	}

	credPath := filepath.Join(identityDir, "node_credentials.json")
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(credPath, data, 0600)
}

// LoadCredentials loads saved node credentials.
func LoadCredentials(identityDir string) (*NodeCredentials, error) {
	credPath := filepath.Join(identityDir, "node_credentials.json")
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, err
	}

	var creds NodeCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}
