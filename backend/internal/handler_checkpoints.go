package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHandlerCheckpoints(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerCheckpoints, error) {
	base, err := newFlinkDeploymentHandler(ctx, config, logger, "handler_checkpoints")
	if err != nil {
		return nil, err
	}

	return &HandlerCheckpoints{flinkDeploymentHandler: base}, nil
}

type HandlerCheckpoints struct {
	flinkDeploymentHandler
}

type GetCheckpointsRequest struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
}

func (h *HandlerCheckpoints) GetCheckpoints(ctx context.Context, request *GetCheckpointsRequest) (httpserver.Response, error) {
	flinkURL, jobID, err := h.watcher.GetFlinkEndpoint(request.Namespace, request.Name)
	if err != nil {
		return nil, err
	}

	h.logger.Info(ctx, "fetching checkpoints for deployment %s/%s (job %s) from %s", request.Namespace, request.Name, jobID, flinkURL)

	stats, err := h.client.GetCheckpoints(ctx, flinkURL, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch checkpoints from Flink: %w", err)
	}

	return httpserver.NewJsonResponse(stats), nil
}
