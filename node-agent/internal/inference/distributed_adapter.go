package inference

import (
	"context"

	distrt "github.com/distributed-ai/distributed-runtime/pkg/runtime"
)

type DistributedRuntimeAdapter struct {
	dr *distrt.DistributedRuntime
}

func NewDistributedRuntimeAdapter(cacheDir string, nodeID string, rpcPort int) (*DistributedRuntimeAdapter, error) {
	dr, err := distrt.NewDistributedRuntime(cacheDir, nodeID, rpcPort)
	if err != nil {
		return nil, err
	}
	return &DistributedRuntimeAdapter{dr: dr}, nil
}

func (dra *DistributedRuntimeAdapter) Name() string {
	return dra.dr.Name()
}

func (dra *DistributedRuntimeAdapter) Start(ctx context.Context) error {
	return dra.dr.Start(ctx)
}

func (dra *DistributedRuntimeAdapter) Stop(ctx context.Context) error {
	return dra.dr.Stop(ctx)
}

func (dra *DistributedRuntimeAdapter) Health(ctx context.Context) error {
	return dra.dr.Health(ctx)
}

func (dra *DistributedRuntimeAdapter) ListModels(ctx context.Context) ([]string, error) {
	return dra.dr.ListModels(ctx)
}

func (dra *DistributedRuntimeAdapter) LoadModel(ctx context.Context, modelID string) error {
	return dra.dr.LoadModel(ctx, modelID)
}

func (dra *DistributedRuntimeAdapter) UnloadModel(ctx context.Context, modelID string) error {
	return dra.dr.UnloadModel(ctx, modelID)
}

func (dra *DistributedRuntimeAdapter) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	distReq := &distrt.GenerateRequest{
		JobID:       req.JobID,
		ModelID:     req.ModelID,
		Prompt:      req.Prompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	resp, err := dra.dr.Generate(ctx, distReq)
	if err != nil {
		return nil, err
	}
	return &GenerateResponse{
		JobID:           resp.JobID,
		Text:            resp.Text,
		TokensGenerated: resp.TokensGenerated,
		FinishReason:    resp.FinishReason,
	}, nil
}

func (dra *DistributedRuntimeAdapter) StreamGenerate(ctx context.Context, req *GenerateRequest, chunkCallback func(chunk string, tokens int) error) error {
	distReq := &distrt.GenerateRequest{
		JobID:       req.JobID,
		ModelID:     req.ModelID,
		Prompt:      req.Prompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	return dra.dr.StreamGenerate(ctx, distReq, chunkCallback)
}

func (dra *DistributedRuntimeAdapter) Cancel(ctx context.Context, jobID string) error {
	return dra.dr.Cancel(ctx, jobID)
}

func (dra *DistributedRuntimeAdapter) Capabilities() []string {
	return dra.dr.Capabilities()
}
