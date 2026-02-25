package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHandlerExceptions(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerExceptions, error) {
	base, err := newFlinkDeploymentHandler(ctx, config, logger, "handler_exceptions")
	if err != nil {
		return nil, err
	}

	return &HandlerExceptions{flinkDeploymentHandler: base}, nil
}

type HandlerExceptions struct {
	flinkDeploymentHandler
}

type GetExceptionsRequest struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
}

func (h *HandlerExceptions) GetExceptions(ctx context.Context, request *GetExceptionsRequest) (httpserver.Response, error) {
	flinkURL, jobID, err := h.watcher.GetFlinkEndpoint(request.Namespace, request.Name)
	if err != nil {
		return nil, err
	}

	h.logger.Info(ctx, "fetching exceptions for deployment %s/%s (job %s) from %s", request.Namespace, request.Name, jobID, flinkURL)

	exceptions, err := h.client.GetExceptions(ctx, flinkURL, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exceptions from Flink: %w", err)
	}

	return httpserver.NewJsonResponse(exceptions), nil
}
