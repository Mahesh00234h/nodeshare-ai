package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type GGUFModelManifest struct {
	ModelID              string `json:"model_id"`
	ModelVersion         string `json:"model_version"`
	Architecture         string `json:"architecture"`
	Quantization         string `json:"quantization"`
	FileSizeBytes        int64  `json:"file_size_bytes"`
	ParameterCount       string `json:"parameter_count"`
	ContextLength        int    `json:"context_length"`
	TotalLayers          int    `json:"total_layers"`
	RequiredVRAMEstimate uint64 `json:"required_vram_estimate_bytes"`
	SupportedRuntime     string `json:"supported_runtime"`
	ExpectedSHA256       string `json:"expected_sha256"`
	DownloadURL          string `json:"download_url"`
	LocalPath            string `json:"local_path"`
}

func VerifyModelFileSHA256(filePath string, expectedSHA256 string) (bool, error) {
	if expectedSHA256 == "" {
		return true, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open model file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, fmt.Errorf("failed to compute file checksum: %w", err)
	}

	calculatedHex := hex.EncodeToString(hasher.Sum(nil))
	if calculatedHex != expectedSHA256 {
		return false, fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, calculatedHex)
	}

	return true, nil
}

func InspectGGUFFile(filePath string) (*GGUFModelManifest, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("model file not found: %w", err)
	}

	baseName := filepath.Base(filePath)

	return &GGUFModelManifest{
		ModelID:              baseName,
		ModelVersion:         "1.0",
		Architecture:         "llama",
		Quantization:         "Q4_K_M",
		FileSizeBytes:        fi.Size(),
		ParameterCount:       "7B",
		ContextLength:        4096,
		TotalLayers:          32,
		RequiredVRAMEstimate: uint64(fi.Size()) + 512*1024*1024,
		SupportedRuntime:     "llama.cpp",
		LocalPath:            filePath,
	}, nil
}
