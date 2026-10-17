package llama

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/distributed-ai/distributed-runtime/pkg/execution"
)

type LlamaProcessManager struct {
	mu         sync.Mutex
	binaryPath string
	activeCmd  *exec.Cmd
	mockMode   bool
}

func NewLlamaProcessManager(binaryPath string) *LlamaProcessManager {
	mock := false
	if binaryPath == "" || !fileExists(binaryPath) {
		foundPath, err := exec.LookPath("llama-cli")
		if err == nil {
			binaryPath = foundPath
		} else {
			progDataPath := `C:\ProgramData\DistributedAI\bin\llama-cli.exe`
			if fileExists(progDataPath) {
				binaryPath = progDataPath
			} else {
				mock = true
				log.Printf("[LlamaProcessManager] llama-cli binary not found on host. Operating in Mock Execution Driver mode.")
			}
		}
	}
	return &LlamaProcessManager{
		binaryPath: binaryPath,
		mockMode:   mock,
	}
}

func (lpm *LlamaProcessManager) IsMockMode() bool {
	lpm.mu.Lock()
	defer lpm.mu.Unlock()
	return lpm.mockMode
}

func (lpm *LlamaProcessManager) ExecuteStream(
	ctx context.Context,
	plan *execution.ExecutionPlan,
	prompt string,
	chunkCallback func(chunk string, tokens int) error,
) error {
	execCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	ctx = execCtx
	lpm.mu.Lock()
	if lpm.mockMode {
		lpm.mu.Unlock()
		return fmt.Errorf("real inference unavailable: llama-cli executable was not found")
	}
	if !fileExists(plan.ModelPath) {
		lpm.mu.Unlock()
		return fmt.Errorf("real inference unavailable: model file not found at %s", plan.ModelPath)
	}
	lpm.mu.Unlock()

	args := []string{
		"-m", plan.ModelPath,
		"-p", prompt,
		"-n", strconv.Itoa(plan.MaxTokens),
		"-c", strconv.Itoa(plan.ContextLength),
		"-b", strconv.Itoa(plan.BatchSize),
		"--single-turn",
		"--no-display-prompt",
	}

	if plan.SplitMode == execution.SplitModeLayerRPC && len(plan.RPCEndpoints) > 0 {
		rpcStr := strings.Join(plan.RPCEndpoints, ",")
		args = append(args, "--rpc", rpcStr)

		if len(plan.TensorSplit) > 0 {
			splitStrs := make([]string, len(plan.TensorSplit))
			for i, val := range plan.TensorSplit {
				splitStrs[i] = fmt.Sprintf("%.2f", val)
			}
			args = append(args, "--tensor-split", strings.Join(splitStrs, ","))
		}
	} else if runtime.GOOS != "windows" && len(plan.Assignments) > 0 && plan.Assignments[0].GPULayersCount > 0 {
		args = append(args, "-ngl", strconv.Itoa(plan.Assignments[0].GPULayersCount))
	} else if runtime.GOOS == "windows" {
		args = append(args, "-ngl", "0")
	}

	cmd := exec.CommandContext(ctx, lpm.binaryPath, args...)
	cmd.Dir = filepath.Dir(lpm.binaryPath)
	cmd.Stdin = strings.NewReader("")
	lpm.mu.Lock()
	lpm.activeCmd = cmd
	lpm.mu.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe llama-cli stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe llama-cli stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama-cli process: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[llama-cli stderr] %s", scanner.Text())
		}
	}()

	reader := bufio.NewReader(stdout)
	tokenCount := 0
	var outputBuffer strings.Builder
	flushOutput := func() error {
		if outputBuffer.Len() == 0 {
			return nil
		}
		chunk := outputBuffer.String()
		outputBuffer.Reset()
		return chunkCallback(chunk, tokenCount)
	}

	for {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return ctx.Err()
		default:
		}

		r, _, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed reading llama-cli output: %w", err)
		}

		charStr := string(r)
		tokenCount++
		outputBuffer.WriteString(charStr)
		if outputBuffer.Len() >= 256 {
			if err := flushOutput(); err != nil {
				_ = cmd.Process.Kill()
				return err
			}
		}
	}
	if err := flushOutput(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("llama-cli timed out after 90 seconds: %w", ctx.Err())
		}
		return fmt.Errorf("llama-cli exited with error: %w", err)
	}
	if tokenCount == 0 {
		return fmt.Errorf("llama-cli produced no output")
	}
	return nil
}

func (lpm *LlamaProcessManager) executeMockStream(
	ctx context.Context,
	plan *execution.ExecutionPlan,
	prompt string,
	chunkCallback func(chunk string, tokens int) error,
) error {
	nodesDesc := "Single-Node"
	if len(plan.Assignments) > 1 {
		nodesDesc = fmt.Sprintf("Distributed Multi-Node (%d PCs)", len(plan.Assignments))
	}

	simulatedText := fmt.Sprintf("[Distributed Runtime v1 Response | Mode: %s | Model: %s] Processed prompt: '%s'. High performance inference execution completed across assigned GPUs.", nodesDesc, filepath.Base(plan.ModelPath), prompt)
	words := strings.Split(simulatedText, " ")

	tokenCount := 0
	for _, word := range words {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(60 * time.Millisecond):
		}

		tokenCount++
		chunk := word + " "
		if err := chunkCallback(chunk, tokenCount); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
