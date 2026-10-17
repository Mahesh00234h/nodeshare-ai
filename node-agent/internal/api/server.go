package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/distributed-ai/node-agent/internal/agent"
	"github.com/distributed-ai/node-agent/internal/hardware"
	"github.com/distributed-ai/node-agent/internal/identity"
	"github.com/distributed-ai/node-agent/internal/jobs"
	"github.com/distributed-ai/node-agent/internal/models"
)

type DiagnosticServer struct {
	port       int
	id         *identity.NodeIdentity
	sm         *agent.StateMachine
	jobEngine  *jobs.JobEngine
	modelMgr   *models.ModelManager
	httpServer *http.Server
	startTime  time.Time
}

func NewDiagnosticServer(
	port int,
	id *identity.NodeIdentity,
	sm *agent.StateMachine,
	jobEngine *jobs.JobEngine,
	modelMgr *models.ModelManager,
) *DiagnosticServer {
	if port <= 0 {
		port = 9090
	}
	return &DiagnosticServer{
		port:      port,
		id:        id,
		sm:        sm,
		jobEngine: jobEngine,
		modelMgr:  modelMgr,
		startTime: time.Now(),
	}
}

func (s *DiagnosticServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "UP",
			"state":   s.sm.GetState(),
			"node_id": s.id.NodeID,
		})
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hw, _ := hardware.DetectAll("1.0.0")

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"node_id":          s.id.NodeID,
			"agent_version":    "1.0.0",
			"state":            s.sm.GetState(),
			"uptime_seconds":   int64(time.Since(s.startTime).Seconds()),
			"active_jobs_count": s.jobEngine.ActiveJobsCount(),
			"active_job_ids":   s.jobEngine.GetActiveJobIDs(),
			"models_count":     len(s.modelMgr.ListModels()),
			"hardware":         hw,
		})
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hw, _ := hardware.DetectAll("1.0.0")

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"timestamp":        time.Now().Unix(),
			"uptime_seconds":   int64(time.Since(s.startTime).Seconds()),
			"cpu_utilization":  hw.CPU.Utilization,
			"ram_utilization":  hw.Memory.UtilizationPercent,
			"ram_used_bytes":   hw.Memory.UsedBytes,
			"ram_total_bytes":  hw.Memory.TotalBytes,
			"disk_free_bytes":  hw.Storage.AvailableBytes,
			"active_jobs":      s.jobEngine.ActiveJobsCount(),
		})
	})

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		_ = s.httpServer.ListenAndServe()
	}()
	return nil
}

func (s *DiagnosticServer) Stop() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(ctx)
	}
}
