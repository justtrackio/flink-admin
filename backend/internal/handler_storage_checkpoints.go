package internal

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/flink-admin/internal/checkpoint"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/marusama/semaphore/v2"
)

const (
	ParseStatusFailed   = "failed"
	ParseStatusParsed   = "parsed"
	StateTypeCheckpoint = "checkpoint"
	StateTypeSavepoint  = "savepoint"
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

type GetStorageCheckpointMetadataRequest struct {
	Namespace string `uri:"namespace"`
	Name      string `uri:"name"`
	EntryType string `uri:"entryType"`
	JobId     string `uri:"jobId"`
	EntryName string `uri:"entryName"`
}

type StorageCheckpointsResponse struct {
	CheckpointDir string       `json:"checkpointDir,omitempty"`
	SavepointDir  string       `json:"savepointDir,omitempty"`
	StateEntries  []StateEntry `json:"stateEntries"`
}

type StorageCheckpointMetadataResponse struct {
	Type           string                            `json:"type"`
	Name           string                            `json:"name"`
	Path           string                            `json:"path"`
	MetadataPath   string                            `json:"metadataPath"`
	JobId          string                            `json:"jobId,omitempty"`
	MetadataExists bool                              `json:"metadataExists"`
	LastModified   *time.Time                        `json:"lastModified,omitempty"`
	Size           *int64                            `json:"size,omitempty"`
	ParseStatus    string                            `json:"parseStatus"`
	ParseError     string                            `json:"parseError,omitempty"`
	Summary        *StorageCheckpointMetadataSummary `json:"summary,omitempty"`
}

type StorageCheckpointMetadataSummary struct {
	Version        int32                               `json:"version"`
	CheckpointID   int64                               `json:"checkpointId"`
	NumOperators   int                                 `json:"numOperators"`
	Operators      []StorageCheckpointOperatorSummary  `json:"operators"`
	StateFilePaths []string                            `json:"stateFilePaths"`
	Properties     *StorageCheckpointPropertiesSummary `json:"properties,omitempty"`
}

type StorageCheckpointOperatorSummary struct {
	Name           string `json:"name"`
	UID            string `json:"uid"`
	OperatorID     string `json:"operatorId"`
	Parallelism    int32  `json:"parallelism"`
	MaxParallelism int32  `json:"maxParallelism"`
}

type StorageCheckpointPropertiesSummary struct {
	CheckpointType  string `json:"checkpointType,omitempty"`
	SharingStrategy string `json:"sharingStrategy,omitempty"`
	Source          string `json:"source,omitempty"`
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

func (h *HandlerStorageCheckpoints) GetStorageCheckpointMetadata(ctx context.Context, request *GetStorageCheckpointMetadataRequest) (httpserver.Response, error) {
	var err error
	var statePath string
	var metadataPath string
	var metadataInfo *MetadataInfo
	var metadataReader io.ReadCloser
	var summary *checkpoint.CheckpointSummary

	deployment, exists := h.watcher.GetDeployment(request.Namespace, request.Name)
	if !exists {
		return nil, fmt.Errorf("deployment %s/%s not found", request.Namespace, request.Name)
	}

	if err := validateStorageEntryRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	flinkConfig := mapx.NewMapX(deployment.Spec.FlinkConfiguration)
	if flinkConfig == nil {
		return nil, fmt.Errorf("flink configuration is missing")
	}

	switch request.EntryType {
	case StateTypeCheckpoint:
		checkpointBaseDir := flinkConfig.Get("execution.checkpointing.dir").Data()
		checkpointDir, ok := checkpointBaseDir.(string)
		if !ok || checkpointDir == "" {
			return nil, fmt.Errorf("checkpoint directory is not configured")
		}

		statePath = joinS3Path(checkpointDir, request.JobId, request.EntryName)
	case StateTypeSavepoint:
		savepointBaseDir := flinkConfig.Get("execution.checkpointing.savepoint-dir").Data()
		savepointDir, ok := savepointBaseDir.(string)
		if !ok || savepointDir == "" {
			return nil, fmt.Errorf("savepoint directory is not configured")
		}

		statePath = joinS3Path(savepointDir, request.EntryName)
	default:
		return nil, fmt.Errorf("unsupported storage entry type: %s", request.EntryType)
	}

	if metadataPath, err = buildMetadataS3URI(statePath); err != nil {
		return nil, fmt.Errorf("failed to build metadata path: %w", err)
	}

	if metadataInfo, err = h.s3Service.GetMetadataInfo(ctx, statePath); err != nil {
		return nil, fmt.Errorf("failed to get metadata info: %w", err)
	}

	response := StorageCheckpointMetadataResponse{
		Type:           request.EntryType,
		Name:           request.EntryName,
		Path:           statePath,
		MetadataPath:   metadataPath,
		MetadataExists: metadataInfo.Exists,
		LastModified:   metadataInfo.LastModified,
		Size:           metadataInfo.Size,
		ParseStatus:    "missing",
	}

	if request.EntryType == StateTypeCheckpoint {
		response.JobId = request.JobId
	}

	if !metadataInfo.Exists {
		return httpserver.NewJsonResponse(response), nil
	}

	if metadataReader, err = h.s3Service.OpenMetadata(ctx, statePath); err != nil {
		response.ParseStatus = ParseStatusFailed
		response.ParseError = err.Error()

		return httpserver.NewJsonResponse(response), nil
	}

	defer func() {
		if cerr := metadataReader.Close(); cerr != nil {
			h.logger.Warn(ctx, "failed to close metadata reader for %s: %v", statePath, cerr)
		}
	}()

	if summary, err = checkpoint.ParseSummary(metadataReader, checkpoint.ParseOptions{IncludeInlineStrings: true}); err != nil {
		response.ParseStatus = ParseStatusFailed
		response.ParseError = err.Error()

		return httpserver.NewJsonResponse(response), nil
	}

	response.ParseStatus = ParseStatusParsed
	response.Summary = buildStorageCheckpointMetadataSummary(summary)

	return httpserver.NewJsonResponse(response), nil
}

func (h *HandlerStorageCheckpoints) populateCheckpoints(ctx context.Context, checkpointBaseDir string, response *StorageCheckpointsResponse) error {
	var err error
	var jobIds []string

	if jobIds, err = h.s3Service.ListJobDirectories(ctx, checkpointBaseDir); err != nil {
		return err
	}

	h.logger.Info(ctx, "found %d job directories to scan", len(jobIds))

	sem := semaphore.New(25)
	cfn := coffin.New()
	checkpointsPerJob := make([][]StateEntry, len(jobIds))

	for j, jobId := range jobIds {
		cfn.GoWithContext(ctx, func(ctx context.Context) error {
			if err = sem.Acquire(ctx, 1); err != nil {
				return fmt.Errorf("failed to acquire semaphore: %w", err)
			}
			defer sem.Release(1)

			if checkpointsPerJob[j], err = h.s3Service.ListValidCheckpoints(ctx, checkpointBaseDir, jobId); err != nil {
				return fmt.Errorf("failed to list checkpoints for job %s: %w", jobId, err)
			}

			return nil
		})
	}

	if err = cfn.Wait(); err != nil {
		return fmt.Errorf("failed waiting for checkpoints: %w", err)
	}

	response.StateEntries = funk.Flatten(checkpointsPerJob)
	cfn = coffin.New()

	for c := range response.StateEntries {
		cfn.GoWithContext(ctx, func(ctx context.Context) error {
			if err = sem.Acquire(ctx, 1); err != nil {
				return fmt.Errorf("failed to acquire semaphore: %w", err)
			}
			defer sem.Release(1)

			if response.StateEntries[c].CheckpointId, err = h.getCheckpointId(ctx, response.StateEntries[c].Path); err != nil {
				return fmt.Errorf("failed to get checkpoint id for %s: %w", response.StateEntries[c].Path, err)
			}

			return nil
		})
	}

	if err = cfn.Wait(); err != nil {
		return fmt.Errorf("failed waiting for checkpoints: %w", err)
	}

	return nil
}

func (h *HandlerStorageCheckpoints) populateSavepoints(ctx context.Context, savepointDir string, response *StorageCheckpointsResponse) error {
	var err error
	var savepoints []StorageEntry
	var metadataInfo *MetadataInfo
	var checkpointId int64

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

		if checkpointId, err = h.getCheckpointId(ctx, savepoint.Path); err != nil {
			return fmt.Errorf("failed to get checkpoint id for %s: %w", savepoint.Path, err)
		}

		response.StateEntries = append(response.StateEntries, StateEntry{
			Type:         StateTypeSavepoint,
			Name:         savepoint.Name,
			Path:         savepoint.Path,
			JobId:        "",
			CheckpointId: checkpointId,
			LastModified: metadataInfo.LastModified,
			Size:         metadataInfo.Size,
		})
	}

	return nil
}

func (h *HandlerStorageCheckpoints) getCheckpointId(ctx context.Context, statePath string) (int64, error) {
	var err error
	var metadataReader io.ReadCloser
	var summary *checkpoint.CheckpointSummary

	if metadataReader, err = h.s3Service.OpenMetadata(ctx, statePath); err != nil {
		return 0, fmt.Errorf("failed to open metadata for %s: %w", statePath, err)
	}

	defer func() {
		if cerr := metadataReader.Close(); cerr != nil {
			h.logger.Warn(ctx, "failed to close metadata reader for %s: %v", statePath, cerr)
		}
	}()

	if summary, err = checkpoint.ParseSummary(metadataReader, checkpoint.ParseOptions{}); err != nil {
		return 0, fmt.Errorf("failed to parse metadata for %s: %w", statePath, err)
	}

	return summary.CheckpointID, nil
}

func validateStorageEntryRequest(request *GetStorageCheckpointMetadataRequest) error {
	if request.EntryName == "" || hasPathSeparator(request.EntryName) || request.EntryName == "." || request.EntryName == ".." {
		return fmt.Errorf("invalid storage entry name: %s", request.EntryName)
	}

	if request.EntryType == StateTypeCheckpoint {
		if request.JobId == "" || request.JobId == "-" || hasPathSeparator(request.JobId) || request.JobId == "." || request.JobId == ".." {
			return fmt.Errorf("invalid checkpoint job id: %s", request.JobId)
		}
	}

	return nil
}

func hasPathSeparator(value string) bool {
	return strings.Contains(value, "/") || strings.Contains(value, "\\")
}

func joinS3Path(base string, parts ...string) string {
	path := strings.TrimRight(base, "/")
	for _, part := range parts {
		path += "/" + strings.Trim(part, "/")
	}

	return path
}

func buildStorageCheckpointMetadataSummary(summary *checkpoint.CheckpointSummary) *StorageCheckpointMetadataSummary {
	operators := make([]StorageCheckpointOperatorSummary, 0, len(summary.Operators))
	for _, operator := range summary.Operators {
		operators = append(operators, StorageCheckpointOperatorSummary{
			Name:           operator.Name,
			UID:            operator.UID,
			OperatorID:     hex.EncodeToString(operator.OperatorID[:]),
			Parallelism:    operator.Parallelism,
			MaxParallelism: operator.MaxParallelism,
		})
	}

	metadataSummary := &StorageCheckpointMetadataSummary{
		Version:        summary.Version,
		CheckpointID:   summary.CheckpointID,
		NumOperators:   summary.NumOperators,
		Operators:      operators,
		StateFilePaths: summary.StateFilePaths,
	}

	if summary.Properties != nil {
		metadataSummary.Properties = &StorageCheckpointPropertiesSummary{
			CheckpointType:  summary.Properties.CheckpointType,
			SharingStrategy: summary.Properties.SharingStrategy,
			Source:          summary.Properties.Source,
		}
	}

	return metadataSummary
}
