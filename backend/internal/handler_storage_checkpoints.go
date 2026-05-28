package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mapx"
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
	CheckpointDir string       `json:"checkpointDir,omitempty"`
	SavepointDir  string       `json:"savepointDir,omitempty"`
	StateEntries  []StateEntry `json:"stateEntries"`
}

func (h *HandlerStorageCheckpoints) GetStorageCheckpoints(ctx context.Context, request *GetStorageCheckpointsRequest) (httpserver.Response, error) {
	var ok bool

	deployment, exists := h.watcher.GetDeployment(request.Namespace, request.Name)
	if !exists {
		return nil, fmt.Errorf("deployment %s/%s not found", request.Namespace, request.Name)
	}

	response := StorageCheckpointsResponse{
		StateEntries: []StateEntry{},
	}

	flinkConfig := mapx.NewMapX(deployment.Spec.FlinkConfiguration)
	if flinkConfig == nil {
		return nil, fmt.Errorf("flink configuration is missing")
	}

	checkpointBaseDir := flinkConfig.Get("execution.checkpointing.dir").Data()
	if response.CheckpointDir, ok = checkpointBaseDir.(string); ok {
		h.logger.Info(ctx, "scanning for checkpoints in all job IDs under %s", checkpointBaseDir)

		if err := h.populateCheckpoints(ctx, response.CheckpointDir, &response); err != nil {
			return nil, fmt.Errorf("failed to list checkpoints: %w", err)
		}
	}

	savepointDir := flinkConfig.Get("execution.checkpointing.savepoint-dir").Data()
	if response.SavepointDir, ok = savepointDir.(string); ok {
		if err := h.populateSavepoints(ctx, response.SavepointDir, &response); err != nil {
			return nil, fmt.Errorf("failed to list savepoints: %v", err)
		}
	}

	return httpserver.NewJsonResponse(response), nil
}

func (h *HandlerStorageCheckpoints) populateCheckpoints(ctx context.Context, checkpointBaseDir string, response *StorageCheckpointsResponse) error {
	var err error
	var jobIds []string
	var checkpoints []StateEntry

	if jobIds, err = h.s3Service.ListJobDirectories(ctx, checkpointBaseDir); err != nil {
		return err
	}

	h.logger.Info(ctx, "found %d job directories to scan", len(jobIds))
	for _, jobId := range jobIds {
		if checkpoints, err = h.s3Service.ListValidCheckpoints(ctx, checkpointBaseDir, jobId); err != nil {
			return fmt.Errorf("failed to list checkpoints for job %s: %v", jobId, err)
		}

		response.StateEntries = append(response.StateEntries, checkpoints...)
	}

	return nil
}

func (h *HandlerStorageCheckpoints) populateSavepoints(ctx context.Context, savepointDir string, response *StorageCheckpointsResponse) error {
	var err error
	var savepoints []StorageEntry
	var metadataInfo *MetadataInfo

	h.logger.Info(ctx, "listing savepoints from %s", savepointDir)

	if savepoints, err = h.s3Service.ListStorageCheckpoints(ctx, savepointDir); err != nil {
		return fmt.Errorf("failed to list savepoints: %w", err)
	}

	for _, savepoint := range savepoints {
		if metadataInfo, err = h.s3Service.GetMetadataInfo(ctx, savepoint.Path); err != nil {
			return fmt.Errorf("failed to get metadata for checkpoint %s: %w", savepoint.Path, err)
		}

		if !metadataInfo.Exists {
			continue
		}

		response.StateEntries = append(response.StateEntries, StateEntry{
			Type:         "savepoint",
			Name:         savepoint.Name,
			Path:         savepoint.Path,
			JobId:        "",
			LastModified: metadataInfo.LastModified,
			Size:         metadataInfo.Size,
		})
	}

	return nil
}
