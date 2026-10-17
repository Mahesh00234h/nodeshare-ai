package hardware

import (
	"os"
	"runtime"

	"github.com/distributed-ai/node-agent/internal/hardware/cpu"
	"github.com/distributed-ai/node-agent/internal/hardware/gpu"
	"github.com/distributed-ai/node-agent/internal/hardware/memory"
	"github.com/distributed-ai/node-agent/internal/hardware/network"
	"github.com/distributed-ai/node-agent/internal/hardware/storage"
)

type FullHardwareReport struct {
	Hostname             string               `json:"hostname"`
	OS                   string               `json:"os"`
	OSVersion            string               `json:"os_version"`
	Architecture         string               `json:"architecture"`
	CPU                  *cpu.CPUInfo         `json:"cpu"`
	Memory               *memory.MemoryInfo   `json:"memory"`
	GPUReport            *gpu.GPUReport       `json:"gpu_report"`
	Storage              *storage.StorageInfo `json:"storage"`
	Network              *network.NetworkInfo `json:"network"`
	AgentVersion         string               `json:"agent_version"`
	RuntimeCapabilities  []string             `json:"runtime_capabilities"`
}

func DetectAll(agentVersion string) (*FullHardwareReport, error) {
	hostname, _ := os.Hostname()

	cpuInfo, _ := cpu.Detect()
	memInfo, _ := memory.Detect()
	gpuInfo, _ := gpu.Detect()
	storInfo, _ := storage.Detect()
	netInfo, _ := network.Detect()

	caps := []string{
		"CPU_INFERENCE",
		"TEXT_GENERATION",
		"STREAMING",
	}

	if gpuInfo != nil && gpuInfo.GPUAvailable && len(gpuInfo.GPUs) > 0 {
		caps = append(caps, "GPU_INFERENCE", "CUDA")
	}

	return &FullHardwareReport{
		Hostname:            hostname,
		OS:                  runtime.GOOS,
		OSVersion:           runtime.GOOS + " " + runtime.GOARCH,
		Architecture:        runtime.GOARCH,
		CPU:                 cpuInfo,
		Memory:              memInfo,
		GPUReport:           gpuInfo,
		Storage:             storInfo,
		Network:             netInfo,
		AgentVersion:        agentVersion,
		RuntimeCapabilities: caps,
	}, nil
}
