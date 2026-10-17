package execution

import (
	"encoding/json"
	"time"
)

type SplitMode string

const (
	SplitModeSingleNode SplitMode = "SINGLE_NODE"
	SplitModeLayerRPC   SplitMode = "LAYER_RPC"
	SplitModeTensorRPC  SplitMode = "TENSOR_RPC"
)

type NodeAssignment struct {
	NodeID          string   `json:"node_id"`
	HostName        string   `json:"hostname"`
	Role            string   `json:"role"` // "LEAD", "RPC_WORKER"
	DeviceID        string   `json:"device_id"`
	RPCEndpoint     string   `json:"rpc_endpoint"`
	MemoryBudgetMB  uint64   `json:"memory_budget_mb"`
	ModelFraction   float64  `json:"model_fraction"`
	GPULayersCount  int      `json:"gpu_layers_count"`
	AssignedDevices []string `json:"assigned_devices"`
}

type ExecutionPlan struct {
	ExecutionID         string           `json:"execution_id"`
	JobID               string           `json:"job_id"`
	ModelID             string           `json:"model_id"`
	ModelVersion        string           `json:"model_version"`
	ModelPath           string           `json:"model_path"`
	ModelSHA256         string           `json:"model_sha256"`
	SplitMode           SplitMode        `json:"split_mode"`
	TensorSplit         []float64        `json:"tensor_split"`
	RPCEndpoints        []string         `json:"rpc_endpoints"`
	Assignments         []NodeAssignment `json:"assignments"`
	ContextLength       int              `json:"context_length"`
	BatchSize           int              `json:"batch_size"`
	MaxTokens           int              `json:"max_tokens"`
	NetworkRequirements string           `json:"network_requirements"`
	CreatedAt           time.Time        `json:"created_at"`
}

func (ep *ExecutionPlan) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(ep, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
