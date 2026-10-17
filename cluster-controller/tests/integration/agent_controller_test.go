package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	protov1 "github.com/distributed-ai/node-agent/api/proto/v1"
	"github.com/distributed-ai/node-agent/internal/agent"
	"github.com/distributed-ai/node-agent/internal/config"
	"github.com/distributed-ai/node-agent/internal/controller"
	"github.com/distributed-ai/node-agent/internal/identity"
	"github.com/distributed-ai/node-agent/internal/inference"
	"github.com/distributed-ai/node-agent/internal/jobs"
	"github.com/distributed-ai/node-agent/internal/models"
	"github.com/distributed-ai/node-agent/internal/state"
	"google.golang.org/grpc"
)

type mockTestServer struct {
	protov1.UnimplementedNodeControllerServiceServer
	enrolled   bool
	registered bool
	heartbeats int
}

func (s *mockTestServer) EnrollNode(ctx context.Context, req *protov1.EnrollmentRequest) (*protov1.EnrollmentResponse, error) {
	s.enrolled = true
	return &protov1.EnrollmentResponse{Success: true, AuthToken: "test-token-123"}, nil
}

func (s *mockTestServer) RegisterNode(ctx context.Context, req *protov1.RegistrationRequest) (*protov1.RegistrationResponse, error) {
	s.registered = true
	return &protov1.RegistrationResponse{Registered: true, HeartbeatIntervalSeconds: 1, TelemetryIntervalSeconds: 1}, nil
}

func (s *mockTestServer) SendHeartbeat(ctx context.Context, req *protov1.HeartbeatRequest) (*protov1.HeartbeatResponse, error) {
	s.heartbeats++
	return &protov1.HeartbeatResponse{Acknowledged: true}, nil
}

func (s *mockTestServer) SendTelemetry(ctx context.Context, req *protov1.TelemetryRequest) (*protov1.TelemetryResponse, error) {
	return &protov1.TelemetryResponse{Received: true}, nil
}

func (s *mockTestServer) StreamControlChannel(stream protov1.NodeControllerService_StreamControlChannelServer) error {
	for {
		_, err := stream.Recv()
		if err != nil {
			return err
		}
	}
}

func TestAgentControllerIntegration(t *testing.T) {
	// Step 1: Start Mock Server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer()
	mockSvc := &mockTestServer{}
	protov1.RegisterNodeControllerServiceServer(grpcServer, mockSvc)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	// Step 2: Configure Agent
	tempDir, err := os.MkdirTemp("", "agent_integration_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	cfg.ControllerURL = lis.Addr().String()
	cfg.NodeDataDir = tempDir
	cfg.HeartbeatIntervalSeconds = 1
	cfg.TelemetryIntervalSeconds = 1
	_ = cfg.EnsureDirectories()

	id, err := identity.LoadOrCreateIdentity(cfg.IdentityDir)
	if err != nil {
		t.Fatalf("Identity creation failed: %v", err)
	}

	stateStore, _ := state.NewStateStore(cfg.StateDir)
	sm := agent.NewStateMachine(agent.StateInitializing, nil)
	modelMgr, _ := models.NewModelManager(cfg.ModelDir)
	mockRuntime := inference.NewMockInferenceRuntime()
	jobEngine := jobs.NewJobEngine(2, modelMgr, mockRuntime)

	client := controller.NewControllerClient(cfg, id, nil, sm, stateStore, jobEngine, modelMgr)
	_ = client.Start()
	defer client.Stop()

	// Step 3: Wait for enrollment and registration
	fmt.Println("[TEST] Waiting for node agent enrollment and heartbeats...")
	time.Sleep(3 * time.Second)

	if !mockSvc.enrolled {
		t.Errorf("Agent failed to enroll with mock controller")
	}
	if !mockSvc.registered {
		t.Errorf("Agent failed to register hardware with mock controller")
	}
	if mockSvc.heartbeats == 0 {
		t.Errorf("No heartbeats received from agent")
	}
	if sm.GetState() != agent.StateReady {
		t.Errorf("Expected agent state READY, got %s", sm.GetState())
	}
}
