package model

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ModelCacheManager struct {
	mu         sync.RWMutex
	downloadMu sync.Mutex
	cacheDir   string
	models     map[string]*GGUFModelManifest
}

func NewModelCacheManager(cacheDir string) (*ModelCacheManager, error) {
	if cacheDir == "" {
		cacheDir = `C:\ProgramData\DistributedAI\models`
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		cacheDir = filepath.Join(".", "models")
		_ = os.MkdirAll(cacheDir, 0755)
	}

	mgr := &ModelCacheManager{
		cacheDir: cacheDir,
		models:   make(map[string]*GGUFModelManifest),
	}

	_ = mgr.RefreshCache()
	return mgr, nil
}

func (mcm *ModelCacheManager) RefreshCache() error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	entries, err := os.ReadDir(mcm.cacheDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".gguf" || ext == ".bin" {
			fullPath := filepath.Join(mcm.cacheDir, entry.Name())
			manifest, err := InspectGGUFFile(fullPath)
			if err == nil {
				mcm.models[entry.Name()] = manifest
			}
		}
	}
	return nil
}

func (mcm *ModelCacheManager) GetModel(modelID string) (*GGUFModelManifest, bool) {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	manifest, exists := mcm.models[modelID]
	if !exists {
		for name, m := range mcm.models {
			if name == modelID || filepath.Base(m.LocalPath) == modelID {
				return m, true
			}
		}
	}
	return manifest, exists
}

// EnsureModel makes a missing model available before an inference request runs.
func (mcm *ModelCacheManager) EnsureModel(modelID string) (*GGUFModelManifest, error) {
	mcm.downloadMu.Lock()
	defer mcm.downloadMu.Unlock()

	if manifest, ok := mcm.GetModel(modelID); ok && fileExists(manifest.LocalPath) {
		return manifest, nil
	}

	url, ok := modelDownloadURL(modelID)
	if !ok {
		return nil, fmt.Errorf("model %s is not installed and has no configured download URL", modelID)
	}

	if err := os.MkdirAll(mcm.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create model cache: %w", err)
	}
	modelPath := filepath.Join(mcm.cacheDir, filepath.Base(modelID))
	partPath := modelPath + ".part"
	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		if err := downloadFile(url, partPath); err == nil {
			if err := verifyGGUF(partPath); err == nil {
				if err := os.Rename(partPath, modelPath); err != nil {
					return nil, fmt.Errorf("finalize model download: %w", err)
				}
				_ = mcm.RefreshCache()
				if manifest, ok := mcm.GetModel(filepath.Base(modelID)); ok {
					return manifest, nil
				}
				return nil, fmt.Errorf("downloaded model %s was not discovered", modelID)
			} else {
				lastErr = err
			}
		} else {
			lastErr = err
		}
		_ = os.Remove(partPath)
		time.Sleep(time.Duration(attempt*attempt) * time.Second)
	}
	return nil, fmt.Errorf("download model %s failed after 5 attempts: %w", modelID, lastErr)
}

func modelDownloadURL(modelID string) (string, bool) {
	baseName := strings.ToLower(filepath.Base(modelID))
	if baseName == "qwen2.5-0.5b-instruct-q4_k_m.gguf" {
		return "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf", true
	}
	return "", false
}

func downloadFile(url string, destination string) error {
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

func verifyGGUF(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("read GGUF header: %w", err)
	}
	if string(header) != "GGUF" {
		return fmt.Errorf("downloaded file is not a GGUF model")
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (mcm *ModelCacheManager) AddModelPlaceholder(modelID string, expectedSize int64, expectedSHA256 string) (*GGUFModelManifest, error) {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	fullPath := filepath.Join(mcm.cacheDir, modelID)
	manifest := &GGUFModelManifest{
		ModelID:              modelID,
		ModelVersion:         "1.0",
		Architecture:         "llama",
		FileSizeBytes:        expectedSize,
		ExpectedSHA256:       expectedSHA256,
		LocalPath:            fullPath,
		SupportedRuntime:     "llama.cpp",
		TotalLayers:          32,
		RequiredVRAMEstimate: uint64(expectedSize) + 512*1024*1024,
	}

	mcm.models[modelID] = manifest
	return manifest, nil
}

func (mcm *ModelCacheManager) ListModels() []*GGUFModelManifest {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	list := make([]*GGUFModelManifest, 0, len(mcm.models))
	for _, m := range mcm.models {
		list = append(list, m)
	}
	return list
}

func (mcm *ModelCacheManager) CacheDir() string {
	return mcm.cacheDir
}
