package main

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
	deployments, updates, stop := h.watcher.Watch()
	defer close(stop)

	var err error
	var data []byte
	firstEvent := true

	// Helper function to create event ID from deployment metadata
	makeEventId := func(deployment *FlinkDeployment) string {
		return fmt.Sprintf("%s-%s", deployment.ObjectMeta.UID, deployment.ObjectMeta.ResourceVersion)
	}

	// Send initial deployments
	for _, nsDeployments := range deployments {
		for _, deployment := range nsDeployments {
			event := DeploymentEvent{
				Type:       watch.Added,
				Deployment: deployment,
			}

			if data, err = json.Marshal(event); err != nil {
				return fmt.Errorf("could not marshal deployment: %w", err)
			}

			sseEvent := httpserver.SseEvent{
				Data: string(data),
				Id:   makeEventId(deployment),
			}

			// Set retry only on first event (tells browser to wait 5s before reconnecting)
			if firstEvent {
				sseEvent.Retry = 5000
				firstEvent = false
			}

			if err := writer.SendEvent(sseEvent); err != nil {
				return fmt.Errorf("could not write deployment to sse stream: %w", err)
			}
		}
	}

	// Stream updates
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if data, err = json.Marshal(update); err != nil {
				return fmt.Errorf("could not marshal deployment update: %w", err)
			}

			sseEvent := httpserver.SseEvent{
				Data: string(data),
				Id:   makeEventId(update.Deployment),
			}

			if err := writer.SendEvent(sseEvent); err != nil {
				return fmt.Errorf("could not write deployment update to sse stream: %w", err)
			}
		}
	}
}
