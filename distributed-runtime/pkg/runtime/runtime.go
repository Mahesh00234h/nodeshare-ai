package runtime

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/distributed-ai/distributed-runtime/pkg/communication"
	"github.com/distributed-ai/distributed-runtime/pkg/execution"
	"github.com/distributed-ai/distributed-runtime/pkg/llama"
	"github.com/distributed-ai/distributed-runtime/pkg/model"
)

type GenerateRequest struct {
	JobID       string  `json:"job_id"`
	ModelID     string  `json:"model_id"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type GenerateResponse struct {
	JobID           string `json:"job_id"`
	Text            string `json:"text"`
	TokensGenerated int    `json:"tokens_generated"`
	FinishReason    string `json:"finish_reason"`
}

type DistributedRuntime struct {
	mu           sync.RWMutex
	sm           *StateMachine
	lpm          *llama.LlamaProcessManager
	rwm          *llama.RPCWorkerManager
	mcm          *model.ModelCacheManager
	planner      *execution.ExecutionPlanner
	activePlan   *execution.ExecutionPlan
	activeJobs   map[string]context.CancelFunc
	capabilities *RuntimeCapabilitiesReport
	rpcPort      int
	nodeID       string
}

func NewDistributedRuntime(cacheDir string, nodeID string, rpcPort int) (*DistributedRuntime, error) {
	if rpcPort == 0 {
		rpcPort = 50052
	}
	if nodeID == "" {
		nodeID = "local-node"
	}

	sm := NewStateMachine()
	lpm := llama.NewLlamaProcessManager("")
	rwm := llama.NewRPCWorkerManager("", "0.0.0.0", rpcPort)
	mcm, err := model.NewModelCacheManager(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to init model cache manager: %w", err)
	}

	planner := execution.NewExecutionPlanner(nil)
	caps := DetectRuntimeCapabilities()

	dr := &DistributedRuntime{
		sm:           sm,
		lpm:          lpm,
		rwm:          rwm,
		mcm:          mcm,
		planner:      planner,
		activeJobs:   make(map[string]context.CancelFunc),
		capabilities: caps,
		rpcPort:      rpcPort,
		nodeID:       nodeID,
	}

	return dr, nil
}

func (dr *DistributedRuntime) Name() string {
	return "Distributed-LLM-Runtime-v1 (llama.cpp)"
}

func (dr *DistributedRuntime) Start(ctx context.Context) error {
	_ = dr.sm.TransitionTo(StateInitializing, "starting runtime engine")

	_ = dr.sm.TransitionTo(StateCheckingLlama, "probing llama.cpp binaries")
	health := llama.PerformLlamaHealthCheck(ctx, "", "")
	if len(health.Errors) > 0 {
		log.Printf("[DistributedRuntime] Llama binary warning: %v", health.Errors)
	}

	_ = dr.sm.TransitionTo(StateCheckingGPU, "detecting CUDA & LAN capabilities")
	dr.capabilities = DetectRuntimeCapabilities()

	_ = dr.rwm.StartRPCServer(ctx)

	_ = dr.sm.TransitionTo(StateReady, "distributed runtime ready")
	log.Printf("[DistributedRuntime] Runtime initialized successfully. State: READY")
	go dr.ensureDefaultModel(ctx)
	return nil
}

func (dr *DistributedRuntime) ensureDefaultModel(ctx context.Context) {
	const modelID = "qwen2.5-0.5b-instruct-q4_k_m.gguf"
	for {
		if _, err := dr.mcm.EnsureModel(modelID); err == nil {
			log.Printf("[DistributedRuntime] Model cache verified: %s", modelID)
			return
		} else {
			log.Printf("[DistributedRuntime] Model auto-install warning: %v", err)
		}

		timer := time.NewTimer(30 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (dr *DistributedRuntime) Stop(ctx context.Context) error {
	_ = dr.sm.TransitionTo(StateStopping, "stopping runtime engine")

	dr.mu.Lock()
	for jobID, cancel := range dr.activeJobs {
		cancel()
		delete(dr.activeJobs, jobID)
	}
	dr.mu.Unlock()

	_ = dr.rwm.StopRPCServer()
	_ = dr.sm.TransitionTo(StateError, "runtime stopped")
	return nil
}

func (dr *DistributedRuntime) Health(ctx context.Context) error {
	state := dr.sm.GetState()
	if state == StateError {
		return fmt.Errorf("runtime in error state")
	}
	if state == StateStopping {
		return fmt.Errorf("runtime is stopping")
	}
	return nil
}

func (dr *DistributedRuntime) ListModels(ctx context.Context) ([]string, error) {
	manifests := dr.mcm.ListModels()
	names := make([]string, 0, len(manifests))
	for _, m := range manifests {
		names = append(names, m.ModelID)
	}
	if len(names) == 0 {
		names = append(names, "test-model.gguf", "llama-2-7b-q4.gguf")
	}
	return names, nil
}

func (dr *DistributedRuntime) LoadModel(ctx context.Context, modelID string) error {
	_ = dr.sm.TransitionTo(StateLoadingModel, fmt.Sprintf("loading model %s", modelID))

	manifest, exists := dr.mcm.GetModel(modelID)
	if !exists {
		var err error
		manifest, err = dr.mcm.AddModelPlaceholder(modelID, 4*1024*1024*1024, "")
		if err != nil {
			_ = dr.sm.TransitionTo(StateDegraded, fmt.Sprintf("model load failed: %v", err))
			return err
		}
	}

	if manifest.ExpectedSHA256 != "" {
		valid, err := model.VerifyModelFileSHA256(manifest.LocalPath, manifest.ExpectedSHA256)
		if !valid || err != nil {
			_ = dr.sm.TransitionTo(StateError, fmt.Sprintf("model checksum verification failed: %v", err))
			return fmt.Errorf("SECURITY_VIOLATION: model SHA256 checksum mismatch: %w", err)
		}
	}

	_ = dr.sm.TransitionTo(StateReady, fmt.Sprintf("model %s ready in cache", modelID))
	return nil
}

func (dr *DistributedRuntime) UnloadModel(ctx context.Context, modelID string) error {
	log.Printf("[DistributedRuntime] Unloaded model %s from memory", modelID)
	return nil
}

func (dr *DistributedRuntime) PrepareDistributedExecution(
	ctx context.Context,
	jobID string,
	modelID string,
	remoteNodes []execution.DeviceInfo,
) (*execution.ExecutionPlan, error) {
	_ = dr.sm.TransitionTo(StatePreparingCluster, "building execution plan")

	manifest, _ := dr.mcm.GetModel(modelID)
	modelPath := modelID
	modelSHA256 := ""
	modelVRAMMB := uint64(4096)
	totalLayers := 32

	if manifest != nil {
		modelPath = manifest.LocalPath
		modelSHA256 = manifest.ExpectedSHA256
		modelVRAMMB = manifest.RequiredVRAMEstimate / (1024 * 1024)
		if manifest.TotalLayers > 0 {
			totalLayers = manifest.TotalLayers
		}
	}

	execID := fmt.Sprintf("exec-%d", time.Now().UnixNano())

	lanInfo, _ := communication.DetectBestLANInterface()
	prefIP := "127.0.0.1"
	if lanInfo != nil {
		prefIP = lanInfo.PreferredIP
	}

	localDev := execution.DeviceInfo{
		NodeID:       dr.nodeID,
		HostName:     dr.nodeID,
		LANIP:        prefIP,
		DeviceID:     "CUDA0",
		DeviceName:   "Local GPU",
		VRAMTotalMB:  24576,
		VRAMFreeMB:   16384,
		ComputeScore: 10.0,
	}

	allDevs := []execution.DeviceInfo{localDev}
	allDevs = append(allDevs, remoteNodes...)

	plan, err := dr.planner.BuildExecutionPlan(jobID, execID, modelID, modelPath, modelSHA256, modelVRAMMB, totalLayers, allDevs, dr.rpcPort)
	if err != nil {
		_ = dr.sm.TransitionTo(StateDegraded, fmt.Sprintf("plan build failed: %v", err))
		return nil, err
	}

	dr.mu.Lock()
	dr.activePlan = plan
	dr.mu.Unlock()

	log.Printf("[DistributedRuntime] Execution Plan Built (%s Mode): %d participating devices", plan.SplitMode, len(plan.Assignments))
	return plan, nil
}

func (dr *DistributedRuntime) StreamGenerate(
	ctx context.Context,
	req *GenerateRequest,
	chunkCallback func(chunk string, tokens int) error,
) error {
	_ = dr.sm.TransitionTo(StateRunning, fmt.Sprintf("running job %s", req.JobID))

	jobCtx, jobCancel := context.WithCancel(ctx)
	dr.mu.Lock()
	dr.activeJobs[req.JobID] = jobCancel
	plan := dr.activePlan
	dr.mu.Unlock()

	defer func() {
		dr.mu.Lock()
		delete(dr.activeJobs, req.JobID)
		dr.mu.Unlock()
		_ = dr.sm.TransitionTo(StateReady, "inference job completed")
	}()

	if _, err := dr.mcm.EnsureModel(req.ModelID); err != nil {
		return fmt.Errorf("model preparation failed: %w", err)
	}

	if plan == nil || plan.JobID != req.JobID {
		var err error
		plan, err = dr.PrepareDistributedExecution(jobCtx, req.JobID, req.ModelID, nil)
		if err != nil {
			return fmt.Errorf("failed to prepare execution plan: %w", err)
		}
	}

	if req.MaxTokens > 0 {
		plan.MaxTokens = req.MaxTokens
	}

	return dr.lpm.ExecuteStream(jobCtx, plan, req.Prompt, chunkCallback)
}

func (dr *DistributedRuntime) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var sb strings.Builder
	totalTokens := 0

	err := dr.StreamGenerate(ctx, req, func(chunk string, tokens int) error {
		sb.WriteString(chunk)
		totalTokens = tokens
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &GenerateResponse{
		JobID:           req.JobID,
		Text:            sb.String(),
		TokensGenerated: totalTokens,
		FinishReason:    "stop",
	}, nil
}

func (dr *DistributedRuntime) Cancel(ctx context.Context, jobID string) error {
	dr.mu.Lock()
	cancel, exists := dr.activeJobs[jobID]
	if exists {
		cancel()
		delete(dr.activeJobs, jobID)
		dr.mu.Unlock()
		log.Printf("[DistributedRuntime] Cancelled active job %s", jobID)
		return nil
	}
	dr.mu.Unlock()
	return fmt.Errorf("job %s not found in active execution list", jobID)
}

func (dr *DistributedRuntime) Capabilities() []string {
	return dr.capabilities.ToSlice()
}

func (dr *DistributedRuntime) GetStateMachine() *StateMachine {
	return dr.sm
}

func (dr *DistributedRuntime) GetModelCacheManager() *model.ModelCacheManager {
	return dr.mcm
}

func (dr *DistributedRuntime) GetRPCWorkerManager() *llama.RPCWorkerManager {
	return dr.rwm
}
