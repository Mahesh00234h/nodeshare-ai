package inference

import (
	"context"
)

type GenerateRequest struct {
	JobID       string  `json:"job_id"`
	ModelID     string  `json:"model_id"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type GenerateResponse struct {
	JobID           string `json:"job_id"`
	Text            string `json:"text"`
	TokensGenerated int    `json:"tokens_generated"`
	FinishReason    string `json:"finish_reason"`
}

type InferenceRuntime interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
	ListModels(ctx context.Context) ([]string, error)
	LoadModel(ctx context.Context, modelID string) error
	UnloadModel(ctx context.Context, modelID string) error
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	StreamGenerate(ctx context.Context, req *GenerateRequest, chunkCallback func(chunk string, tokens int) error) error
	Cancel(ctx context.Context, jobID string) error
	Capabilities() []string
}
