package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHandlerEvents(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerEvents, error) {
	k8sService, err := ProvideK8sService(ctx, config, logger)
	if err != nil {
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

type GetEventsRequest struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
}

func (h *HandlerEvents) GetEvents(ctx context.Context, request *GetEventsRequest) (httpserver.Response, error) {
	h.logger.Info(ctx, "fetching events for deployment %s/%s", request.Namespace, request.Name)

	eventList, err := h.k8sService.GetEvents(ctx, request.Namespace, request.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events from kubernetes: %w", err)
	}

	response := K8sEventsResponse{
		Events: toK8sEvents(eventList.Items),
	}

	return httpserver.NewJsonResponse(response), nil
}
