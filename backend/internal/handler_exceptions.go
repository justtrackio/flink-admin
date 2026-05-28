package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHandlerExceptions(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerExceptions, error) {
	var err error
	var base flinkDeploymentHandler

	if base, err = newFlinkDeploymentHandler(ctx, config, logger, "handler_exceptions"); err != nil {
		return nil, err
	}

	return &HandlerExceptions{flinkDeploymentHandler: base}, nil
}

type HandlerExceptions struct {
	flinkDeploymentHandler
}

func (h *HandlerExceptions) GetExceptions(ctx context.Context, request *DeploymentSelectorInput) (httpserver.Response, error) {
	var err error
	var flinkURL string
	var jobID string
	var exceptions *FlinkJobExceptions

	if flinkURL, jobID, err = h.watcher.GetFlinkEndpoint(request.Namespace, request.Name); err != nil {
		return nil, err
	}

	h.logger.Info(ctx, "fetching exceptions for deployment %s/%s (job %s) from %s", request.Namespace, request.Name, jobID, flinkURL)

	if exceptions, err = h.client.GetExceptions(ctx, flinkURL, jobID); err != nil {
		return nil, fmt.Errorf("failed to fetch exceptions from Flink: %w", err)
	}

	return httpserver.NewJsonResponse(exceptions), nil
}
