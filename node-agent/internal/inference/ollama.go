package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type OllamaRuntime struct {
	mu         sync.RWMutex
	endpoint   string
	httpClient *http.Client
	activeJobs map[string]context.CancelFunc
}

func NewOllamaRuntime(endpoint string) *OllamaRuntime {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	return &OllamaRuntime{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		activeJobs: make(map[string]context.CancelFunc),
	}
}

func (o *OllamaRuntime) Name() string {
	return "OllamaRuntime"
}

func (o *OllamaRuntime) Start(ctx context.Context) error {
	return o.Health(ctx)
}

func (o *OllamaRuntime) Stop(ctx context.Context) error {
	return nil
}

func (o *OllamaRuntime) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/version", nil)
	if err != nil {
		return err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama service unreachable at %s: %w", o.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health returned status code %d", resp.StatusCode)
	}
	return nil
}

func (o *OllamaRuntime) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type tagModel struct {
		Name string `json:"name"`
	}
	type tagsResponse struct {
		Models []tagModel `json:"models"`
	}

	var res tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(res.Models))
	for _, m := range res.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

func (o *OllamaRuntime) LoadModel(ctx context.Context, modelID string) error {
	body, _ := json.Marshal(map[string]string{"name": modelID})
	req, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/show", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("model %s not available in Ollama", modelID)
	}
	return nil
}

func (o *OllamaRuntime) UnloadModel(ctx context.Context, modelID string) error {
	return nil
}

func (o *OllamaRuntime) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var fullText string
	var totalTokens int

	err := o.StreamGenerate(ctx, req, func(chunk string, tokens int) error {
		fullText += chunk
		totalTokens = tokens
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &GenerateResponse{
		JobID:           req.JobID,
		Text:            fullText,
		TokensGenerated: totalTokens,
		FinishReason:    "completed",
	}, nil
}

func (o *OllamaRuntime) StreamGenerate(ctx context.Context, req *GenerateRequest, chunkCallback func(chunk string, tokens int) error) error {
	jobCtx, cancel := context.WithCancel(ctx)
	o.mu.Lock()
	o.activeJobs[req.JobID] = cancel
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		delete(o.activeJobs, req.JobID)
		o.mu.Unlock()
		cancel()
	}()

	payload := map[string]interface{}{
		"model":  req.ModelID,
		"prompt": req.Prompt,
		"stream": true,
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(jobCtx, "POST", o.endpoint+"/api/generate", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	tokens := 0

	type generateChunk struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	for {
		var chunk generateChunk
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		tokens++
		if err := chunkCallback(chunk.Response, tokens); err != nil {
			return err
		}

		if chunk.Done {
			break
		}
	}

	return nil
}

func (o *OllamaRuntime) Cancel(ctx context.Context, jobID string) error {
	o.mu.Lock()
	cancel, ok := o.activeJobs[jobID]
	o.mu.Unlock()

	if ok && cancel != nil {
		cancel()
		return nil
	}
	return fmt.Errorf("job %s not found or finished", jobID)
}

func (o *OllamaRuntime) Capabilities() []string {
	return []string{"TEXT_GENERATION", "STREAMING", "EMBEDDINGS"}
}
