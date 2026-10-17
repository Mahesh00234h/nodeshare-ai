package unit

import (
	"context"
	"testing"
	"time"

	protov1 "github.com/distributed-ai/node-agent/api/proto/v1"
	"github.com/distributed-ai/node-agent/internal/hardware"
	"github.com/distributed-ai/node-agent/internal/inference"
	"github.com/distributed-ai/node-agent/internal/jobs"
)

func TestJobValidationAndResourceReservation(t *testing.T) {
	mockRuntime := inference.NewMockInferenceRuntime()
	jobEngine := jobs.NewJobEngine(2, nil, mockRuntime)

	hwReport, _ := hardware.DetectAll("1.0.0")

	// Job 1: Normal job requiring 512MB RAM
	job1 := &protov1.JobAssignment{
		JobId:             "job-1",
		JobType:           "INFERENCE",
		ModelId:           "mock-model",
		RequiredRamBytes:  512 * 1024 * 1024,
		RequiredVramBytes: 0,
	}

	reason, err := jobEngine.ValidateAndReserve(job1, hwReport)
	if err != nil {
		t.Fatalf("Job 1 reservation failed: %v (reason: %s)", err, reason)
	}

	// Job 2: Execute Job 1 and verify token streaming
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var chunks []string
	err = jobEngine.ExecuteJob(ctx, job1, func(chunk string, tokens int, status string, errStr string) {
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	})

	if err != nil {
		t.Fatalf("Job 1 execution failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatalf("Expected streamed output chunks from mock runtime")
	}
}
