package gpu

import (
	"runtime"
	"syscall"
	"unsafe"
)

type GPUInfo struct {
	Vendor            string  `json:"vendor"`
	Name              string  `json:"name"`
	VRAMTotalBytes    uint64  `json:"vram_total_bytes"`
	VRAMUsedBytes     uint64  `json:"vram_used_bytes"`
	VRAMAvailBytes    uint64  `json:"vram_available_bytes"`
	UtilizationPercent float64 `json:"utilization_percent"`
	TemperatureC      uint32  `json:"temperature_c"`
	PowerUsageWatts   float64 `json:"power_usage_watts"`
	DriverVersion     string  `json:"driver_version"`
	ComputeCapability string  `json:"compute_capability"`
}

type GPUReport struct {
	GPUAvailable bool      `json:"gpu_available"`
	GPUs         []GPUInfo `json:"gpus"`
}

// Detect performs dynamic NVML loading to query NVIDIA GPU details safely.
func Detect() (*GPUReport, error) {
	report := &GPUReport{
		GPUAvailable: false,
		GPUs:         make([]GPUInfo, 0),
	}

	var libName string
	if runtime.GOOS == "windows" {
		libName = "nvml.dll"
	} else {
		libName = "libnvidia-ml.so"
	}

	mod, err := syscall.LoadLibrary(libName)
	if err != nil {
		// NVML library not found on this system -> return safely
		return report, nil
	}
	defer syscall.FreeLibrary(mod)

	initProc, err := syscall.GetProcAddress(mod, "nvmlInit_v2")
	if err != nil {
		initProc, err = syscall.GetProcAddress(mod, "nvmlInit")
		if err != nil {
			return report, nil
		}
	}

	ret, _, _ := syscall.SyscallN(initProc)
	if ret != 0 {
		return report, nil
	}

	defer func() {
		if shutdownProc, err := syscall.GetProcAddress(mod, "nvmlShutdown"); err == nil {
			_, _, _ = syscall.SyscallN(shutdownProc)
		}
	}()

	getCountProc, err := syscall.GetProcAddress(mod, "nvmlDeviceGetCount_v2")
	if err != nil {
		getCountProc, _ = syscall.GetProcAddress(mod, "nvmlDeviceGetCount")
	}
	if getCountProc == 0 {
		return report, nil
	}

	var count uint32
	ret, _, _ = syscall.SyscallN(getCountProc, uintptr(unsafe.Pointer(&count)))
	if ret != 0 || count == 0 {
		return report, nil
	}

	getHandleProc, _ := syscall.GetProcAddress(mod, "nvmlDeviceGetHandleByIndex_v2")
	if getHandleProc == 0 {
		getHandleProc, _ = syscall.GetProcAddress(mod, "nvmlDeviceGetHandleByIndex")
	}
	getNameProc, _ := syscall.GetProcAddress(mod, "nvmlDeviceGetName")
	getMemProc, _ := syscall.GetProcAddress(mod, "nvmlDeviceGetMemoryInfo")

	if getHandleProc == 0 || getNameProc == 0 || getMemProc == 0 {
		return report, nil
	}

	report.GPUAvailable = true

	type nvmlMemory struct {
		Total uint64
		Free  uint64
		Used  uint64
	}

	for i := uint32(0); i < count; i++ {
		var device uintptr
		ret, _, _ = syscall.SyscallN(getHandleProc, uintptr(i), uintptr(unsafe.Pointer(&device)))
		if ret != 0 {
			continue
		}

		nameBuf := make([]byte, 64)
		ret, _, _ = syscall.SyscallN(getNameProc, device, uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(len(nameBuf)))
		gpuName := "NVIDIA GPU"
		if ret == 0 {
			for idx, b := range nameBuf {
				if b == 0 {
					gpuName = string(nameBuf[:idx])
					break
				}
			}
		}

		var memInfo nvmlMemory
		ret, _, _ = syscall.SyscallN(getMemProc, device, uintptr(unsafe.Pointer(&memInfo)))
		var total, used, free uint64
		if ret == 0 {
			total = memInfo.Total
			used = memInfo.Used
			free = memInfo.Free
		}

		var utilPercent float64
		if total > 0 {
			utilPercent = (float64(used) / float64(total)) * 100.0
		}

		gpu := GPUInfo{
			Vendor:             "NVIDIA",
			Name:               gpuName,
			VRAMTotalBytes:     total,
			VRAMUsedBytes:      used,
			VRAMAvailBytes:     free,
			UtilizationPercent: utilPercent,
			DriverVersion:      "Dynamic NVML",
			ComputeCapability:  "CUDA",
		}

		report.GPUs = append(report.GPUs, gpu)
	}

	return report, nil
}
