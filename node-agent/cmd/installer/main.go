package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	baseDir           = `C:\ProgramData\DistributedAI`
	binDir            = `C:\ProgramData\DistributedAI\bin`
	configDir         = `C:\ProgramData\DistributedAI\config`
	configPath        = `C:\ProgramData\DistributedAI\config\config.json`
	modelDir          = `C:\ProgramData\DistributedAI\models`
	logDir            = `C:\ProgramData\DistributedAI\logs`
	serviceName       = "DistributedAINodeAgent"
	defaultController = "https://llm-cluster.onrender.com:443"
	defaultToken      = "dev-enrollment-token"
	defaultModelURL   = "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"
	llamaReleaseAPI   = "https://api.github.com/repos/ggml-org/llama.cpp/releases?per_page=20"
)

func main() {
	role := flag.String("role", "primary", "Node role: 'primary' or 'worker'")
	controllerURL := flag.String("controller-url", defaultController, "Target Python Controller URL")
	enrollmentToken := flag.String("enrollment-token", defaultToken, "Cluster enrollment token")
	rpcPort := flag.String("rpc-port", "50052", "LAN RPC port for worker offloading")
	skipModel := flag.Bool("skip-model", false, "Skip automatic model download")
	daemon := flag.Bool("daemon", false, "Run Node Agent as detached background process")
	flag.Parse()

	fmt.Println("=================================================================")
	fmt.Println("   Distributed AI System — Automated One-Click PC Installer     ")
	fmt.Println("=================================================================")

	// 1. Create System Directories
	fmt.Printf("\n[1/5] Setting up system directories under %s...\n", baseDir)
	for _, dir := range []string{baseDir, binDir, configDir, modelDir, logDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("  [!] Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}
	fmt.Println("  [✓] Directories ready.")

	// 2. Check / Copy Node Agent Executable
	fmt.Println("\n[2/5] Preparing Node Agent binary...")
	targetAgentBin := filepath.Join(binDir, "node-agent.exe")
	if _, err := os.Stat(targetAgentBin); os.IsNotExist(err) {
		// Look for local node-agent.exe in current working dir
		if _, errLocal := os.Stat("node-agent.exe"); errLocal == nil {
			copyFile("node-agent.exe", targetAgentBin)
			fmt.Printf("  [✓] Installed node-agent.exe to %s\n", targetAgentBin)
		} else if _, errBin := os.Stat(`bin\node-agent.exe`); errBin == nil {
			copyFile(`bin\node-agent.exe`, targetAgentBin)
			fmt.Printf("  [✓] Installed node-agent.exe to %s\n", targetAgentBin)
		} else {
			fmt.Printf("  [i] Target agent binary ready at %s\n", targetAgentBin)
		}
	} else {
		fmt.Printf("  [✓] Node Agent present at %s\n", targetAgentBin)
	}
	if !fileExists(targetAgentBin) {
		fmt.Printf("  [!] Node Agent binary is missing at %s. Build it before installing.\n", targetAgentBin)
		os.Exit(1)
	}

	configJSON := fmt.Sprintf(`{
  "controller_url": %q,
  "enrollment_token": %q,
  "node_data_dir": %q,
  "log_level": "INFO",
  "telemetry_interval_seconds": 5,
  "heartbeat_interval_seconds": 5,
  "max_concurrent_tasks": 4
}
`, *controllerURL, *enrollmentToken, baseDir)
	if err := os.WriteFile(configPath, []byte(configJSON), 0600); err != nil {
		fmt.Printf("  [!] Could not write agent config: %v\n", err)
		os.Exit(1)
	}

	// 3. Verify llama.cpp Binaries
	fmt.Println("\n[3/5] Checking llama.cpp CUDA binaries...")
	llamaCli := filepath.Join(binDir, "llama-cli.exe")
	llamaRpc := filepath.Join(binDir, "llama-rpc-server.exe")

	hasLlamaCli := fileExists(llamaCli)
	hasLlamaRpc := fileExists(llamaRpc)

	if !hasLlamaCli || !hasRuntimeDLLs(binDir) {
		fmt.Printf("  [i] llama.cpp binaries missing in %s. Downloading them now...\n", binDir)
		if err := downloadAndExtractLlama(binDir); err != nil {
			fmt.Printf("  [!] Could not install llama.cpp: %v\n", err)
			fmt.Println("  [i] Continuing installation. The node will connect to the controller, but inference remains unavailable until llama.cpp is installed.")
		}
		hasLlamaCli = fileExists(llamaCli)
		hasLlamaRpc = fileExists(llamaRpc)
	}
	if hasLlamaCli {
		if hasLlamaRpc {
			fmt.Println("  [✓] llama.cpp CLI and RPC binaries detected!")
		} else {
			fmt.Println("  [✓] llama.cpp CLI detected. RPC worker binary is optional for single-node inference.")
		}
	} else {
		fmt.Println("  [!] llama.cpp CLI is not installed. Node service will still start and remain online.")
	}

	// 4. Model Acquisition
	if !*skipModel {
		fmt.Println("\n[4/5] Checking GGUF LLM Model...")
		targetModel := filepath.Join(modelDir, "qwen2.5-0.5b-instruct-q4_k_m.gguf")
		if !fileExists(targetModel) {
			fmt.Printf("  [i] Downloading Qwen2.5 0.5B test model to %s...\n", targetModel)
			if err := downloadFile(defaultModelURL, targetModel); err != nil {
				fmt.Printf("  [!] Model download warning: %v\n", err)
			} else {
				fmt.Println("  [✓] GGUF Model downloaded successfully!")
			}
		} else {
			fmt.Println("  [✓] GGUF Model present.")
		}
	} else {
		fmt.Println("\n[4/5] Model download skipped.")
	}

	if err := installWindowsService(targetAgentBin); err != nil {
		fmt.Printf("  [!] Could not install Windows service: %v\n", err)
		fmt.Println("  [i] Run this installer from an elevated Administrator PowerShell.")
		os.Exit(1)
	}
	fmt.Printf("  [✓] Windows service %s registered for automatic startup.\n", serviceName)

	// 5. Launch Services
	fmt.Println("\n[5/5] Launching Distributed Node Services...")

	if hasLlamaRpc && (*role == "worker" || *role == "primary") {
		fmt.Printf("  [i] Starting background llama-rpc-server on 0.0.0.0:%s...\n", *rpcPort)
		cmdRPC := exec.Command(llamaRpc, "-H", "0.0.0.0", "-p", *rpcPort)
		if err := cmdRPC.Start(); err != nil {
			fmt.Printf("  [!] RPC Server start warning: %v\n", err)
		} else {
			fmt.Println("  [✓] llama-rpc-server running in background.")
		}
	}

	// Determine node-agent executable path
	executableToRun := targetAgentBin
	if !fileExists(executableToRun) {
		if fileExists("node-agent.exe") {
			executableToRun = "node-agent.exe"
		} else if fileExists(`bin\node-agent.exe`) {
			executableToRun = `bin\node-agent.exe`
		}
	}

	fmt.Printf("\nConnecting Node Agent to Controller at %s...\n", *controllerURL)

	if *daemon {
		fmt.Println("  [i] The Windows service owns the Node Agent process and will restart it after reboot or failure.")
	} else {
		cmdAgent := exec.Command(executableToRun, "--foreground", "--controller-url", *controllerURL, "--enrollment-token", *enrollmentToken)
		cmdAgent.Stdout = os.Stdout
		cmdAgent.Stderr = os.Stderr

		if err := cmdAgent.Run(); err != nil {
			fmt.Printf("Node Agent execution stopped: %v\n", err)
		}
	}
}

func installWindowsService(agentPath string) error {
	quotedBinary := fmt.Sprintf(`"%s" --config "%s"`, agentPath, configPath)

	if _, err := exec.LookPath("sc.exe"); err != nil {
		return fmt.Errorf("sc.exe not found: %w", err)
	}

	if query := exec.Command("sc.exe", "query", serviceName); query.Run() == nil {
		_ = exec.Command("sc.exe", "stop", serviceName).Run()
		_ = exec.Command("sc.exe", "delete", serviceName).Run()
		time.Sleep(2 * time.Second)
	}

	if output, err := exec.Command("sc.exe", "create", serviceName, "binPath=", quotedBinary, "start=", "delayed-auto", "DisplayName=", "Distributed AI Compute Node Agent").CombinedOutput(); err != nil {
		return fmt.Errorf("sc create failed: %v (%s)", err, output)
	}
	if output, err := exec.Command("sc.exe", "config", serviceName, "depend=", "Tcpip").CombinedOutput(); err != nil {
		return fmt.Errorf("sc dependency configuration failed: %v (%s)", err, output)
	}
	if output, err := exec.Command("sc.exe", "failure", serviceName, "reset=", "86400", "actions=", "restart/5000/restart/10000/restart/20000").CombinedOutput(); err != nil {
		return fmt.Errorf("sc failure configuration failed: %v (%s)", err, output)
	}
	if output, err := exec.Command("sc.exe", "start", serviceName).CombinedOutput(); err != nil {
		return fmt.Errorf("sc start failed: %v (%s)", err, output)
	}
	return nil
}

func downloadAndExtractLlama(destinationDir string) error {
	archivePath := filepath.Join(os.TempDir(), "distributed-ai-llama-cuda.zip")
	defer os.Remove(archivePath)

	llamaURL, err := resolveLlamaReleaseURL()
	if err != nil {
		return err
	}
	if err := downloadFile(llamaURL, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open archive failed: %w", err)
	}
	defer archive.Close()

	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return err
		}
		name := filepath.Base(entry.Name)
		output, err := os.Create(filepath.Join(destinationDir, name))
		if err == nil {
			_, err = io.Copy(output, reader)
		}
		_ = output.Close()
		_ = reader.Close()
		if err != nil {
			return fmt.Errorf("extract %s failed: %w", name, err)
		}
	}
	return nil
}

func resolveLlamaReleaseURL() (string, error) {
	request, err := http.NewRequest(http.MethodGet, llamaReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "DistributedAI-NodeAgent-Installer")
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return "", fmt.Errorf("query llama.cpp releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("query llama.cpp releases: HTTP %d", response.StatusCode)
	}
	var releases []struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("parse llama.cpp release metadata: %w", err)
	}
	var cpuURL string
	for _, release := range releases {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if !strings.HasPrefix(name, "llama-") || !strings.HasSuffix(name, ".zip") || !strings.Contains(name, "win") || !strings.Contains(name, "x64") {
				continue
			}
			if strings.Contains(name, "cuda") {
				return asset.URL, nil
			}
			if cpuURL == "" && strings.Contains(name, "cpu") {
				cpuURL = asset.URL
			}
		}
	}
	if cpuURL != "" {
		return cpuURL, nil
	}
	return "", fmt.Errorf("no Windows llama.cpp zip asset found in latest release")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasRuntimeDLLs(directory string) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".dll") {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: 15 * time.Minute}
	partialPath := filepath + ".part"
	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		resp, err := client.Get(url)
		if err == nil {
			if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("http status %d", resp.StatusCode)
				_ = resp.Body.Close()
			} else if out, createErr := os.Create(partialPath); createErr != nil {
				lastErr = createErr
				_ = resp.Body.Close()
			} else {
				progress := &downloadProgress{
					writer: out,
					name:   filepath,
					total:  resp.ContentLength,
				}
				_, copyErr := io.Copy(progress, resp.Body)
				_ = out.Close()
				_ = resp.Body.Close()
				progress.finish(copyErr == nil)
				if copyErr == nil {
					if renameErr := os.Rename(partialPath, filepath); renameErr == nil {
						return nil
					} else {
						lastErr = renameErr
					}
				} else {
					lastErr = copyErr
				}
			}
		} else {
			lastErr = err
		}
		_ = os.Remove(partialPath)
		time.Sleep(time.Duration(attempt*attempt) * time.Second)
	}
	return fmt.Errorf("download failed after 5 attempts: %w", lastErr)
}

type downloadProgress struct {
	writer      io.Writer
	name        string
	total       int64
	downloaded  int64
	lastPercent int
}

func (p *downloadProgress) Write(data []byte) (int, error) {
	n, err := p.writer.Write(data)
	p.downloaded += int64(n)
	if p.total > 0 {
		percent := int(p.downloaded * 100 / p.total)
		if percent != p.lastPercent {
			p.lastPercent = percent
			fmt.Printf("\rDownloading %s: %3d%% (%d/%d MB)", filepath.Base(p.name), percent, p.downloaded/(1024*1024), p.total/(1024*1024))
		}
	} else if p.downloaded%(10*1024*1024) < int64(n) {
		fmt.Printf("\rDownloading %s: %d MB", filepath.Base(p.name), p.downloaded/(1024*1024))
	}
	return n, err
}

func (p *downloadProgress) finish(success bool) {
	if success {
		fmt.Printf("\rDownloading %s: 100%% complete                 \n", filepath.Base(p.name))
	} else {
		fmt.Printf("\rDownloading %s: failed, retrying...\n", filepath.Base(p.name))
	}
}
