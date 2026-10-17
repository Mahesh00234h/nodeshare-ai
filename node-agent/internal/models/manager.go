package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ModelMetadata struct {
	ModelID          string    `json:"model_id"`
	Version          string    `json:"version"`
	SizeBytes        uint64    `json:"size_bytes"`
	SHA256Checksum   string    `json:"sha256_checksum"`
	Status           string    `json:"status"` // DOWNLOADING, VERIFIED, READY, ERROR
	DownloadedAt     time.Time `json:"downloaded_at"`
	LastVerifiedAt   time.Time `json:"last_verified_at"`
	MetadataFilePath string    `json:"-"`
}

type ModelManager struct {
	mu       sync.RWMutex
	modelDir string
	models   map[string]*ModelMetadata
}

func NewModelManager(modelDir string) (*ModelManager, error) {
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil, err
	}

	mm := &ModelManager{
		modelDir: modelDir,
		models:   make(map[string]*ModelMetadata),
	}

	_ = mm.ScanLocalModels()
	return mm, nil
}

func (mm *ModelManager) ScanLocalModels() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	entries, err := os.ReadDir(mm.modelDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			metaPath := filepath.Join(mm.modelDir, entry.Name(), "metadata.json")
			if data, err := os.ReadFile(metaPath); err == nil {
				var meta ModelMetadata
				if err := json.Unmarshal(data, &meta); err == nil {
					meta.MetadataFilePath = metaPath
					mm.models[meta.ModelID] = &meta
				}
			}
		}
	}
	return nil
}

func (mm *ModelManager) ListModels() []*ModelMetadata {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	list := make([]*ModelMetadata, 0, len(mm.models))
	for _, m := range mm.models {
		list = append(list, m)
	}
	return list
}

func (mm *ModelManager) GetModel(modelID string) (*ModelMetadata, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	m, ok := mm.models[modelID]
	return m, ok
}

func (mm *ModelManager) VerifyModel(modelID string, expectedSHA256 string, expectedSize uint64) (bool, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	modelPath := filepath.Join(mm.modelDir, modelID, "model.bin")
	info, err := os.Stat(modelPath)
	if err != nil {
		return false, fmt.Errorf("model binary not found: %w", err)
	}

	if expectedSize > 0 && uint64(info.Size()) != expectedSize {
		return false, fmt.Errorf("size mismatch: got %d, expected %d", info.Size(), expectedSize)
	}

	if expectedSHA256 != "" {
		file, err := os.Open(modelPath)
		if err != nil {
			return false, err
		}
		defer file.Close()

		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return false, err
		}

		actualHash := hex.EncodeToString(hasher.Sum(nil))
		if actualHash != expectedSHA256 {
			return false, fmt.Errorf("checksum mismatch: got %s, expected %s", actualHash, expectedSHA256)
		}
	}

	if meta, ok := mm.models[modelID]; ok {
		meta.Status = "READY"
		meta.LastVerifiedAt = time.Now()
		_ = mm.saveMetadata(meta)
	}

	return true, nil
}

func (mm *ModelManager) PrepareModel(modelID string) error {
	mm.mu.RLock()
	meta, ok := mm.models[modelID]
	mm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("model %s not found locally", modelID)
	}

	if meta.Status != "READY" && meta.Status != "VERIFIED" {
		return fmt.Errorf("model %s is in state %s, not ready", modelID, meta.Status)
	}

	return nil
}

func (mm *ModelManager) DeleteModel(modelID string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	dir := filepath.Join(mm.modelDir, modelID)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}

	delete(mm.models, modelID)
	return nil
}

func (mm *ModelManager) CreatePlaceholderModel(modelID string, version string, size uint64, sha256Hex string) (*ModelMetadata, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	dir := filepath.Join(mm.modelDir, modelID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	binPath := filepath.Join(dir, "model.bin")
	metaPath := filepath.Join(dir, "metadata.json")

	// Create dummy binary if not exists
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		dummyData := []byte(fmt.Sprintf("MOCK_MODEL_DATA_FOR_%s", modelID))
		_ = os.WriteFile(binPath, dummyData, 0644)
		size = uint64(len(dummyData))
		hasher := sha256.New()
		hasher.Write(dummyData)
		sha256Hex = hex.EncodeToString(hasher.Sum(nil))
	}

	meta := &ModelMetadata{
		ModelID:          modelID,
		Version:          version,
		SizeBytes:        size,
		SHA256Checksum:   sha256Hex,
		Status:           "READY",
		DownloadedAt:     time.Now(),
		LastVerifiedAt:   time.Now(),
		MetadataFilePath: metaPath,
	}

	data, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaPath, data, 0644)

	mm.models[modelID] = meta
	return meta, nil
}

func (mm *ModelManager) saveMetadata(meta *ModelMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(meta.MetadataFilePath, data, 0644)
}
