package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
	"k8s.io/apimachinery/pkg/watch"
)

func NewHandlerDeployments(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerDeployments, error) {
	var err error
	var watcher *DeploymentWatcherModule

	if watcher, err = ProvideDeploymentWatcherModule(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not initialize deployment watcher: %w", err)
	}

	return &HandlerDeployments{
		watcher: watcher,
	}, nil
}

type HandlerDeployments struct {
	watcher *DeploymentWatcherModule
}

func (h *HandlerDeployments) WatchDeployments(ctx context.Context, writer *httpserver.SseWriter) error {
	deployments, updates, stop := h.watcher.Watch(ctx)
	defer close(stop)

	firstEvent := true
	if err := sendInitialDeployments(writer, deployments, &firstEvent); err != nil {
		return err
	}

	return streamDeploymentUpdates(ctx, writer, updates)
}

func sendInitialDeployments(writer *httpserver.SseWriter, deployments map[string]map[string]*FlinkDeployment, firstEvent *bool) error {
	for _, nsDeployments := range deployments {
		for _, deployment := range nsDeployments {
			event := DeploymentEvent{
				Type:       watch.Added,
				Deployment: deployment,
			}

			data, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("could not marshal deployment: %w", err)
			}

			sseEvent := httpserver.SseEvent{
				Data: string(data),
				Id:   makeDeploymentEventID(deployment),
			}

			// Set retry only on first event (tells browser to wait 5s before reconnecting)
			if *firstEvent {
				sseEvent.Retry = 5000
				*firstEvent = false
			}

			if err := writer.SendEvent(sseEvent); err != nil {
				return fmt.Errorf("could not write deployment to sse stream: %w", err)
			}
		}
	}

	return nil
}

func streamDeploymentUpdates(ctx context.Context, writer *httpserver.SseWriter, updates <-chan DeploymentEvent) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil // channel closed, watcher is shutting down
			}
			data, err := json.Marshal(update)
			if err != nil {
				return fmt.Errorf("could not marshal deployment update: %w", err)
			}

			sseEvent := httpserver.SseEvent{
				Data: string(data),
				Id:   makeDeploymentEventID(update.Deployment),
			}

			if err := writer.SendEvent(sseEvent); err != nil {
				return fmt.Errorf("could not write deployment update to sse stream: %w", err)
			}
		}
	}
}

func makeDeploymentEventID(deployment *FlinkDeployment) string {
	return fmt.Sprintf("%s-%s", deployment.UID, deployment.ResourceVersion)
}
