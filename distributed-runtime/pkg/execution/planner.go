package execution

import (
	"fmt"
	"time"
)

type DeviceInfo struct {
	NodeID           string  `json:"node_id"`
	HostName         string  `json:"hostname"`
	LANIP            string  `json:"lan_ip"`
	DeviceID         string  `json:"device_id"`
	DeviceName       string  `json:"device_name"`
	VRAMTotalMB      uint64  `json:"vram_total_mb"`
	VRAMFreeMB       uint64  `json:"vram_free_mb"`
	ComputeScore     float64 `json:"compute_score"`
	NetworkLatencyMS float64 `json:"network_latency_ms"`
}

type DeviceScorer interface {
	ScoreDevice(device *DeviceInfo, modelReqVRAMMB uint64) float64
}

type DefaultDeviceScorer struct{}

func (ds *DefaultDeviceScorer) ScoreDevice(device *DeviceInfo, modelReqVRAMMB uint64) float64 {
	if device.VRAMFreeMB == 0 {
		return 0.0
	}
	vramScore := float64(device.VRAMFreeMB) / 1024.0 // Score in GB
	latencyPenalty := device.NetworkLatencyMS * 0.1
	score := vramScore + device.ComputeScore - latencyPenalty
	if score < 0 {
		return 0.1
	}
	return score
}

type ExecutionPlanner struct {
	scorer DeviceScorer
}

func NewExecutionPlanner(scorer DeviceScorer) *ExecutionPlanner {
	if scorer == nil {
		scorer = &DefaultDeviceScorer{}
	}
	return &ExecutionPlanner{scorer: scorer}
}

func (ep *ExecutionPlanner) BuildExecutionPlan(
	jobID string,
	executionID string,
	modelID string,
	modelPath string,
	modelSHA256 string,
	modelVRAMMB uint64,
	totalModelLayers int,
	availableDevices []DeviceInfo,
	rpcPort int,
) (*ExecutionPlan, error) {
	if len(availableDevices) == 0 {
		return nil, fmt.Errorf("INSUFFICIENT_CLUSTER_CAPACITY: no available devices provided")
	}

	plan := &ExecutionPlan{
		ExecutionID:   executionID,
		JobID:         jobID,
		ModelID:       modelID,
		ModelPath:     modelPath,
		ModelSHA256:   modelSHA256,
		ContextLength: 4096,
		BatchSize:     512,
		MaxTokens:     512,
		CreatedAt:     time.Now(),
		Assignments:   make([]NodeAssignment, 0),
		RPCEndpoints:  make([]string, 0),
		TensorSplit:   make([]float64, 0),
	}

	var totalFreeVRAM uint64
	for _, dev := range availableDevices {
		totalFreeVRAM += dev.VRAMFreeMB
	}

	if totalFreeVRAM < modelVRAMMB {
		return nil, fmt.Errorf("INSUFFICIENT_CLUSTER_CAPACITY: total cluster free VRAM (%d MB) is less than required model VRAM (%d MB)", totalFreeVRAM, modelVRAMMB)
	}

	if len(availableDevices) == 1 || availableDevices[0].VRAMFreeMB >= modelVRAMMB {
		plan.SplitMode = SplitModeSingleNode
		leadDev := availableDevices[0]
		assign := NodeAssignment{
			NodeID:          leadDev.NodeID,
			HostName:        leadDev.HostName,
			Role:            "LEAD",
			DeviceID:        leadDev.DeviceID,
			RPCEndpoint:     "",
			MemoryBudgetMB:  modelVRAMMB,
			ModelFraction:   1.0,
			GPULayersCount:  totalModelLayers,
			AssignedDevices: []string{leadDev.DeviceID},
		}
		plan.Assignments = append(plan.Assignments, assign)
		plan.TensorSplit = []float64{1.0}
		return plan, nil
	}

	plan.SplitMode = SplitModeLayerRPC
	var runningVRAM uint64
	participatingDevices := make([]DeviceInfo, 0)

	for _, dev := range availableDevices {
		participatingDevices = append(participatingDevices, dev)
		runningVRAM += dev.VRAMFreeMB
		if runningVRAM >= modelVRAMMB {
			break
		}
	}

	if runningVRAM < modelVRAMMB {
		return nil, fmt.Errorf("INSUFFICIENT_CLUSTER_CAPACITY: selected devices free VRAM (%d MB) insufficient for model VRAM (%d MB)", runningVRAM, modelVRAMMB)
	}

	totalSelectedVRAM := float64(runningVRAM)
	remainingLayers := totalModelLayers

	for i, dev := range participatingDevices {
		fraction := float64(dev.VRAMFreeMB) / totalSelectedVRAM
		plan.TensorSplit = append(plan.TensorSplit, fraction)

		layersForNode := int(float64(totalModelLayers) * fraction)
		if i == len(participatingDevices)-1 {
			layersForNode = remainingLayers
		}
		remainingLayers -= layersForNode

		role := "LEAD"
		rpcEndpoint := ""

		if i > 0 {
			role = "RPC_WORKER"
			port := rpcPort
			if port == 0 {
				port = 50052
			}
			rpcEndpoint = fmt.Sprintf("%s:%d", dev.LANIP, port)
			plan.RPCEndpoints = append(plan.RPCEndpoints, rpcEndpoint)
		}

		assign := NodeAssignment{
			NodeID:          dev.NodeID,
			HostName:        dev.HostName,
			Role:            role,
			DeviceID:        dev.DeviceID,
			RPCEndpoint:     rpcEndpoint,
			MemoryBudgetMB:  uint64(float64(modelVRAMMB) * fraction),
			ModelFraction:   fraction,
			GPULayersCount:  layersForNode,
			AssignedDevices: []string{dev.DeviceID},
		}
		plan.Assignments = append(plan.Assignments, assign)
	}

	return plan, nil
}
