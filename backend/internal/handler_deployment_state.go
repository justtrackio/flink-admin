package internal

import (
	"context"
	"fmt"
	"strings"

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

type RecoverDeploymentRequest struct {
	Namespace            string `uri:"namespace"`
	Name                 string `uri:"name"`
	InitialSavepointPath string `json:"initialSavepointPath"`
}

type RecoverDeploymentResponse struct {
	Namespace              string `json:"namespace"`
	Name                   string `json:"name"`
	State                  string `json:"state"`
	InitialSavepointPath   string `json:"initialSavepointPath"`
	SavepointRedeployNonce int64  `json:"savepointRedeployNonce"`
	Status                 string `json:"status"`
}

func (h *HandlerDeploymentState) Suspend(ctx context.Context, request *DeploymentSelectorInput) (httpserver.Response, error) {
	return h.updateState(ctx, request, FlinkJobStateSuspended)
}

func (h *HandlerDeploymentState) Resume(ctx context.Context, request *DeploymentSelectorInput) (httpserver.Response, error) {
	return h.updateState(ctx, request, FlinkJobStateRunning)
}

func (h *HandlerDeploymentState) Recover(ctx context.Context, request *RecoverDeploymentRequest) (httpserver.Response, error) {
	var err error
	var deployment *FlinkDeployment

	initialSavepointPath := strings.TrimSpace(request.InitialSavepointPath)
	if initialSavepointPath == "" {
		return nil, fmt.Errorf("initial savepoint path is required")
	}

	if deployment, err = h.k8sService.GetDeployment(ctx, request.Namespace, request.Name); err != nil {
		return nil, fmt.Errorf("failed to get deployment for recovery: %w", err)
	}

	nonce := int64(1)
	if deployment.Spec.Job != nil && deployment.Spec.Job.SavepointRedeployNonce != nil {
		nonce = *deployment.Spec.Job.SavepointRedeployNonce + 1
	}

	h.logger.Info(ctx, "patching deployment %s/%s for recovery from %s with nonce %d", request.Namespace, request.Name, initialSavepointPath, nonce)

	spec := map[string]any{
		"state":                  FlinkJobStateRunning,
		"initialSavepointPath":   initialSavepointPath,
		"savepointRedeployNonce": nonce,
	}

	if err := h.k8sService.PatchDeploymentJobSpec(ctx, request.Namespace, request.Name, spec); err != nil {
		return nil, fmt.Errorf("failed to recover deployment from savepoint: %w", err)
	}

	response := RecoverDeploymentResponse{
		Namespace:              request.Namespace,
		Name:                   request.Name,
		State:                  FlinkJobStateRunning,
		InitialSavepointPath:   initialSavepointPath,
		SavepointRedeployNonce: nonce,
		Status:                 "ok",
	}

	return httpserver.NewJsonResponse(response), nil
}

func (h *HandlerDeploymentState) updateState(ctx context.Context, request *DeploymentSelectorInput, state string) (httpserver.Response, error) {
	h.logger.Info(ctx, "patching deployment %s/%s job state to %s", request.Namespace, request.Name, state)

	spec := map[string]any{
		"state": state,
	}

	if err := h.k8sService.PatchDeploymentJobSpec(ctx, request.Namespace, request.Name, spec); err != nil {
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
