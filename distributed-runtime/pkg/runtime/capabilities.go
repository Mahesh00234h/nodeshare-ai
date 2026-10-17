package runtime

import (
	"runtime"
	"strings"

	"github.com/distributed-ai/distributed-runtime/pkg/communication"
	"github.com/distributed-ai/distributed-runtime/pkg/llama"
)

type RuntimeCapabilitiesReport struct {
	OS                  string   `json:"os"`
	Architecture        string   `json:"architecture"`
	SupportedRuntimes   []string `json:"supported_runtimes"`
	CUDADevices         int      `json:"cuda_devices"`
	TotalVRAMMB         uint64   `json:"total_vram_mb"`
	RPCServerSupported  bool     `json:"rpc_server_supported"`
	LayerSplitSupported bool     `json:"layer_split_supported"`
	PreferredLANIP      string   `json:"preferred_lan_ip"`
	RuntimeVersion      string   `json:"runtime_version"`
}

func DetectRuntimeCapabilities() *RuntimeCapabilitiesReport {
	lanInfo, _ := communication.DetectBestLANInterface()
	prefIP := "127.0.0.1"
	if lanInfo != nil && lanInfo.PreferredIP != "" {
		prefIP = lanInfo.PreferredIP
	}

	cudaDevs, err := llama.DetectLocalCUDADevices()
	var totalVRAM uint64
	devCount := 0

	if err == nil && len(cudaDevs) > 0 {
		devCount = len(cudaDevs)
		for _, dev := range cudaDevs {
			totalVRAM += dev.TotalVRAMBytes / (1024 * 1024)
		}
	}

	return &RuntimeCapabilitiesReport{
		OS:                  runtime.GOOS,
		Architecture:        runtime.GOARCH,
		SupportedRuntimes:   []string{"llama.cpp", "gguf", "rpc-worker"},
		CUDADevices:         devCount,
		TotalVRAMMB:         totalVRAM,
		RPCServerSupported:  true,
		LayerSplitSupported: true,
		PreferredLANIP:      prefIP,
		RuntimeVersion:      "1.0.0",
	}
}

func (rcr *RuntimeCapabilitiesReport) ToSlice() []string {
	caps := []string{
		"runtime:distributed-v1",
		"engine:llama.cpp",
		"format:gguf",
		"rpc:worker",
		"rpc:lead",
		"split:layer",
		"split:tensor",
	}
	if rcr.CUDADevices > 0 {
		caps = append(caps, "gpu:cuda")
	}
	if strings.HasPrefix(rcr.PreferredLANIP, "192.168.") || strings.HasPrefix(rcr.PreferredLANIP, "10.") {
		caps = append(caps, "net:lan-private")
	}
	return caps
}
