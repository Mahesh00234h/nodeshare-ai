package inference

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MockInferenceRuntime struct {
	mu           sync.RWMutex
	loadedModels map[string]bool
	activeJobs   map[string]context.CancelFunc
	healthy      bool
}

func NewMockInferenceRuntime() *MockInferenceRuntime {
	return &MockInferenceRuntime{
		loadedModels: make(map[string]bool),
		activeJobs:   make(map[string]context.CancelFunc),
		healthy:      true,
	}
}

func (m *MockInferenceRuntime) Name() string {
	return "MockInferenceRuntime"
}

func (m *MockInferenceRuntime) Start(ctx context.Context) error {
	m.mu.Lock()
	m.healthy = true
	m.mu.Unlock()
	return nil
}

func (m *MockInferenceRuntime) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.healthy = false
	for _, cancel := range m.activeJobs {
		cancel()
	}
	m.mu.Unlock()
	return nil
}

func (m *MockInferenceRuntime) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.healthy {
		return fmt.Errorf("mock runtime unhealthy")
	}
	return nil
}

func (m *MockInferenceRuntime) ListModels(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]string, 0, len(m.loadedModels))
	for k := range m.loadedModels {
		list = append(list, k)
	}
	return list, nil
}

func (m *MockInferenceRuntime) LoadModel(ctx context.Context, modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadedModels[modelID] = true
	return nil
}

func (m *MockInferenceRuntime) UnloadModel(ctx context.Context, modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.loadedModels, modelID)
	return nil
}

func (m *MockInferenceRuntime) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var fullText string
	var totalTokens int

	err := m.StreamGenerate(ctx, req, func(chunk string, tokens int) error {
		fullText += chunk
		totalTokens = tokens
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &GenerateResponse{
		JobID:           req.JobID,
		Text:            fullText,
		TokensGenerated: totalTokens,
		FinishReason:    "completed",
	}, nil
}

func (m *MockInferenceRuntime) StreamGenerate(ctx context.Context, req *GenerateRequest, chunkCallback func(chunk string, tokens int) error) error {
	jobCtx, cancel := context.WithCancel(ctx)
	m.mu.Lock()
	m.activeJobs[req.JobID] = cancel
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.activeJobs, req.JobID)
		m.mu.Unlock()
		cancel()
	}()

	mockWords := []string{
		"Distributed", "AI", "cluster", "node", "successfully", "processed",
		"inference", "workload", "for", "model", req.ModelID, "with", "high",
		"throughput", "and", "low", "latency.",
	}

	maxTok := req.MaxTokens
	if maxTok <= 0 {
		maxTok = len(mockWords)
	}

	for i := 0; i < maxTok; i++ {
		select {
		case <-jobCtx.Done():
			return fmt.Errorf("job %s cancelled: %w", req.JobID, jobCtx.Err())
		case <-time.After(20 * time.Millisecond):
			word := mockWords[i%len(mockWords)] + " "
			if err := chunkCallback(word, i+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *MockInferenceRuntime) Cancel(ctx context.Context, jobID string) error {
	m.mu.Lock()
	cancel, ok := m.activeJobs[jobID]
	m.mu.Unlock()

	if ok && cancel != nil {
		cancel()
		return nil
	}
	return fmt.Errorf("job %s not found or already finished", jobID)
}

func (m *MockInferenceRuntime) Capabilities() []string {
	return []string{"CPU_INFERENCE", "GPU_INFERENCE", "TEXT_GENERATION", "STREAMING"}
}
