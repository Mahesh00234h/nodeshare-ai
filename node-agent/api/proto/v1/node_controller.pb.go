package protov1

import (
	"encoding/json"
)

// EnrollmentRequest represents node registration request using enrollment token.
type EnrollmentRequest struct {
	NodeId          string `json:"node_id,omitempty"`
	EnrollmentToken string `json:"enrollment_token,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	AgentVersion    string `json:"agent_version,omitempty"`
}

type EnrollmentResponse struct {
	Success    bool   `json:"success,omitempty"`
	AuthToken  string `json:"auth_token,omitempty"`
	NodeSecret string `json:"node_secret,omitempty"`
	Message    string `json:"message,omitempty"`
}

type CPUInfoProto struct {
	Model              string  `json:"model,omitempty"`
	Vendor             string  `json:"vendor,omitempty"`
	Cores              int32   `json:"cores,omitempty"`
	Threads            int32   `json:"threads,omitempty"`
	UtilizationPercent float64 `json:"utilization_percent,omitempty"`
}

type MemoryInfoProto struct {
	TotalBytes         uint64  `json:"total_bytes,omitempty"`
	AvailableBytes     uint64  `json:"available_bytes,omitempty"`
	UsedBytes          uint64  `json:"used_bytes,omitempty"`
	UtilizationPercent float64 `json:"utilization_percent,omitempty"`
}

type GPUInfoProto struct {
	Vendor             string  `json:"vendor,omitempty"`
	Name               string  `json:"name,omitempty"`
	VramTotalBytes     uint64  `json:"vram_total_bytes,omitempty"`
	VramUsedBytes      uint64  `json:"vram_used_bytes,omitempty"`
	VramAvailableBytes uint64  `json:"vram_available_bytes,omitempty"`
	UtilizationPercent float64 `json:"utilization_percent,omitempty"`
	TemperatureC       uint32  `json:"temperature_c,omitempty"`
	PowerUsageWatts    float64 `json:"power_usage_watts,omitempty"`
	DriverVersion      string  `json:"driver_version,omitempty"`
	ComputeCapability  string  `json:"compute_capability,omitempty"`
}

type StorageInfoProto struct {
	TotalBytes     uint64  `json:"total_bytes,omitempty"`
	AvailableBytes uint64  `json:"available_bytes,omitempty"`
	UsedBytes      uint64  `json:"used_bytes,omitempty"`
	UsedPercent    float64 `json:"used_percent,omitempty"`
}

type NetworkInterfaceProto struct {
	Name                  string   `json:"name,omitempty"`
	Ips                   []string `json:"ips,omitempty"`
	Mac                   string   `json:"mac,omitempty"`
	IsWired               bool     `json:"is_wired,omitempty"`
	EstimatedBandwidthBps uint64   `json:"estimated_bandwidth_bps,omitempty"`
}

type NetworkInfoProto struct {
	Interfaces         []*NetworkInterfaceProto `json:"interfaces,omitempty"`
	EstimatedBandwidth uint64                   `json:"estimated_bandwidth,omitempty"`
}

type RegistrationRequest struct {
	NodeId              string              `json:"node_id,omitempty"`
	AuthToken           string              `json:"auth_token,omitempty"`
	Hostname            string              `json:"hostname,omitempty"`
	Os                  string              `json:"os,omitempty"`
	OsVersion           string              `json:"os_version,omitempty"`
	Architecture        string              `json:"architecture,omitempty"`
	Cpu                 *CPUInfoProto       `json:"cpu,omitempty"`
	Memory              *MemoryInfoProto    `json:"memory,omitempty"`
	Gpus                []*GPUInfoProto     `json:"gpus,omitempty"`
	Storage             *StorageInfoProto   `json:"storage,omitempty"`
	Network             *NetworkInfoProto   `json:"network,omitempty"`
	AgentVersion        string              `json:"agent_version,omitempty"`
	RuntimeCapabilities []string            `json:"runtime_capabilities,omitempty"`
	ProtocolVersion     string              `json:"protocol_version,omitempty"`
}

type RegistrationResponse struct {
	Registered               bool   `json:"registered,omitempty"`
	ClusterNodeId            string `json:"cluster_node_id,omitempty"`
	HeartbeatIntervalSeconds int32  `json:"heartbeat_interval_seconds,omitempty"`
	TelemetryIntervalSeconds int32  `json:"telemetry_interval_seconds,omitempty"`
	Message                  string `json:"message,omitempty"`
}

type HeartbeatRequest struct {
	NodeId          string   `json:"node_id,omitempty"`
	AuthToken       string   `json:"auth_token,omitempty"`
	Timestamp       int64    `json:"timestamp,omitempty"`
	UptimeSeconds   int64    `json:"uptime_seconds,omitempty"`
	State           string   `json:"state,omitempty"`
	ActiveJobIds    []string `json:"active_job_ids,omitempty"`
	CpuUtilization  float64  `json:"cpu_utilization,omitempty"`
	RamUtilization  float64  `json:"ram_utilization,omitempty"`
	GpuUtilization  float64  `json:"gpu_utilization,omitempty"`
	VramUtilization float64  `json:"vram_utilization,omitempty"`
	GpuTemperature  uint32   `json:"gpu_temperature,omitempty"`
	AgentVersion    string   `json:"agent_version,omitempty"`
}

type HeartbeatResponse struct {
	Acknowledged bool   `json:"acknowledged,omitempty"`
	DesiredState string `json:"desired_state,omitempty"`
	Message      string `json:"message,omitempty"`
}

type TelemetryRequest struct {
	NodeId        string           `json:"node_id,omitempty"`
	AuthToken     string           `json:"auth_token,omitempty"`
	Timestamp     int64            `json:"timestamp,omitempty"`
	Cpu           *CPUInfoProto    `json:"cpu,omitempty"`
	Memory        *MemoryInfoProto `json:"memory,omitempty"`
	Gpus          []*GPUInfoProto  `json:"gpus,omitempty"`
	Storage       *StorageInfoProto `json:"storage,omitempty"`
	ActiveJobs    int32            `json:"active_jobs,omitempty"`
	QueuedJobs    int32            `json:"queued_jobs,omitempty"`
	UptimeSeconds int64            `json:"uptime_seconds,omitempty"`
}

type TelemetryResponse struct {
	Received bool `json:"received,omitempty"`
}

type JobAssignment struct {
	JobId             string `json:"job_id,omitempty"`
	JobType           string `json:"job_type,omitempty"`
	ModelId           string `json:"model_id,omitempty"`
	ExecutionId       string `json:"execution_id,omitempty"`
	PartitionId       string `json:"partition_id,omitempty"`
	Payload           string `json:"payload,omitempty"`
	DeadlineUnix      int64  `json:"deadline_unix,omitempty"`
	Priority          int32  `json:"priority,omitempty"`
	RequiredVramBytes uint64 `json:"required_vram_bytes,omitempty"`
	RequiredRamBytes  uint64 `json:"required_ram_bytes,omitempty"`
}

type JobCancel struct {
	JobId  string `json:"job_id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type ModelCommand struct {
	CommandType        string `json:"command_type,omitempty"`
	ModelId            string `json:"model_id,omitempty"`
	DownloadUrl        string `json:"download_url,omitempty"`
	ExpectedSha256     string `json:"expected_sha256,omitempty"`
	ExpectedSizeBytes uint64 `json:"expected_size_bytes,omitempty"`
}

type DrainCommand struct {
	Reason string `json:"reason,omitempty"`
	Force  bool   `json:"force,omitempty"`
}

type ControllerMessage struct {
	MessageId     string         `json:"message_id,omitempty"`
	JobAssignment *JobAssignment `json:"job_assignment,omitempty"`
	JobCancel     *JobCancel     `json:"job_cancel,omitempty"`
	ModelCommand  *ModelCommand  `json:"model_command,omitempty"`
	DrainCommand  *DrainCommand  `json:"drain_command,omitempty"`
}

type JobProgressUpdate struct {
	JobId           string `json:"job_id,omitempty"`
	ExecutionId     string `json:"execution_id,omitempty"`
	Status          string `json:"status,omitempty"`
	OutputChunk     string `json:"output_chunk,omitempty"`
	TokensGenerated int32  `json:"tokens_generated,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

type NodeMessage struct {
	NodeId            string             `json:"node_id,omitempty"`
	MessageId         string             `json:"message_id,omitempty"`
	JobProgress       *JobProgressUpdate `json:"job_progress,omitempty"`
	DrainAcknowledged string             `json:"drain_acknowledged,omitempty"`
	ErrorReport       string             `json:"error_report,omitempty"`
}

func (*EnrollmentRequest) Reset()                  {}
func (*EnrollmentRequest) String() string          { return "" }
func (*EnrollmentRequest) ProtoMessage()           {}

func (*EnrollmentResponse) Reset()                 {}
func (*EnrollmentResponse) String() string         { return "" }
func (*EnrollmentResponse) ProtoMessage()          {}

func (*RegistrationRequest) Reset()                {}
func (*RegistrationRequest) String() string        { return "" }
func (*RegistrationRequest) ProtoMessage()         {}

func (*RegistrationResponse) Reset()               {}
func (*RegistrationResponse) String() string       { return "" }
func (*RegistrationResponse) ProtoMessage()        {}

func (*HeartbeatRequest) Reset()                   {}
func (*HeartbeatRequest) String() string           { return "" }
func (*HeartbeatRequest) ProtoMessage()            {}

func (*HeartbeatResponse) Reset()                  {}
func (*HeartbeatResponse) String() string          { return "" }
func (*HeartbeatResponse) ProtoMessage()           {}

func (*TelemetryRequest) Reset()                   {}
func (*TelemetryRequest) String() string           { return "" }
func (*TelemetryRequest) ProtoMessage()            {}

func (*TelemetryResponse) Reset()                  {}
func (*TelemetryResponse) String() string          { return "" }
func (*TelemetryResponse) ProtoMessage()           {}

func (*NodeMessage) Reset()                        {}
func (*NodeMessage) String() string                { return "" }
func (*NodeMessage) ProtoMessage()                 {}

func (*ControllerMessage) Reset()                  {}
func (*ControllerMessage) String() string          { return "" }
func (*ControllerMessage) ProtoMessage()           {}

func (*CPUInfoProto) Reset()                       {}
func (*CPUInfoProto) String() string               { return "" }
func (*CPUInfoProto) ProtoMessage()                {}

func (*MemoryInfoProto) Reset()                    {}
func (*MemoryInfoProto) String() string            { return "" }
func (*MemoryInfoProto) ProtoMessage()             {}

func (*GPUInfoProto) Reset()                       {}
func (*GPUInfoProto) String() string               { return "" }
func (*GPUInfoProto) ProtoMessage()                {}

func (*StorageInfoProto) Reset()                   {}
func (*StorageInfoProto) String() string            { return "" }
func (*StorageInfoProto) ProtoMessage()             {}

func (*NetworkInterfaceProto) Reset()              {}
func (*NetworkInterfaceProto) String() string      { return "" }
func (*NetworkInterfaceProto) ProtoMessage()       {}

func (*NetworkInfoProto) Reset()                   {}
func (*NetworkInfoProto) String() string           { return "" }
func (*NetworkInfoProto) ProtoMessage()            {}

func (*JobAssignment) Reset()                      {}
func (*JobAssignment) String() string              { return "" }
func (*JobAssignment) ProtoMessage()               {}

func (*JobCancel) Reset()                          {}
func (*JobCancel) String() string                  { return "" }
func (*JobCancel) ProtoMessage()                   {}

func (*ModelCommand) Reset()                       {}
func (*ModelCommand) String() string               { return "" }
func (*ModelCommand) ProtoMessage()                {}

func (*DrainCommand) Reset()                       {}
func (*DrainCommand) String() string               { return "" }
func (*DrainCommand) ProtoMessage()                {}

func (*JobProgressUpdate) Reset()                  {}
func (*JobProgressUpdate) String() string          { return "" }
func (*JobProgressUpdate) ProtoMessage()           {}

func (m *EnrollmentRequest) Marshal() ([]byte, error)       { return json.Marshal(m) }
func (m *EnrollmentRequest) Unmarshal(b []byte) error      { return json.Unmarshal(b, m) }
func (m *EnrollmentResponse) Marshal() ([]byte, error)      { return json.Marshal(m) }
func (m *EnrollmentResponse) Unmarshal(b []byte) error     { return json.Unmarshal(b, m) }
func (m *RegistrationRequest) Marshal() ([]byte, error)     { return json.Marshal(m) }
func (m *RegistrationRequest) Unmarshal(b []byte) error    { return json.Unmarshal(b, m) }
func (m *RegistrationResponse) Marshal() ([]byte, error)    { return json.Marshal(m) }
func (m *RegistrationResponse) Unmarshal(b []byte) error   { return json.Unmarshal(b, m) }
func (m *HeartbeatRequest) Marshal() ([]byte, error)        { return json.Marshal(m) }
func (m *HeartbeatRequest) Unmarshal(b []byte) error       { return json.Unmarshal(b, m) }
func (m *HeartbeatResponse) Marshal() ([]byte, error)       { return json.Marshal(m) }
func (m *HeartbeatResponse) Unmarshal(b []byte) error      { return json.Unmarshal(b, m) }
func (m *TelemetryRequest) Marshal() ([]byte, error)        { return json.Marshal(m) }
func (m *TelemetryRequest) Unmarshal(b []byte) error       { return json.Unmarshal(b, m) }
func (m *TelemetryResponse) Marshal() ([]byte, error)       { return json.Marshal(m) }
func (m *TelemetryResponse) Unmarshal(b []byte) error      { return json.Unmarshal(b, m) }
func (m *NodeMessage) Marshal() ([]byte, error)             { return json.Marshal(m) }
func (m *NodeMessage) Unmarshal(b []byte) error            { return json.Unmarshal(b, m) }
func (m *ControllerMessage) Marshal() ([]byte, error)       { return json.Marshal(m) }
func (m *ControllerMessage) Unmarshal(b []byte) error      { return json.Unmarshal(b, m) }
func (m *CPUInfoProto) Marshal() ([]byte, error)            { return json.Marshal(m) }
func (m *CPUInfoProto) Unmarshal(b []byte) error           { return json.Unmarshal(b, m) }
func (m *MemoryInfoProto) Marshal() ([]byte, error)         { return json.Marshal(m) }
func (m *MemoryInfoProto) Unmarshal(b []byte) error        { return json.Unmarshal(b, m) }
func (m *GPUInfoProto) Marshal() ([]byte, error)            { return json.Marshal(m) }
func (m *GPUInfoProto) Unmarshal(b []byte) error           { return json.Unmarshal(b, m) }
func (m *StorageInfoProto) Marshal() ([]byte, error)        { return json.Marshal(m) }
func (m *StorageInfoProto) Unmarshal(b []byte) error       { return json.Unmarshal(b, m) }
func (m *NetworkInterfaceProto) Marshal() ([]byte, error)   { return json.Marshal(m) }
func (m *NetworkInterfaceProto) Unmarshal(b []byte) error  { return json.Unmarshal(b, m) }
func (m *NetworkInfoProto) Marshal() ([]byte, error)        { return json.Marshal(m) }
func (m *NetworkInfoProto) Unmarshal(b []byte) error       { return json.Unmarshal(b, m) }
func (m *JobAssignment) Marshal() ([]byte, error)           { return json.Marshal(m) }
func (m *JobAssignment) Unmarshal(b []byte) error          { return json.Unmarshal(b, m) }
func (m *JobCancel) Marshal() ([]byte, error)               { return json.Marshal(m) }
func (m *JobCancel) Unmarshal(b []byte) error              { return json.Unmarshal(b, m) }
func (m *ModelCommand) Marshal() ([]byte, error)            { return json.Marshal(m) }
func (m *ModelCommand) Unmarshal(b []byte) error           { return json.Unmarshal(b, m) }
func (m *DrainCommand) Marshal() ([]byte, error)            { return json.Marshal(m) }
func (m *DrainCommand) Unmarshal(b []byte) error           { return json.Unmarshal(b, m) }
func (m *JobProgressUpdate) Marshal() ([]byte, error)       { return json.Marshal(m) }
func (m *JobProgressUpdate) Unmarshal(b []byte) error      { return json.Unmarshal(b, m) }

func (x *CPUInfoProto) GetModel() string        { if x != nil { return x.Model }; return "" }
func (x *CPUInfoProto) GetCores() int32         { if x != nil { return x.Cores }; return 0 }
func (x *MemoryInfoProto) GetTotalBytes() uint64 { if x != nil { return x.TotalBytes }; return 0 }

