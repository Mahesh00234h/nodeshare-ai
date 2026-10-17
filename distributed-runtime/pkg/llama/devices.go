package llama

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type LocalCUDADevice struct {
	Index          int    `json:"index"`
	Name           string `json:"name"`
	TotalVRAMBytes uint64 `json:"total_vram_bytes"`
	FreeVRAMBytes  uint64 `json:"free_vram_bytes"`
}

func DetectLocalCUDADevices() ([]LocalCUDADevice, error) {
	devices := make([]LocalCUDADevice, 0)

	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return []LocalCUDADevice{
			{
				Index:          0,
				Name:           "NVIDIA GeForce RTX 4090 (Mock)",
				TotalVRAMBytes: 24 * 1024 * 1024 * 1024,
				FreeVRAMBytes:  20 * 1024 * 1024 * 1024,
			},
		}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 4 {
			idx, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			name := strings.TrimSpace(parts[1])
			totalMB, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
			freeMB, _ := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 64)

			dev := LocalCUDADevice{
				Index:          idx,
				Name:           name,
				TotalVRAMBytes: totalMB * 1024 * 1024,
				FreeVRAMBytes:  freeMB * 1024 * 1024,
			}
			devices = append(devices, dev)
		}
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no CUDA devices detected on host (%s)", runtime.GOOS)
	}

	return devices, nil
}
