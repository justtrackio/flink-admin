package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHandlerStorageCheckpoints(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerStorageCheckpoints, error) {
	var err error
	var watcher *DeploymentWatcherModule
	var s3Service *S3Service

	if watcher, err = ProvideDeploymentWatcherModule(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not initialize deployment watcher: %w", err)
	}

	if s3Service, err = ProvideS3Service(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not initialize s3 service: %w", err)
	}

	return &HandlerStorageCheckpoints{
		logger:    logger.WithChannel("handler_storage_checkpoints"),
		watcher:   watcher,
		s3Service: s3Service,
	}, nil
}

type HandlerStorageCheckpoints struct {
	logger    log.Logger
	watcher   *DeploymentWatcherModule
	s3Service *S3Service
}

type GetStorageCheckpointsRequest struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
}

type StorageCheckpointsResponse struct {
	JobId         string         `json:"jobId,omitempty"`
	CheckpointDir string         `json:"checkpointDir,omitempty"`
	SavepointDir  string         `json:"savepointDir,omitempty"`
	Checkpoints   []StorageEntry `json:"checkpoints"`
	Savepoints    []StorageEntry `json:"savepoints"`
}

func (h *HandlerStorageCheckpoints) GetStorageCheckpoints(ctx context.Context, request *GetStorageCheckpointsRequest) (httpserver.Response, error) {
	deployment, exists := h.watcher.GetDeployment(request.Namespace, request.Name)
	if !exists {
		return nil, fmt.Errorf("deployment %s/%s not found", request.Namespace, request.Name)
	}

	response := StorageCheckpointsResponse{
		Checkpoints: []StorageEntry{},
		Savepoints:  []StorageEntry{},
	}

	flinkConfig := deployment.Spec.FlinkConfiguration
	if flinkConfig == nil {
		h.logger.Info(ctx, "returning %d checkpoints and %d savepoints for %s/%s",
			len(response.Checkpoints), len(response.Savepoints), request.Namespace, request.Name)

		return httpserver.NewJsonResponse(response), nil
	}

	if checkpointBaseDir, ok := getStringConfig(flinkConfig, "execution.checkpointing.dir"); ok {
		response.CheckpointDir = checkpointBaseDir
		h.logger.Info(ctx, "scanning for checkpoints in all job IDs under %s", checkpointBaseDir)
		if err := h.populateCheckpoints(ctx, checkpointBaseDir, &response); err != nil {
			h.logger.Warn(ctx, "failed to list job directories: %v", err)
		}
	}

	jobId := deployment.Status.JobStatus.JobId
	if jobId != "" {
		response.JobId = jobId
	}

	if savepointDir, ok := getStringConfig(flinkConfig, "execution.checkpointing.savepoint-dir"); ok {
		response.SavepointDir = savepointDir
		h.populateSavepoints(ctx, savepointDir, jobId, &response)
	}

	h.logger.Info(ctx, "returning %d checkpoints and %d savepoints for %s/%s",
		len(response.Checkpoints), len(response.Savepoints), request.Namespace, request.Name)

	return httpserver.NewJsonResponse(response), nil
}

func getStringConfig(config map[string]any, key string) (string, bool) {
	value, ok := config[key]
	if !ok {
		return "", false
	}

	stringValue, ok := value.(string)
	if !ok || stringValue == "" {
		return "", false
	}

	return stringValue, true
}

func buildSavepointPath(savepointDir string, jobId string) string {
	dashlessJobId := strings.ReplaceAll(jobId, "-", "")
	path := savepointDir
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	return path + dashlessJobId
}

func (h *HandlerStorageCheckpoints) populateCheckpoints(ctx context.Context, checkpointBaseDir string, response *StorageCheckpointsResponse) error {
	jobIds, err := h.s3Service.ListJobDirectories(ctx, checkpointBaseDir)
	if err != nil {
		return err
	}

	h.logger.Info(ctx, "found %d job directories to scan", len(jobIds))
	for _, jobId := range jobIds {
		checkpoints, err := h.s3Service.ListValidCheckpoints(ctx, checkpointBaseDir, jobId)
		if err != nil {
			h.logger.Warn(ctx, "failed to list checkpoints for job %s: %v", jobId, err)

			continue
		}
		response.Checkpoints = append(response.Checkpoints, checkpoints...)
	}

	return nil
}

func (h *HandlerStorageCheckpoints) populateSavepoints(ctx context.Context, savepointDir string, jobId string, response *StorageCheckpointsResponse) {
	if jobId == "" {
		return
	}

	savepointPath := buildSavepointPath(savepointDir, jobId)
	h.logger.Info(ctx, "listing savepoints from %s", savepointPath)

	savepoints, err := h.s3Service.ListStorageCheckpoints(ctx, savepointPath)
	if err != nil {
		h.logger.Warn(ctx, "failed to list savepoints: %v", err)

		return
	}

	response.Savepoints = savepoints
}
