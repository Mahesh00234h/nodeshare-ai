package unit

import (
	"testing"

	"github.com/distributed-ai/distributed-runtime/pkg/communication"
)

func TestLANSecurityGuard(t *testing.T) {
	guard := communication.NewLANSecurityGuard(nil)

	// Valid private IPs
	validHosts := []string{
		"192.168.1.50:50052",
		"10.0.0.12:50052",
		"127.0.0.1:50052",
	}

	for _, host := range validHosts {
		if err := guard.ValidateRPCEndpoint(host); err != nil {
			t.Errorf("expected host %s to pass LAN validation, got: %v", host, err)
		}
	}

	// Invalid public IPs / hostnames
	invalidHosts := []string{
		"8.8.8.8:50052",
		"1.1.1.1:50052",
		"llm-cluster.onrender.com:50052",
	}

	for _, host := range invalidHosts {
		if err := guard.ValidateRPCEndpoint(host); err == nil {
			t.Errorf("expected public endpoint %s to fail LAN validation, but it passed", host)
		}
	}
}
