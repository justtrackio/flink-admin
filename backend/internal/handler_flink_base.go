package internal

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// flinkDeploymentHandler holds the common dependencies for handlers that call the Flink REST API.
type flinkDeploymentHandler struct {
	logger  log.Logger
	client  *FlinkClient
	watcher *DeploymentWatcherModule
}

func newFlinkDeploymentHandler(ctx context.Context, config cfg.Config, logger log.Logger, channel string) (flinkDeploymentHandler, error) {
	client, err := ProvideFlinkClient(ctx, config, logger)
	if err != nil {
		return flinkDeploymentHandler{}, fmt.Errorf("could not create flink client: %w", err)
	}

	watcher, err := ProvideDeploymentWatcherModule(ctx, config, logger)
	if err != nil {
		return flinkDeploymentHandler{}, fmt.Errorf("could not initialize deployment watcher: %w", err)
	}

	return flinkDeploymentHandler{
		logger:  logger.WithChannel(channel),
		client:  client,
		watcher: watcher,
	}, nil
}
