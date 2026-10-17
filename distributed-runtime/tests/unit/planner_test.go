package unit

import (
	"testing"

	"github.com/distributed-ai/distributed-runtime/pkg/execution"
)

func TestSingleNodePlanner(t *testing.T) {
	planner := execution.NewExecutionPlanner(nil)

	devices := []execution.DeviceInfo{
		{
			NodeID:      "pc1",
			HostName:    "PC1",
			LANIP:       "192.168.1.10",
			DeviceID:    "CUDA0",
			VRAMTotalMB: 24576,
			VRAMFreeMB:  16384,
		},
	}

	plan, err := planner.BuildExecutionPlan("job-1", "exec-1", "test.gguf", "/path/test.gguf", "", 4096, 32, devices, 50052)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.SplitMode != execution.SplitModeSingleNode {
		t.Errorf("expected SINGLE_NODE split mode, got %s", plan.SplitMode)
	}
	if len(plan.Assignments) != 1 {
		t.Errorf("expected 1 assignment, got %d", len(plan.Assignments))
	}
	if plan.Assignments[0].GPULayersCount != 32 {
		t.Errorf("expected 32 layers assigned to single node, got %d", plan.Assignments[0].GPULayersCount)
	}
}

func TestMultiNodeDistributedPlanner(t *testing.T) {
	planner := execution.NewExecutionPlanner(nil)

	devices := []execution.DeviceInfo{
		{
			NodeID:      "pc1",
			HostName:    "PC1",
			LANIP:       "192.168.1.10",
			DeviceID:    "CUDA0",
			VRAMTotalMB: 8192,
			VRAMFreeMB:  4096,
		},
		{
			NodeID:      "pc2",
			HostName:    "PC2",
			LANIP:       "192.168.1.50",
			DeviceID:    "CUDA0",
			VRAMTotalMB: 8192,
			VRAMFreeMB:  4096,
		},
	}

	plan, err := planner.BuildExecutionPlan("job-2", "exec-2", "big-model.gguf", "/path/big.gguf", "", 6144, 32, devices, 50052)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.SplitMode != execution.SplitModeLayerRPC {
		t.Errorf("expected LAYER_RPC split mode, got %s", plan.SplitMode)
	}
	if len(plan.Assignments) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(plan.Assignments))
	}
	if len(plan.RPCEndpoints) != 1 || plan.RPCEndpoints[0] != "192.168.1.50:50052" {
		t.Errorf("expected RPC endpoint 192.168.1.50:50052, got %v", plan.RPCEndpoints)
	}
}

func TestInsufficientVRAMPlanner(t *testing.T) {
	planner := execution.NewExecutionPlanner(nil)

	devices := []execution.DeviceInfo{
		{
			NodeID:      "pc1",
			HostName:    "PC1",
			LANIP:       "192.168.1.10",
			DeviceID:    "CUDA0",
			VRAMTotalMB: 4096,
			VRAMFreeMB:  1024,
		},
	}

	_, err := planner.BuildExecutionPlan("job-3", "exec-3", "huge.gguf", "/path/huge.gguf", "", 16384, 60, devices, 50052)
	if err == nil {
		t.Fatalf("expected error for insufficient VRAM capacity, got nil")
	}
}
