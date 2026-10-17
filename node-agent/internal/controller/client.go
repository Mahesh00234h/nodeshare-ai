package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	protov1 "github.com/distributed-ai/node-agent/api/proto/v1"
	"github.com/distributed-ai/node-agent/internal/agent"
	"github.com/distributed-ai/node-agent/internal/config"
	"github.com/distributed-ai/node-agent/internal/hardware"
	"github.com/distributed-ai/node-agent/internal/identity"
	"github.com/distributed-ai/node-agent/internal/jobs"
	"github.com/distributed-ai/node-agent/internal/logging"
	"github.com/distributed-ai/node-agent/internal/models"
	"github.com/distributed-ai/node-agent/internal/protocol"
	"github.com/distributed-ai/node-agent/internal/security"
	"github.com/distributed-ai/node-agent/internal/state"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type ControllerClient struct {
	mu           sync.Mutex
	cfg          *config.Config
	id           *identity.NodeIdentity
	creds        *identity.NodeCredentials
	sm           *agent.StateMachine
	stateStore   *state.StateStore
	jobEngine    *jobs.JobEngine
	modelMgr     *models.ModelManager
	conn         *grpc.ClientConn
	grpcClient   protov1.NodeControllerServiceClient
	streamClient protov1.NodeControllerService_StreamControlChannelClient
	httpClient   *http.Client
	startTime    time.Time
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewControllerClient(
	cfg *config.Config,
	id *identity.NodeIdentity,
	creds *identity.NodeCredentials,
	sm *agent.StateMachine,
	stateStore *state.StateStore,
	jobEngine *jobs.JobEngine,
	modelMgr *models.ModelManager,
) *ControllerClient {
	ctx, cancel := context.WithCancel(context.Background())
	tr := &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{},
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}

	return &ControllerClient{
		cfg:        cfg,
		id:         id,
		creds:      creds,
		sm:         sm,
		stateStore: stateStore,
		jobEngine:  jobEngine,
		modelMgr:   modelMgr,
		httpClient: client,
		startTime:  time.Now(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (cc *ControllerClient) Start() error {
	go cc.reconnectLoop()
	return nil
}

func (cc *ControllerClient) Stop() {
	cc.cancel()
	cc.mu.Lock()
	if cc.conn != nil {
		_ = cc.conn.Close()
	}
	cc.mu.Unlock()
}

func (cc *ControllerClient) reconnectLoop() {
	logger := logging.GetLogger()
	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-cc.ctx.Done():
			return
		default:
		}

		_ = cc.sm.TransitionTo(agent.StateConnecting)
		logger.Info("Attempting gRPC controller connection", "url", cc.cfg.ControllerURL, "node_id", cc.id.NodeID)

		err := cc.connectAndRegister()
		if err == nil {
			// Connected and registered cleanly -> reset backoff
			backoff = 1 * time.Second
			_ = cc.sm.TransitionTo(agent.StateReady)
			logger.Info("Controller connected and registered successfully", "event", "controller_connected", "node_id", cc.id.NodeID)

			// Start background heartbeats, telemetry, and control stream loops
			streamErr := cc.runSessionLoops()
			logger.Warn("Controller connection lost or stream ended", "event", "controller_disconnected", "error", streamErr)
			_ = cc.sm.TransitionTo(agent.StateDisconnected)
		} else {
			logger.Warn("Failed to connect/register with controller", "error", err, "backoff", backoff.String())
		}

		// Exponential backoff with jitter
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		sleepTime := backoff + jitter

		select {
		case <-cc.ctx.Done():
			return
		case <-time.After(sleepTime):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (cc *ControllerClient) connectAndRegister() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	targetURL := cc.cfg.ControllerURL
	var credsOpt grpc.DialOption

	if strings.HasPrefix(targetURL, "https://") {
		targetURL = strings.TrimPrefix(targetURL, "https://")
		if !strings.Contains(targetURL, ":") {
			targetURL = targetURL + ":443"
		}
		credsOpt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else if strings.HasSuffix(targetURL, ":443") {
		credsOpt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else {
		credsOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	opts := []grpc.DialOption{credsOpt}

	conn, err := grpc.DialContext(cc.ctx, targetURL, opts...)
	if err != nil {
		return fmt.Errorf("gRPC dial failed: %w", err)
	}

	client := protov1.NewNodeControllerServiceClient(conn)
	cc.conn = conn
	cc.grpcClient = client

	_ = cc.sm.TransitionTo(agent.StateAuthenticating)

	// Step 1: Enroll if credentials not yet established
	if cc.creds == nil || cc.creds.AuthToken == "" {
		if cc.cfg.EnrollmentToken == "" {
			cc.cfg.EnrollmentToken = "dev-enrollment-token"
		}
		enrollReq := &protov1.EnrollmentRequest{
			NodeId:          cc.id.NodeID,
			EnrollmentToken: cc.cfg.EnrollmentToken,
			Hostname:        cc.id.NodeID,
			AgentVersion:    "1.0.0",
		}
		var enrollResp *protov1.EnrollmentResponse
		if strings.HasPrefix(cc.cfg.ControllerURL, "https://") {
			enrollResp = &protov1.EnrollmentResponse{}
			if httpErr := cc.httpPostProto("/node_controller.v1.NodeControllerService/EnrollNode", enrollReq, enrollResp); httpErr != nil {
				_ = conn.Close()
				return fmt.Errorf("enrollment failed: %v", httpErr)
			}
		} else {
			ctxEnroll, cancelEnroll := context.WithTimeout(cc.ctx, 3*time.Second)
			var err error
			enrollResp, err = client.EnrollNode(ctxEnroll, enrollReq)
			cancelEnroll()
			if err != nil {
				enrollResp = &protov1.EnrollmentResponse{}
				if httpErr := cc.httpPostProto("/node_controller.v1.NodeControllerService/EnrollNode", enrollReq, enrollResp); httpErr != nil {
					_ = conn.Close()
					return fmt.Errorf("enrollment failed: %w (http fallback: %v)", err, httpErr)
				}
			}
		}
		if !enrollResp.Success {
			_ = conn.Close()
			return fmt.Errorf("enrollment rejected: %s", enrollResp.Message)
		}

		creds := &identity.NodeCredentials{
			NodeID:     cc.id.NodeID,
			AuthToken:  enrollResp.AuthToken,
			NodeSecret: enrollResp.NodeSecret,
			EnrolledAt: time.Now().Format(time.RFC3339),
		}
		_ = identity.SaveCredentials(cc.cfg.IdentityDir, creds)
		cc.creds = creds
	}

	_ = cc.sm.TransitionTo(agent.StateRegistering)

	// Step 2: Register hardware and capabilities
	hwReport, err := hardware.DetectAll("1.0.0")
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("hardware detection failed: %w", err)
	}

	regReq := protocol.ToRegistrationRequest(hwReport, cc.id.NodeID, cc.creds.AuthToken)
	var regResp *protov1.RegistrationResponse
	if strings.HasPrefix(cc.cfg.ControllerURL, "https://") {
		regResp = &protov1.RegistrationResponse{}
		if httpErr := cc.httpPostProto("/node_controller.v1.NodeControllerService/RegisterNode", regReq, regResp); httpErr != nil {
			_ = conn.Close()
			return fmt.Errorf("node registration failed: %v", httpErr)
		}
	} else {
		ctxReg, cancelReg := context.WithTimeout(cc.ctx, 3*time.Second)
		var err error
		regResp, err = client.RegisterNode(ctxReg, regReq)
		cancelReg()
		if err != nil {
			regResp = &protov1.RegistrationResponse{}
			if httpErr := cc.httpPostProto("/node_controller.v1.NodeControllerService/RegisterNode", regReq, regResp); httpErr != nil {
				_ = conn.Close()
				return fmt.Errorf("node registration failed: %w (http fallback: %v)", err, httpErr)
			}
		}
	}

	if !regResp.Registered {
		_ = conn.Close()
		return fmt.Errorf("registration rejected by controller: %s", regResp.Message)
	}

	_ = cc.stateStore.UpdateState(string(agent.StateReady), cc.cfg.ControllerURL)
	return nil
}

func (cc *ControllerClient) runSessionLoops() error {
	sessionCtx, sessionCancel := context.WithCancel(cc.ctx)
	defer sessionCancel()
	logger := logging.GetLogger()

	// Open bidirectional streaming channel if native gRPC, otherwise use HTTP control stream poll loop
	if !strings.HasPrefix(cc.cfg.ControllerURL, "https://") {
		streamClient, err := cc.grpcClient.StreamControlChannel(sessionCtx)
		if err == nil {
			cc.streamClient = streamClient
			_ = streamClient.Send(&protov1.NodeMessage{
				NodeId:    cc.id.NodeID,
				MessageId: "stream-init",
			})
		} else {
			logger.Warn("gRPC StreamControlChannel unavailable (using HTTP fallback polling)", "error", err)
		}
	} else {
		logger.Info("HTTPS controller URL detected: using HTTP control stream polling for Render compatibility")
	}

	errChan := make(chan error, 3)

	// Heartbeat loop
	go func() {
		ticker := time.NewTicker(time.Duration(cc.cfg.HeartbeatIntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sessionCtx.Done():
				return
			case <-ticker.C:
				if err := cc.sendHeartbeat(sessionCtx); err != nil {
					logger.Warn("Heartbeat send warning", "error", err)
				}
			}
		}
	}()

	// Telemetry loop
	go func() {
		ticker := time.NewTicker(time.Duration(cc.cfg.TelemetryIntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sessionCtx.Done():
				return
			case <-ticker.C:
				if err := cc.sendTelemetry(sessionCtx); err != nil {
					logger.Warn("Telemetry send warning", "error", err)
				}
			}
		}
	}()

	// Stream receiver loop for incoming controller commands (or HTTP fallback polling)
	if cc.streamClient != nil {
		go func() {
			for {
				cc.mu.Lock()
				sc := cc.streamClient
				cc.mu.Unlock()
				if sc == nil {
					return
				}
				msg, err := sc.Recv()
				if err != nil {
					if err != io.EOF {
						logger.Warn("Control stream recv error", "error", err)
					}
					cc.mu.Lock()
					cc.streamClient = nil
					cc.mu.Unlock()
					return
				}
				cc.handleControllerMessage(sessionCtx, msg)
			}
		}()
	} else {
		// HTTP Fallback Polling Loop for edge proxies like Render
		go func() {
			logger.Info("Starting HTTP control stream poll loop (1s interval)")
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-sessionCtx.Done():
					return
				case <-ticker.C:
					pollReq := &protov1.NodeMessage{
						NodeId:    cc.id.NodeID,
						MessageId: "poll",
					}
					ctrlMsg := &protov1.ControllerMessage{}
					if err := cc.httpPostProto("/node_controller.v1.NodeControllerService/StreamControlChannel", pollReq, ctrlMsg); err == nil {
						if ctrlMsg.MessageId != "" && ctrlMsg.MessageId != "idle" && ctrlMsg.MessageId != "err-no-node-id" && ctrlMsg.MessageId != "ack" {
							logger.Info("HTTP poll received controller command", "msg_id", ctrlMsg.MessageId)
							cc.handleControllerMessage(sessionCtx, ctrlMsg)
						}
					}
				}
			}
		}()
	}

	select {
	case <-cc.ctx.Done():
		return nil
	case err := <-errChan:
		return err
	}
}

func (cc *ControllerClient) sendHeartbeat(ctx context.Context) error {
	hwReport, _ := hardware.DetectAll("1.0.0")

	// Evaluate safety thresholds
	safety := security.EvaluateHardwareSafety(hwReport, cc.cfg)
	if !safety.Safe {
		_ = cc.sm.TransitionTo(agent.StateDegraded)
	} else if cc.sm.GetState() == agent.StateDegraded {
		_ = cc.sm.TransitionTo(agent.StateReady)
	}

	var cpuUtil, ramUtil, gpuUtil, vramUtil float64
	var gpuTemp uint32

	if hwReport.CPU != nil {
		cpuUtil = hwReport.CPU.Utilization
	}
	if hwReport.Memory != nil {
		ramUtil = hwReport.Memory.UtilizationPercent
	}
	if hwReport.GPUReport != nil && len(hwReport.GPUReport.GPUs) > 0 {
		g := hwReport.GPUReport.GPUs[0]
		gpuUtil = g.UtilizationPercent
		if g.VRAMTotalBytes > 0 {
			vramUtil = (float64(g.VRAMUsedBytes) / float64(g.VRAMTotalBytes)) * 100.0
		}
		gpuTemp = g.TemperatureC
	}

	req := &protov1.HeartbeatRequest{
		NodeId:          cc.id.NodeID,
		AuthToken:       cc.creds.AuthToken,
		Timestamp:       time.Now().Unix(),
		UptimeSeconds:   int64(time.Since(cc.startTime).Seconds()),
		State:           string(cc.sm.GetState()),
		ActiveJobIds:    cc.jobEngine.GetActiveJobIDs(),
		CpuUtilization:  cpuUtil,
		RamUtilization:  ramUtil,
		GpuUtilization:  gpuUtil,
		VramUtilization: vramUtil,
		GpuTemperature:  gpuTemp,
		AgentVersion:    "1.0.0",
	}

	var resp *protov1.HeartbeatResponse
	if strings.HasPrefix(cc.cfg.ControllerURL, "https://") {
		resp = &protov1.HeartbeatResponse{}
		if err := cc.httpPostProto("/node_controller.v1.NodeControllerService/SendHeartbeat", req, resp); err != nil {
			return err
		}
	} else {
		ctxHb, cancelHb := context.WithTimeout(ctx, 3*time.Second)
		var err error
		resp, err = cc.grpcClient.SendHeartbeat(ctxHb, req)
		cancelHb()
		if err != nil {
			resp = &protov1.HeartbeatResponse{}
			if httpErr := cc.httpPostProto("/node_controller.v1.NodeControllerService/SendHeartbeat", req, resp); httpErr != nil {
				return err
			}
		}
	}

	if resp != nil {
		logger := logging.GetLogger()
		if resp.DesiredState == "DRAIN" {
			cc.HandleDrain(ctx, "Controller requested drain")
		} else if !resp.Acknowledged || resp.DesiredState == "RE_REGISTER" {
			logger.Warn("Heartbeat unacknowledged by controller, re-registering node", "desired_state", resp.DesiredState, "message", resp.Message)
			_ = cc.connectAndRegister()
		}
	}

	return nil
}

func (cc *ControllerClient) sendTelemetry(ctx context.Context) error {
	hwReport, _ := hardware.DetectAll("1.0.0")

	req := &protov1.TelemetryRequest{
		NodeId:        cc.id.NodeID,
		AuthToken:     cc.creds.AuthToken,
		Timestamp:     time.Now().Unix(),
		ActiveJobs:    int32(cc.jobEngine.ActiveJobsCount()),
		UptimeSeconds: int64(time.Since(cc.startTime).Seconds()),
	}

	if hwReport.CPU != nil {
		req.Cpu = &protov1.CPUInfoProto{
			Model:              hwReport.CPU.Model,
			Vendor:             hwReport.CPU.Vendor,
			Cores:              int32(hwReport.CPU.Cores),
			Threads:            int32(hwReport.CPU.Threads),
			UtilizationPercent: hwReport.CPU.Utilization,
		}
	}

	if hwReport.Memory != nil {
		req.Memory = &protov1.MemoryInfoProto{
			TotalBytes:         hwReport.Memory.TotalBytes,
			AvailableBytes:     hwReport.Memory.AvailableBytes,
			UsedBytes:          hwReport.Memory.UsedBytes,
			UtilizationPercent: hwReport.Memory.UtilizationPercent,
		}
	}

	if strings.HasPrefix(cc.cfg.ControllerURL, "https://") {
		resp := &protov1.TelemetryResponse{}
		return cc.httpPostProto("/node_controller.v1.NodeControllerService/SendTelemetry", req, resp)
	} else {
		ctxTelem, cancelTelem := context.WithTimeout(ctx, 3*time.Second)
		_, err := cc.grpcClient.SendTelemetry(ctxTelem, req)
		cancelTelem()
		if err != nil {
			resp := &protov1.TelemetryResponse{}
			_ = cc.httpPostProto("/node_controller.v1.NodeControllerService/SendTelemetry", req, resp)
		}
	}
	return nil
}

func (cc *ControllerClient) httpPostProto(endpoint string, reqMsg interface{}, respMsg interface{}) error {
	rawURL := cc.cfg.ControllerURL
	scheme := "http"
	if strings.HasPrefix(rawURL, "https://") || strings.HasSuffix(rawURL, ":443") {
		scheme = "https"
	}

	cleanHost := rawURL
	cleanHost = strings.TrimPrefix(cleanHost, "https://")
	cleanHost = strings.TrimPrefix(cleanHost, "http://")
	if scheme == "https" && strings.HasSuffix(cleanHost, ":443") {
		cleanHost = strings.TrimSuffix(cleanHost, ":443")
	}
	cleanHost = strings.TrimRight(cleanHost, "/")

	url := fmt.Sprintf("%s://%s%s", scheme, cleanHost, endpoint)

	reqBytes, err := json.Marshal(reqMsg)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(cc.ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := cc.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBytes, respMsg); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}
	return nil
}

func (cc *ControllerClient) handleControllerMessage(ctx context.Context, msg *protov1.ControllerMessage) {
	logger := logging.GetLogger()

	if msg.JobAssignment != nil {
		assign := msg.JobAssignment
		logger.Info("Received job assignment", "event", "job_received", "job_id", assign.JobId, "type", assign.JobType, "model_id", assign.ModelId)

		if cc.sm.GetState() == agent.StateDraining {
			logger.Warn("Rejecting job assignment while DRAINING", "job_id", assign.JobId)
			return
		}

		hwReport, _ := hardware.DetectAll("1.0.0")
		rejectReason, err := cc.jobEngine.ValidateAndReserve(assign, hwReport)
		if err != nil {
			logger.Warn("Rejected job assignment", "job_id", assign.JobId, "reason", rejectReason, "error", err)
			if rejectReason == "DUPLICATE_ASSIGNMENT" {
				logger.Info("Ignoring duplicate job assignment because the job is already running", "job_id", assign.JobId)
				return
			}
			_ = cc.sendNodeMessage(&protov1.NodeMessage{
				NodeId:    cc.id.NodeID,
				MessageId: msg.MessageId,
				JobProgress: &protov1.JobProgressUpdate{
					JobId:        assign.JobId,
					ExecutionId:  assign.ExecutionId,
					Status:       "REJECTED",
					ErrorMessage: fmt.Sprintf("[%s] %v", rejectReason, err),
				},
			})
			return
		}

		// Execute job in background goroutine
		go func() {
			_ = cc.sm.TransitionTo(agent.StateBusy)
			logger.Info("Executing job", "event", "job_started", "job_id", assign.JobId)

			err := cc.jobEngine.ExecuteJob(ctx, assign, func(chunk string, tokens int, status string, errStr string) {
				_ = cc.sendNodeMessage(&protov1.NodeMessage{
					NodeId:    cc.id.NodeID,
					MessageId: msg.MessageId,
					JobProgress: &protov1.JobProgressUpdate{
						JobId:           assign.JobId,
						ExecutionId:     assign.ExecutionId,
						Status:          status,
						OutputChunk:     chunk,
						TokensGenerated: int32(tokens),
						ErrorMessage:    errStr,
					},
				})
			})

			if err != nil {
				logger.Error("Job failed", "event", "job_failed", "job_id", assign.JobId, "error", err)
			} else {
				logger.Info("Job completed successfully", "event", "job_completed", "job_id", assign.JobId)
			}

			if cc.jobEngine.ActiveJobsCount() == 0 && cc.sm.GetState() == agent.StateBusy {
				_ = cc.sm.TransitionTo(agent.StateReady)
			}
		}()

	} else if msg.JobCancel != nil {
		cancel := msg.JobCancel
		logger.Info("Received job cancel request", "job_id", cancel.JobId, "reason", cancel.Reason)
		cc.jobEngine.CancelJob(cancel.JobId)

	} else if msg.ModelCommand != nil {
		cmd := msg.ModelCommand
		logger.Info("Received model command", "command", cmd.CommandType, "model_id", cmd.ModelId)
		if cmd.CommandType == "PREPARE" || cmd.CommandType == "DOWNLOAD" {
			_, _ = cc.modelMgr.CreatePlaceholderModel(cmd.ModelId, "1.0", cmd.ExpectedSizeBytes, cmd.ExpectedSha256)
		} else if cmd.CommandType == "DELETE" {
			_ = cc.modelMgr.DeleteModel(cmd.ModelId)
		}

	} else if msg.DrainCommand != nil {
		cc.HandleDrain(ctx, msg.DrainCommand.Reason)
	}
}

func (cc *ControllerClient) sendNodeMessage(msg *protov1.NodeMessage) error {
	cc.mu.Lock()
	stream := cc.streamClient
	cc.mu.Unlock()

	if stream != nil {
		return stream.Send(msg)
	}

	// HTTP Fallback Post
	resp := &protov1.ControllerMessage{}
	return cc.httpPostProto("/node_controller.v1.NodeControllerService/SendNodeMessage", msg, resp)
}

func (cc *ControllerClient) HandleDrain(ctx context.Context, reason string) {
	logger := logging.GetLogger()
	logger.Info("Initiating graceful node drain", "event", "node_draining", "reason", reason)

	_ = cc.sm.TransitionTo(agent.StateDraining)

	// Notify controller drain acknowledged
	_ = cc.sendNodeMessage(&protov1.NodeMessage{
		NodeId:            cc.id.NodeID,
		DrainAcknowledged: "DRAINED",
	})
}
