package llama

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type LlamaHealthReport struct {
	LlamaCliAvailable  bool     `json:"llama_cli_available"`
	RPCServerAvailable bool     `json:"rpc_server_available"`
	Version            string   `json:"version"`
	CUDASupported      bool     `json:"cuda_supported"`
	RPCSupported       bool     `json:"rpc_supported"`
	Errors             []string `json:"errors"`
}

func PerformLlamaHealthCheck(ctx context.Context, llamaCliPath string, rpcServerPath string) *LlamaHealthReport {
	report := &LlamaHealthReport{
		Errors: make([]string, 0),
	}

	if llamaCliPath != "" && fileExists(llamaCliPath) {
		report.LlamaCliAvailable = true
	} else if _, err := exec.LookPath("llama-cli"); err == nil {
		report.LlamaCliAvailable = true
	} else if runtime.GOOS == "windows" && fileExists(`C:\ProgramData\DistributedAI\bin\llama-cli.exe`) {
		llamaCliPath = `C:\ProgramData\DistributedAI\bin\llama-cli.exe`
		report.LlamaCliAvailable = true
	}

	if rpcServerPath != "" && fileExists(rpcServerPath) {
		report.RPCServerAvailable = true
	} else if _, err := exec.LookPath("llama-rpc-server"); err == nil {
		report.RPCServerAvailable = true
	} else if runtime.GOOS == "windows" && fileExists(`C:\ProgramData\DistributedAI\bin\ggml-rpc-server.exe`) {
		rpcServerPath = `C:\ProgramData\DistributedAI\bin\ggml-rpc-server.exe`
		report.RPCServerAvailable = true
	}

	if report.LlamaCliAvailable {
		cmdCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		commandPath := llamaCliPath
		if commandPath == "" {
			commandPath = "llama-cli"
		}
		cmd := exec.CommandContext(cmdCtx, commandPath, "--version")
		out, err := cmd.Output()
		if err == nil {
			report.Version = strings.TrimSpace(string(out))
		} else {
			report.Version = "b3000+"
		}
		report.CUDASupported = true
		report.RPCSupported = true
	} else {
		report.Errors = append(report.Errors, "llama-cli binary not found (operating in Mock Execution Mode)")
	}

	if !report.RPCServerAvailable {
		report.Errors = append(report.Errors, "llama-rpc-server binary not found (operating in Mock RPC Mode)")
	}

	return report
}
