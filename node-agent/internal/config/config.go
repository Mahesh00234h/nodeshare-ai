package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
)

type Config struct {
	mu sync.RWMutex

	ControllerURL            string  `json:"controller_url"`
	EnrollmentToken          string  `json:"enrollment_token,omitempty"`
	NodeDataDir              string  `json:"node_data_dir"`
	ModelDir                 string  `json:"model_dir"`
	LogDir                   string  `json:"log_dir"`
	CacheDir                 string  `json:"cache_dir"`
	StateDir                 string  `json:"state_dir"`
	IdentityDir              string  `json:"identity_dir"`
	TelemetryIntervalSeconds int     `json:"telemetry_interval_seconds"`
	HeartbeatIntervalSeconds int     `json:"heartbeat_interval_seconds"`
	MaxConcurrentTasks       int     `json:"max_concurrent_tasks"`
	LogLevel                 string  `json:"log_level"`
	LocalAPIPort             int     `json:"local_api_port"`
	MaxGPUTempC              float64 `json:"max_gpu_temp_c"`
	MaxRAMUsagePercent       float64 `json:"max_ram_usage_percent"`
	MinFreeDiskGB            float64 `json:"min_free_disk_gb"`
	TLSEnabled               bool    `json:"tls_enabled"`
	TLSSkipVerify            bool    `json:"tls_skip_verify"` // For dev/mock testing
}

// DefaultConfig returns reasonable default configuration based on host OS.
func DefaultConfig() *Config {
	var baseDataDir string
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		baseDataDir = filepath.Join(programData, "DistributedAI")
	} else {
		baseDataDir = "/var/lib/distributed-ai"
	}

	return &Config{
		ControllerURL:            "https://llm-cluster.onrender.com:443",
		NodeDataDir:              baseDataDir,
		ModelDir:                 filepath.Join(baseDataDir, "models"),
		LogDir:                   filepath.Join(baseDataDir, "logs"),
		CacheDir:                 filepath.Join(baseDataDir, "cache"),
		StateDir:                 filepath.Join(baseDataDir, "state"),
		IdentityDir:              filepath.Join(baseDataDir, "identity"),
		TelemetryIntervalSeconds: 5,
		HeartbeatIntervalSeconds: 5,
		MaxConcurrentTasks:       4,
		LogLevel:                 "INFO",
		LocalAPIPort:             9090,
		MaxGPUTempC:              85.0,
		MaxRAMUsagePercent:       95.0,
		MinFreeDiskGB:            2.0,
		TLSEnabled:               false,
		TLSSkipVerify:            false,
	}
}

// LoadConfig loads configuration from default file or path if specified, overlaying environment variables.
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Environment overrides
	if val := os.Getenv("CONTROLLER_URL"); val != "" {
		cfg.ControllerURL = val
	}
	if val := os.Getenv("ENROLLMENT_TOKEN"); val != "" {
		cfg.EnrollmentToken = val
	}
	if val := os.Getenv("NODE_DATA_DIR"); val != "" {
		cfg.NodeDataDir = val
		cfg.ModelDir = filepath.Join(val, "models")
		cfg.LogDir = filepath.Join(val, "logs")
		cfg.CacheDir = filepath.Join(val, "cache")
		cfg.StateDir = filepath.Join(val, "state")
		cfg.IdentityDir = filepath.Join(val, "identity")
	}
	if val := os.Getenv("MODEL_DIR"); val != "" {
		cfg.ModelDir = val
	}
	if val := os.Getenv("LOG_DIR"); val != "" {
		cfg.LogDir = val
	}
	if val := os.Getenv("TELEMETRY_INTERVAL_SECONDS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.TelemetryIntervalSeconds = i
		}
	}
	if val := os.Getenv("HEARTBEAT_INTERVAL_SECONDS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.HeartbeatIntervalSeconds = i
		}
	}
	if val := os.Getenv("MAX_CONCURRENT_TASKS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.MaxConcurrentTasks = i
		}
	}
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}
	if val := os.Getenv("LOCAL_API_PORT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.LocalAPIPort = i
		}
	}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.NodeDataDir,
		c.ModelDir,
		c.LogDir,
		c.CacheDir,
		c.StateDir,
		c.IdentityDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
