package llama

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"sync"

	"github.com/distributed-ai/distributed-runtime/pkg/communication"
)

type RPCWorkerManager struct {
	mu         sync.Mutex
	binaryPath string
	bindHost   string
	port       int
	cmd        *exec.Cmd
	security   *communication.LANSecurityGuard
	isRunning  bool
	mockWorker bool
}

func NewRPCWorkerManager(binaryPath string, bindHost string, port int) *RPCWorkerManager {
	if port == 0 {
		port = 50052
	}
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}

	mock := false
	if binaryPath == "" || !fileExists(binaryPath) {
		foundPath, err := exec.LookPath("llama-rpc-server")
		if err == nil {
			binaryPath = foundPath
		} else {
			progDataPath := `C:\ProgramData\DistributedAI\bin\llama-rpc-server.exe`
			if fileExists(progDataPath) {
				binaryPath = progDataPath
			} else {
				mock = true
			}
		}
	}

	return &RPCWorkerManager{
		binaryPath: binaryPath,
		bindHost:   bindHost,
		port:       port,
		security:   communication.NewLANSecurityGuard(nil),
		mockWorker: mock,
	}
}

func (rwm *RPCWorkerManager) StartRPCServer(ctx context.Context) error {
	rwm.mu.Lock()
	defer rwm.mu.Unlock()

	if rwm.isRunning {
		return nil
	}

	if err := rwm.security.ValidateBindHost(rwm.bindHost); err != nil {
		return fmt.Errorf("SECURITY_FAILURE: failed to start RPC server: %w", err)
	}

	if rwm.mockWorker {
		log.Printf("[RPCWorkerManager] Starting Mock RPC Server listening on LAN interface %s:%d", rwm.bindHost, rwm.port)
		rwm.isRunning = true
		return nil
	}

	args := []string{
		"-H", rwm.bindHost,
		"-p", strconv.Itoa(rwm.port),
	}

	cmd := exec.CommandContext(ctx, rwm.binaryPath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch llama-rpc-server process: %w", err)
	}

	rwm.cmd = cmd
	rwm.isRunning = true
	log.Printf("[RPCWorkerManager] llama-rpc-server started on %s:%d (PID %d)", rwm.bindHost, rwm.port, cmd.Process.Pid)

	go func() {
		_ = cmd.Wait()
		rwm.mu.Lock()
		rwm.isRunning = false
		rwm.mu.Unlock()
		log.Printf("[RPCWorkerManager] llama-rpc-server on port %d exited", rwm.port)
	}()

	return nil
}

func (rwm *RPCWorkerManager) StopRPCServer() error {
	rwm.mu.Lock()
	defer rwm.mu.Unlock()

	if !rwm.isRunning {
		return nil
	}

	if rwm.cmd != nil && rwm.cmd.Process != nil {
		_ = rwm.cmd.Process.Kill()
	}
	rwm.isRunning = false
	log.Printf("[RPCWorkerManager] llama-rpc-server stopped")
	return nil
}

func (rwm *RPCWorkerManager) Health() error {
	rwm.mu.Lock()
	defer rwm.mu.Unlock()

	if !rwm.isRunning {
		return fmt.Errorf("RPC worker is not running")
	}

	if rwm.mockWorker {
		return nil
	}

	endpoint := fmt.Sprintf("127.0.0.1:%d", rwm.port)
	_, err := communication.MeasureLANLatency(endpoint)
	if err != nil {
		return fmt.Errorf("RPC server tcp health check failed on %s: %w", endpoint, err)
	}
	return nil
}

func (rwm *RPCWorkerManager) Endpoint(lanIP string) string {
	rwm.mu.Lock()
	defer rwm.mu.Unlock()
	return fmt.Sprintf("%s:%d", lanIP, rwm.port)
}
