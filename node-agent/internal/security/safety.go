package security

import (
	"fmt"

	"github.com/distributed-ai/node-agent/internal/config"
	"github.com/distributed-ai/node-agent/internal/hardware"
)

type SafetyCheckResult struct {
	Safe         bool     `json:"safe"`
	Violations   []string `json:"violations"`
	ThermalAlert bool     `json:"thermal_alert"`
	MemoryAlert  bool     `json:"memory_alert"`
	StorageAlert bool     `json:"storage_alert"`
}

func EvaluateHardwareSafety(report *hardware.FullHardwareReport, cfg *config.Config) SafetyCheckResult {
	res := SafetyCheckResult{
		Safe:       true,
		Violations: make([]string, 0),
	}

	if report.Memory != nil && cfg.MaxRAMUsagePercent > 0 {
		if report.Memory.UtilizationPercent > cfg.MaxRAMUsagePercent {
			res.Safe = false
			res.MemoryAlert = true
			res.Violations = append(res.Violations, fmt.Sprintf("RAM usage (%.1f%%) exceeds maximum safety limit (%.1f%%)", report.Memory.UtilizationPercent, cfg.MaxRAMUsagePercent))
		}
	}

	if report.Storage != nil && cfg.MinFreeDiskGB > 0 {
		freeGB := float64(report.Storage.AvailableBytes) / (1024 * 1024 * 1024)
		if freeGB < cfg.MinFreeDiskGB {
			res.Safe = false
			res.StorageAlert = true
			res.Violations = append(res.Violations, fmt.Sprintf("Free disk space (%.2f GB) is below minimum required (%.2f GB)", freeGB, cfg.MinFreeDiskGB))
		}
	}

	if report.GPUReport != nil && report.GPUReport.GPUAvailable {
		for idx, g := range report.GPUReport.GPUs {
			if cfg.MaxGPUTempC > 0 && float64(g.TemperatureC) > cfg.MaxGPUTempC {
				res.Safe = false
				res.ThermalAlert = true
				res.Violations = append(res.Violations, fmt.Sprintf("GPU %d (%s) temperature (%d°C) exceeds safety limit (%.1f°C)", idx, g.Name, g.TemperatureC, cfg.MaxGPUTempC))
			}
		}
	}

	return res
}
