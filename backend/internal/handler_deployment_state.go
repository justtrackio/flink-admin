package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	FlinkJobStateRunning   = "running"
	FlinkJobStateSuspended = "suspended"
)

func NewHandlerDeploymentState(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerDeploymentState, error) {
	var err error
	var k8sService *K8sService

	if k8sService, err = ProvideK8sService(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not provide k8s service: %w", err)
	}

	return &HandlerDeploymentState{
		logger:     logger.WithChannel("handler_deployment_state"),
		k8sService: k8sService,
	}, nil
}

type HandlerDeploymentState struct {
	logger     log.Logger
	k8sService *K8sService
}

type UpdateDeploymentStateResponse struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	State     string `json:"state"`
	Status    string `json:"status"`
}

func (h *HandlerDeploymentState) SuspendDeployment(ctx context.Context, request *DeploymentSelectorInput) (httpserver.Response, error) {
	return h.updateDeploymentState(ctx, request, FlinkJobStateSuspended)
}

func (h *HandlerDeploymentState) ResumeDeployment(ctx context.Context, request *DeploymentSelectorInput) (httpserver.Response, error) {
	return h.updateDeploymentState(ctx, request, FlinkJobStateRunning)
}

func (h *HandlerDeploymentState) updateDeploymentState(ctx context.Context, request *DeploymentSelectorInput, state string) (httpserver.Response, error) {
	h.logger.Info(ctx, "patching deployment %s/%s job state to %s", request.Namespace, request.Name, state)

	if err := h.k8sService.PatchDeploymentJobState(ctx, request.Namespace, request.Name, state); err != nil {
		return nil, fmt.Errorf("failed to update deployment job state: %w", err)
	}

	response := UpdateDeploymentStateResponse{
		Namespace: request.Namespace,
		Name:      request.Name,
		State:     state,
		Status:    "ok",
	}

	return httpserver.NewJsonResponse(response), nil
}
