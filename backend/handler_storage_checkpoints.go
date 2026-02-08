package main

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

	// Get checkpoint directory from flinkConfiguration
	var checkpointBaseDir string
	if deployment.Spec.FlinkConfiguration != nil {
		if checkpointDir, ok := deployment.Spec.FlinkConfiguration["execution.checkpointing.dir"]; ok {
			if checkpointDirStr, ok := checkpointDir.(string); ok && checkpointDirStr != "" {
				checkpointBaseDir = checkpointDirStr
				response.CheckpointDir = checkpointDirStr

				h.logger.Info(ctx, "scanning for checkpoints in all job IDs under %s", checkpointBaseDir)

				// List all job ID directories
				jobIds, err := h.s3Service.ListJobDirectories(ctx, checkpointBaseDir)
				if err != nil {
					h.logger.Warn(ctx, "failed to list job directories: %v", err)
				} else {
					h.logger.Info(ctx, "found %d job directories to scan", len(jobIds))

					// For each job ID, list valid checkpoints
					for _, jobId := range jobIds {
						checkpoints, err := h.s3Service.ListValidCheckpoints(ctx, checkpointBaseDir, jobId)
						if err != nil {
							h.logger.Warn(ctx, "failed to list checkpoints for job %s: %v", jobId, err)
							continue
						}
						response.Checkpoints = append(response.Checkpoints, checkpoints...)
					}
				}
			}
		}

		// Get current job ID for display purposes
		jobId := deployment.Status.JobStatus.JobId
		if jobId != "" {
			response.JobId = jobId
		}

		// Get savepoint directory from flinkConfiguration (unchanged logic)
		if savepointDir, ok := deployment.Spec.FlinkConfiguration["execution.checkpointing.savepoint-dir"]; ok {
			if savepointDirStr, ok := savepointDir.(string); ok && savepointDirStr != "" {
				response.SavepointDir = savepointDirStr

				// Get current job ID in S3 format (without dashes) for savepoints
				jobId := deployment.Status.JobStatus.JobId
				if jobId != "" {
					jobId = strings.ReplaceAll(jobId, "-", "")

					// Append job ID to savepoint directory path
					savepointPath := savepointDirStr
					if !strings.HasSuffix(savepointPath, "/") {
						savepointPath += "/"
					}
					savepointPath += jobId

					h.logger.Info(ctx, "listing savepoints from %s", savepointPath)

					savepoints, err := h.s3Service.ListStorageCheckpoints(ctx, savepointPath)
					if err != nil {
						h.logger.Warn(ctx, "failed to list savepoints: %v", err)
					} else {
						response.Savepoints = savepoints
					}
				}
			}
		}
	}

	h.logger.Info(ctx, "returning %d checkpoints and %d savepoints for %s/%s",
		len(response.Checkpoints), len(response.Savepoints), request.Namespace, request.Name)

	return httpserver.NewJsonResponse(response), nil
}
