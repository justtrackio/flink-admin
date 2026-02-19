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
	var client *FlinkClient
	var watcher *DeploymentWatcherModule

	if client, err = ProvideFlinkClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create flink client: %w", err)
	}

	if watcher, err = ProvideDeploymentWatcherModule(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not initialize deployment watcher: %w", err)
	}

	return &HandlerExceptions{
		logger:  logger.WithChannel("handler_exceptions"),
		client:  client,
		watcher: watcher,
	}, nil
}

type HandlerExceptions struct {
	logger  log.Logger
	client  *FlinkClient
	watcher *DeploymentWatcherModule
}

type GetExceptionsRequest struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
}

func (h *HandlerExceptions) GetExceptions(ctx context.Context, request *GetExceptionsRequest) (httpserver.Response, error) {
	deployment, exists := h.watcher.GetDeployment(request.Namespace, request.Name)
	if !exists {
		return nil, fmt.Errorf("deployment %s/%s not found", request.Namespace, request.Name)
	}

	flinkURL := "https://" + deployment.Spec.Ingress.Template
	jobID := deployment.Status.JobStatus.JobId

	h.logger.Info(ctx, "fetching exceptions for deployment %s/%s (job %s) from %s", request.Namespace, request.Name, jobID, flinkURL)

	exceptions, err := h.client.GetExceptions(ctx, flinkURL, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exceptions from Flink: %w", err)
	}

	return httpserver.NewJsonResponse(exceptions), nil
}
