package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	eventsv1 "k8s.io/api/events/v1"
)

func NewHandlerEvents(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerEvents, error) {
	var err error
	var k8sService *K8sService

	if k8sService, err = ProvideK8sService(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not provide k8s service: %w", err)
	}

	return &HandlerEvents{
		logger:     logger.WithChannel("handler_events"),
		k8sService: k8sService,
	}, nil
}

type HandlerEvents struct {
	logger     log.Logger
	k8sService *K8sService
}

func (h *HandlerEvents) GetEvents(ctx context.Context, request *DeploymentSelectorInput) (httpserver.Response, error) {
	h.logger.Info(ctx, "fetching events for deployment %s/%s", request.Namespace, request.Name)

	var err error
	var eventList *eventsv1.EventList

	if eventList, err = h.k8sService.GetEvents(ctx, request.Namespace, request.Name); err != nil {
		return nil, fmt.Errorf("failed to fetch events from kubernetes: %w", err)
	}

	response := K8sEventsResponse{
		Events: toK8sEvents(eventList.Items),
	}

	return httpserver.NewJsonResponse(response), nil
}
