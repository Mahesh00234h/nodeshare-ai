package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	protov1 "github.com/distributed-ai/node-agent/api/proto/v1"
	"github.com/distributed-ai/node-agent/internal/hardware"
	"github.com/distributed-ai/node-agent/internal/inference"
	"github.com/distributed-ai/node-agent/internal/models"
)

type JobStatus string

const (
	StatusQueued    JobStatus = "QUEUED"
	StatusRunning   JobStatus = "RUNNING"
	StatusCompleted JobStatus = "COMPLETED"
	StatusFailed    JobStatus = "FAILED"
	StatusCancelled JobStatus = "CANCELLED"
	StatusRejected  JobStatus = "REJECTED"
)

type ActiveJob struct {
	Assignment        *protov1.JobAssignment
	Cancel            context.CancelFunc
	Status            JobStatus
	TokensGenerated   int
	ReservedVRAMBytes uint64
	ReservedRAMBytes  uint64
	StartedAt         time.Time
}

type JobEngine struct {
	mu                sync.RWMutex
	activeJobs        map[string]*ActiveJob
	reservedRAMBytes  uint64
	reservedVRAMBytes uint64
	maxConcurrent     int
	modelMgr          *models.ModelManager
	runtime           inference.InferenceRuntime
}

func NewJobEngine(maxConcurrent int, modelMgr *models.ModelManager, runtime inference.InferenceRuntime) *JobEngine {
	return &JobEngine{
		activeJobs:    make(map[string]*ActiveJob),
		maxConcurrent: maxConcurrent,
		modelMgr:      modelMgr,
		runtime:       runtime,
	}
}

func (je *JobEngine) SetRuntime(rt inference.InferenceRuntime) {
	je.mu.Lock()
	defer je.mu.Unlock()
	je.runtime = rt
}

func (je *JobEngine) ValidateAndReserve(assign *protov1.JobAssignment, hw *hardware.FullHardwareReport) (string, error) {
	je.mu.Lock()
	defer je.mu.Unlock()

	if len(je.activeJobs) >= je.maxConcurrent {
		return "RESOURCE_BUSY", fmt.Errorf("node is at maximum concurrent task capacity (%d)", je.maxConcurrent)
	}

	if _, exists := je.activeJobs[assign.JobId]; exists {
		return "DUPLICATE_ASSIGNMENT", fmt.Errorf("job %s is already running", assign.JobId)
	}

	// Check runtime health
	if je.runtime == nil {
		return "RUNTIME_UNAVAILABLE", fmt.Errorf("no inference runtime registered")
	}

	if err := je.runtime.Health(context.Background()); err != nil {
		return "RUNTIME_UNAVAILABLE", fmt.Errorf("runtime health check failed: %w", err)
	}

	// Check VRAM requirements
	if assign.RequiredVramBytes > 0 && hw.GPUReport != nil && hw.GPUReport.GPUAvailable {
		var maxFreeVRAM uint64
		for _, g := range hw.GPUReport.GPUs {
			if g.VRAMAvailBytes > maxFreeVRAM {
				maxFreeVRAM = g.VRAMAvailBytes
			}
		}

		availableVRAM := maxFreeVRAM
		if availableVRAM > je.reservedVRAMBytes {
			availableVRAM -= je.reservedVRAMBytes
		} else {
			availableVRAM = 0
		}

		if assign.RequiredVramBytes > availableVRAM {
			return "INSUFFICIENT_VRAM", fmt.Errorf("required VRAM %d bytes exceeds available unreserved VRAM %d bytes", assign.RequiredVramBytes, availableVRAM)
		}
	}

	// Reserve capacity
	je.reservedRAMBytes += assign.RequiredRamBytes
	je.reservedVRAMBytes += assign.RequiredVramBytes

	return "", nil
}

func (je *JobEngine) ExecuteJob(parentCtx context.Context, assign *protov1.JobAssignment, progressCallback func(chunk string, tokens int, status string, errStr string)) error {
	ctx, cancel := context.WithCancel(parentCtx)

	job := &ActiveJob{
		Assignment:        assign,
		Cancel:            cancel,
		Status:            StatusRunning,
		ReservedRAMBytes:  assign.RequiredRamBytes,
		ReservedVRAMBytes: assign.RequiredVramBytes,
		StartedAt:         time.Now(),
	}

	je.mu.Lock()
	je.activeJobs[assign.JobId] = job
	je.mu.Unlock()

	defer func() {
		je.mu.Lock()
		delete(je.activeJobs, assign.JobId)
		if je.reservedRAMBytes >= assign.RequiredRamBytes {
			je.reservedRAMBytes -= assign.RequiredRamBytes
		} else {
			je.reservedRAMBytes = 0
		}
		if je.reservedVRAMBytes >= assign.RequiredVramBytes {
			je.reservedVRAMBytes -= assign.RequiredVramBytes
		} else {
			je.reservedVRAMBytes = 0
		}
		je.mu.Unlock()
		cancel()
	}()

	genReq := &inference.GenerateRequest{
		JobID:       assign.JobId,
		ModelID:     assign.ModelId,
		Prompt:      assign.Payload,
		MaxTokens:   512,
		Temperature: 0.7,
	}

	err := je.runtime.StreamGenerate(ctx, genReq, func(chunk string, tokens int) error {
		job.TokensGenerated = tokens
		progressCallback(chunk, tokens, string(StatusRunning), "")
		return nil
	})

	if err != nil {
		if ctx.Err() == context.Canceled {
			job.Status = StatusCancelled
			progressCallback("", job.TokensGenerated, string(StatusCancelled), "job cancelled by controller")
			return nil
		}
		job.Status = StatusFailed
		progressCallback("", job.TokensGenerated, string(StatusFailed), err.Error())
		return err
	}

	if job.TokensGenerated == 0 {
		job.Status = StatusFailed
		err := fmt.Errorf("inference runtime completed without producing output")
		progressCallback("", 0, string(StatusFailed), err.Error())
		return err
	}

	job.Status = StatusCompleted
	progressCallback("", job.TokensGenerated, string(StatusCompleted), "")
	return nil
}

func (je *JobEngine) CancelJob(jobID string) bool {
	je.mu.Lock()
	defer je.mu.Unlock()

	job, ok := je.activeJobs[jobID]
	if ok && job.Cancel != nil {
		job.Cancel()
		job.Status = StatusCancelled
		return true
	}
	return false
}

func (je *JobEngine) GetActiveJobIDs() []string {
	je.mu.RLock()
	defer je.mu.RUnlock()

	ids := make([]string, 0, len(je.activeJobs))
	for k := range je.activeJobs {
		ids = append(ids, k)
	}
	return ids
}

func (je *JobEngine) ActiveJobsCount() int {
	je.mu.RLock()
	defer je.mu.RUnlock()
	return len(je.activeJobs)
}
