package memory

import (
	"github.com/shirou/gopsutil/v3/mem"
)

type MemoryInfo struct {
	TotalBytes         uint64  `json:"total_bytes"`
	AvailableBytes     uint64  `json:"available_bytes"`
	UsedBytes          uint64  `json:"used_bytes"`
	UtilizationPercent float64 `json:"utilization_percent"`
}

func Detect() (*MemoryInfo, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return &MemoryInfo{}, err
	}

	return &MemoryInfo{
		TotalBytes:         v.Total,
		AvailableBytes:     v.Available,
		UsedBytes:          v.Used,
		UtilizationPercent: v.UsedPercent,
	}, nil
}
