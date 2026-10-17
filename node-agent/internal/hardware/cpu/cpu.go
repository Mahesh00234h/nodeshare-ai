package cpu

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

type CPUInfo struct {
	Model         string  `json:"model"`
	Vendor        string  `json:"vendor"`
	Cores         int     `json:"cores"`
	Threads       int     `json:"threads"`
	Utilization   float64 `json:"utilization_percent"`
}

func Detect() (*CPUInfo, error) {
	info := &CPUInfo{
		Cores:   runtime.NumCPU(),
		Threads: runtime.NumCPU(),
	}

	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		info.Model = cpuInfos[0].ModelName
		info.Vendor = cpuInfos[0].VendorID
		if cpuInfos[0].Cores > 0 {
			info.Cores = int(cpuInfos[0].Cores)
		}
	} else {
		info.Model = runtime.GOARCH
		info.Vendor = "Generic"
	}

	percent, err := cpu.Percent(100*time.Millisecond, false)
	if err == nil && len(percent) > 0 {
		info.Utilization = percent[0]
	}

	return info, nil
}
