package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/distributed-ai/distributed-runtime/pkg/llama"
)

type HealthChecker struct {
	sm         *StateMachine
	lpm        *llama.LlamaProcessManager
	rwm        *llama.RPCWorkerManager
	lastCheck  time.Time
	lastReport *llama.LlamaHealthReport
}

func NewHealthChecker(sm *StateMachine, lpm *llama.LlamaProcessManager, rwm *llama.RPCWorkerManager) *HealthChecker {
	return &HealthChecker{
		sm:  sm,
		lpm: lpm,
		rwm: rwm,
	}
}

func (hc *HealthChecker) CheckHealth(ctx context.Context) error {
	state := hc.sm.GetState()
	if state == StateError {
		return fmt.Errorf("runtime in ERROR state")
	}
	if state == StateStopping {
		return fmt.Errorf("runtime is STOPPING")
	}

	report := llama.PerformLlamaHealthCheck(ctx, "", "")
	hc.lastReport = report
	hc.lastCheck = time.Now()

	if hc.rwm != nil {
		if err := hc.rwm.Health(); err != nil {
			_ = hc.sm.TransitionTo(StateDegraded, fmt.Sprintf("RPC worker health issue: %v", err))
		}
	}

	return nil
}

func (hc *HealthChecker) LastReport() *llama.LlamaHealthReport {
	return hc.lastReport
}
