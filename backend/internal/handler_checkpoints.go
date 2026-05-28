package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHandlerCheckpoints(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerCheckpoints, error) {
	var err error
	var base flinkDeploymentHandler

	if base, err = newFlinkDeploymentHandler(ctx, config, logger, "handler_checkpoints"); err != nil {
		return nil, err
	}

	return &HandlerCheckpoints{flinkDeploymentHandler: base}, nil
}

type HandlerCheckpoints struct {
	flinkDeploymentHandler
}

func (h *HandlerCheckpoints) GetCheckpoints(ctx context.Context, request *DeploymentSelectorInput) (httpserver.Response, error) {
	var err error
	var flinkURL string
	var jobID string
	var stats *FlinkCheckpointStatistics

	if flinkURL, jobID, err = h.watcher.GetFlinkEndpoint(request.Namespace, request.Name); err != nil {
		return nil, err
	}

	h.logger.Info(ctx, "fetching checkpoints for deployment %s/%s (job %s) from %s", request.Namespace, request.Name, jobID, flinkURL)

	if stats, err = h.client.GetCheckpoints(ctx, flinkURL, jobID); err != nil {
		return nil, fmt.Errorf("failed to fetch checkpoints from Flink: %w", err)
	}

	return httpserver.NewJsonResponse(stats), nil
}
