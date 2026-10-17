package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/distributed-ai/node-agent/internal/agent"
	"github.com/distributed-ai/node-agent/internal/api"
	"github.com/distributed-ai/node-agent/internal/config"
	"github.com/distributed-ai/node-agent/internal/controller"
	"github.com/distributed-ai/node-agent/internal/identity"
	"github.com/distributed-ai/node-agent/internal/inference"
	"github.com/distributed-ai/node-agent/internal/jobs"
	"github.com/distributed-ai/node-agent/internal/logging"
	"github.com/distributed-ai/node-agent/internal/models"
	"github.com/distributed-ai/node-agent/internal/service"
	"github.com/distributed-ai/node-agent/internal/state"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration JSON file")
	controllerURL := flag.String("controller-url", "", "Controller URL (e.g. https://llm-cluster.onrender.com:443)")
	enrollmentToken := flag.String("enrollment-token", "", "Enrollment token for node registration")
	foreground := flag.Bool("foreground", false, "Run in foreground mode (for dev/debugging)")
	flag.Bool("f", false, "Alias for --foreground")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		writeStartupError(fmt.Errorf("configuration initialization failed: %w", err))
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	if *controllerURL != "" {
		cfg.ControllerURL = *controllerURL
	}
	if *enrollmentToken != "" {
		cfg.EnrollmentToken = *enrollmentToken
	}

	// Initialize Logger
	logger, err := logging.InitLogger(cfg.LogDir, cfg.LogLevel, *foreground)
	if err != nil {
		writeStartupError(fmt.Errorf("logger initialization failed: %w", err))
		fmt.Printf("Error initializing logger: %v\n", err)
		return
	}

	isSvc, _ := service.IsWindowsService()
	if isSvc && !*foreground {
		err := service.RunWindowsService("DistributedAINodeAgent", func(ctx context.Context) error {
			return runAgent(ctx, cfg)
		}, false)
		if err != nil {
			logger.Error("Windows service execution failed", "error", err)
		}
		return
	}

	// Foreground mode
	logger.Info("Starting Distributed AI Node Agent in foreground mode", "event", "agent_started")
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		service.WaitForShutdownSignal(cancel)
	}()

	if err := runAgent(ctx, cfg); err != nil {
		logger.Error("Node Agent stopped with error", "error", err)
	}
}

func writeStartupError(err error) {
	path := `C:\ProgramData\DistributedAI\logs\startup-error.log`
	if makeErr := os.MkdirAll(filepath.Dir(path), 0755); makeErr != nil {
		return
	}
	file, openErr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if openErr != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "%s %v\n", time.Now().Format(time.RFC3339), err)
}

func runAgent(ctx context.Context, cfg *config.Config) error {
	logger := logging.GetLogger()

	// 1. Identity
	id, err := identity.LoadOrCreateIdentity(cfg.IdentityDir)
	if err != nil {
		return fmt.Errorf("failed to load/create identity: %w", err)
	}
	logger.Info("Loaded persistent Node Identity", "node_id", id.NodeID)

	creds, _ := identity.LoadCredentials(cfg.IdentityDir)

	// 2. State Store
	stateStore, err := state.NewStateStore(cfg.StateDir)
	if err != nil {
		return fmt.Errorf("failed to initialize state store: %w", err)
	}

	// 3. State Machine
	sm := agent.NewStateMachine(agent.StateInitializing, func(from, to agent.NodeState) {
		logger.Info("State transition", "from", from, "to", to)
	})

	// 4. Model Manager
	modelMgr, err := models.NewModelManager(cfg.ModelDir)
	if err != nil {
		return fmt.Errorf("failed to initialize model manager: %w", err)
	}

	// 5. Inference Runtime (Use Distributed AI Inference Runtime v1)
	var rt inference.InferenceRuntime
	distAdapter, err := inference.NewDistributedRuntimeAdapter(cfg.ModelDir, id.NodeID, 50052)
	if err == nil {
		logger.Info("Using DistributedInferenceRuntime (llama.cpp RPC engine)", "event", "runtime_started", "node_id", id.NodeID)
		rt = distAdapter
	} else {
		logger.Info("Using MockInferenceRuntime fallback", "event", "runtime_started")
		rt = inference.NewMockInferenceRuntime()
	}
	_ = rt.Start(ctx)

	// 6. Job Engine
	jobEngine := jobs.NewJobEngine(cfg.MaxConcurrentTasks, modelMgr, rt)

	// 7. Local Diagnostic Server
	diagServer := api.NewDiagnosticServer(cfg.LocalAPIPort, id, sm, jobEngine, modelMgr)
	_ = diagServer.Start()
	defer diagServer.Stop()

	// 8. Controller Client
	client := controller.NewControllerClient(cfg, id, creds, sm, stateStore, jobEngine, modelMgr)
	_ = client.Start()
	defer client.Stop()

	<-ctx.Done()

	logger.Info("Initiating Node Agent graceful shutdown", "event", "node_draining")
	client.HandleDrain(context.Background(), "OS service stopping")
	_ = sm.TransitionTo(agent.StateStopping)
	_ = rt.Stop(context.Background())

	logger.Info("Node Agent shutdown complete")
	return nil
}
