package storage

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/disk"
)

type StorageInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	UsedPercent    float64 `json:"used_percent"`
}

func Detect() (*StorageInfo, error) {
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:\\"
	}

	usage, err := disk.Usage(path)
	if err != nil {
		return &StorageInfo{}, err
	}

	return &StorageInfo{
		TotalBytes:     usage.Total,
		AvailableBytes: usage.Free,
		UsedBytes:      usage.Used,
		UsedPercent:    usage.UsedPercent,
	}, nil
}
