package main

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHandlerCheckpoints(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerCheckpoints, error) {
	var err error
	var client *FlinkClient
	var watcher *DeploymentWatcherModule

	if client, err = ProvideFlinkClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create flink client: %w", err)
	}

	if watcher, err = ProvideDeploymentWatcherModule(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not initialize deployment watcher: %w", err)
	}

	return &HandlerCheckpoints{
		logger:  logger.WithChannel("handler_checkpoints"),
		client:  client,
		watcher: watcher,
	}, nil
}

type HandlerCheckpoints struct {
	logger  log.Logger
	client  *FlinkClient
	watcher *DeploymentWatcherModule
}

type GetCheckpointsRequest struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
}

func (h *HandlerCheckpoints) GetCheckpoints(ctx context.Context, request *GetCheckpointsRequest) (httpserver.Response, error) {
	deployment, exists := h.watcher.GetDeployment(request.Namespace, request.Name)
	if !exists {
		return nil, fmt.Errorf("deployment %s/%s not found", request.Namespace, request.Name)
	}

	flinkURL := "https://" + deployment.Spec.Ingress.Template
	jobID := deployment.Status.JobStatus.JobId

	h.logger.Info(ctx, "fetching checkpoints for deployment %s/%s (job %s) from %s", request.Namespace, request.Name, jobID, flinkURL)

	stats, err := h.client.GetCheckpoints(ctx, flinkURL, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch checkpoints from Flink: %w", err)
	}

	return httpserver.NewJsonResponse(stats), nil
}
