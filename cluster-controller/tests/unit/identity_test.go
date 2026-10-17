package unit

import (
	"os"
	"testing"

	"github.com/distributed-ai/node-agent/internal/identity"
)

func TestNodeIdentityPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "node_identity_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Step 1: Generate identity
	id1, err := identity.LoadOrCreateIdentity(tempDir)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity failed: %v", err)
	}
	if id1.NodeID == "" {
		t.Fatalf("Expected non-empty NodeID")
	}

	// Step 2: Reload identity and verify UUID remains identical
	id2, err := identity.LoadOrCreateIdentity(tempDir)
	if err != nil {
		t.Fatalf("Second LoadOrCreateIdentity failed: %v", err)
	}
	if id1.NodeID != id2.NodeID {
		t.Fatalf("NodeID changed across reloads! Got %s, expected %s", id2.NodeID, id1.NodeID)
	}

	// Step 3: Test credential storage & reload
	creds := &identity.NodeCredentials{
		NodeID:     id1.NodeID,
		AuthToken:  "test-auth-token-123",
		NodeSecret: "test-secret-456",
		EnrolledAt: "2026-01-01T00:00:00Z",
	}
	if err := identity.SaveCredentials(tempDir, creds); err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	loadedCreds, err := identity.LoadCredentials(tempDir)
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}
	if loadedCreds.AuthToken != creds.AuthToken {
		t.Fatalf("AuthToken mismatch! Got %s, expected %s", loadedCreds.AuthToken, creds.AuthToken)
	}
}
