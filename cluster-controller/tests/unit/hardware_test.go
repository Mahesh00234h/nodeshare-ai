package unit

import (
	"testing"

	"github.com/distributed-ai/node-agent/internal/hardware"
)

func TestHardwareDetection(t *testing.T) {
	report, err := hardware.DetectAll("1.0.0")
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	if report.OS == "" {
		t.Errorf("Expected OS string to be non-empty")
	}
	if report.CPU == nil || report.CPU.Cores <= 0 {
		t.Errorf("Expected CPU cores > 0, got %v", report.CPU)
	}
	if report.Memory == nil || report.Memory.TotalBytes == 0 {
		t.Errorf("Expected Total RAM > 0, got %v", report.Memory)
	}
	if report.Storage == nil || report.Storage.TotalBytes == 0 {
		t.Errorf("Expected Total Storage > 0, got %v", report.Storage)
	}

	// Verify GPU detection does not panic or crash
	if report.GPUReport == nil {
		t.Errorf("GPUReport should not be nil")
	}
}
