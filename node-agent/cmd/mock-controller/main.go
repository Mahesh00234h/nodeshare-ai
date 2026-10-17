package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	protov1 "github.com/distributed-ai/node-agent/api/proto/v1"
	"google.golang.org/grpc"
)

type MockControllerServer struct {
	protov1.UnimplementedNodeControllerServiceServer
	mu            sync.Mutex
	enrolledTokens map[string]string
	registeredNodes map[string]*protov1.RegistrationRequest
	streams        map[string]protov1.NodeControllerService_StreamControlChannelServer
}

func NewMockControllerServer() *MockControllerServer {
	return &MockControllerServer{
		enrolledTokens:  make(map[string]string),
		registeredNodes: make(map[string]*protov1.RegistrationRequest),
		streams:         make(map[string]protov1.NodeControllerService_StreamControlChannelServer),
	}
}

func (s *MockControllerServer) EnrollNode(ctx context.Context, req *protov1.EnrollmentRequest) (*protov1.EnrollmentResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Printf("[MOCK CONTROLLER] Received enrollment request for node_id=%s, token=%s, host=%s\n", req.NodeId, req.EnrollmentToken, req.Hostname)

	authToken := fmt.Sprintf("auth-token-%s", req.NodeId[:8])
	s.enrolledTokens[req.NodeId] = authToken

	return &protov1.EnrollmentResponse{
		Success:    true,
		AuthToken:  authToken,
		NodeSecret: "mock-secret-key-12345",
		Message:    "Enrollment successful",
	}, nil
}

func (s *MockControllerServer) RegisterNode(ctx context.Context, req *protov1.RegistrationRequest) (*protov1.RegistrationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Printf("[MOCK CONTROLLER] Node registered: ID=%s, Host=%s, OS=%s, CPU=%s, Cores=%d, RAM=%.2f GB, GPUs=%d\n",
		req.NodeId, req.Hostname, req.Os, req.Cpu.GetModel(), req.Cpu.GetCores(),
		float64(req.Memory.GetTotalBytes())/(1024*1024*1024), len(req.Gpus))

	s.registeredNodes[req.NodeId] = req

	return &protov1.RegistrationResponse{
		Registered:               true,
		ClusterNodeId:            fmt.Sprintf("cluster-node-%s", req.NodeId[:6]),
		HeartbeatIntervalSeconds: 5,
		TelemetryIntervalSeconds: 5,
		Message:                  "Node successfully registered in cluster",
	}, nil
}

func (s *MockControllerServer) SendHeartbeat(ctx context.Context, req *protov1.HeartbeatRequest) (*protov1.HeartbeatResponse, error) {
	fmt.Printf("[MOCK CONTROLLER] Heartbeat from %s: State=%s, CPU=%.1f%%, RAM=%.1f%%, GPU=%.1f%%, Temp=%d°C, Uptime=%ds\n",
		req.NodeId[:8], req.State, req.CpuUtilization, req.RamUtilization, req.GpuUtilization, req.GpuTemperature, req.UptimeSeconds)

	return &protov1.HeartbeatResponse{
		Acknowledged: true,
		DesiredState: "READY",
		Message:      "Heartbeat ACK",
	}, nil
}

func (s *MockControllerServer) SendTelemetry(ctx context.Context, req *protov1.TelemetryRequest) (*protov1.TelemetryResponse, error) {
	fmt.Printf("[MOCK CONTROLLER] Telemetry from %s: ActiveJobs=%d, Uptime=%ds\n", req.NodeId[:8], req.ActiveJobs, req.UptimeSeconds)
	return &protov1.TelemetryResponse{Received: true}, nil
}

func (s *MockControllerServer) StreamControlChannel(stream protov1.NodeControllerService_StreamControlChannelServer) error {
	var nodeID string

	// Read initial message to bind nodeID stream
	inMsg, err := stream.Recv()
	if err != nil {
		return err
	}

	nodeID = inMsg.NodeId
	s.mu.Lock()
	s.streams[nodeID] = stream
	s.mu.Unlock()

	fmt.Printf("[MOCK CONTROLLER] Control channel opened for node %s\n", nodeID)

	// Send a test inference job automatically after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Printf("[MOCK CONTROLLER] Dispatching test inference job to node %s...\n", nodeID)
		_ = stream.Send(&protov1.ControllerMessage{
			MessageId: "msg-job-101",
			JobAssignment: &protov1.JobAssignment{
				JobId:             "job-test-001",
				JobType:           "INFERENCE",
				ModelId:           "mock-llama-3",
				ExecutionId:       "exec-101",
				PartitionId:       "partition-0",
				Payload:           "What is the capital of distributed AI computing?",
				Priority:          1,
				RequiredRamBytes:  1024 * 1024 * 512, // 512MB
				RequiredVramBytes: 0,
			},
		})
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if msg.JobProgress != nil {
			prog := msg.JobProgress
			if prog.OutputChunk != "" {
				fmt.Printf("[MOCK CONTROLLER] Streamed token from job %s: %s\n", prog.JobId, prog.OutputChunk)
			} else {
				fmt.Printf("[MOCK CONTROLLER] Job %s status update: %s (Tokens=%d, Error=%s)\n", prog.JobId, prog.Status, prog.TokensGenerated, prog.ErrorMessage)
			}
		} else if msg.DrainAcknowledged != "" {
			fmt.Printf("[MOCK CONTROLLER] Node %s drain acknowledged: %s\n", msg.NodeId, msg.DrainAcknowledged)
		}
	}

	s.mu.Lock()
	delete(s.streams, nodeID)
	s.mu.Unlock()
	fmt.Printf("[MOCK CONTROLLER] Control channel closed for node %s\n", nodeID)
	return nil
}

func main() {
	port := 50051
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Printf("Failed to listen on port %d: %v\n", port, err)
		return
	}

	grpcServer := grpc.NewServer()
	mockSvr := NewMockControllerServer()
	protov1.RegisterNodeControllerServiceServer(grpcServer, mockSvr)

	fmt.Printf("=== DISTRIBUTED AI MOCK CONTROLLER STARTED ===\nListening on gRPC port :%d\nPress Ctrl+C to terminate.\n\n", port)
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Mock controller server error: %v\n", err)
	}
}
