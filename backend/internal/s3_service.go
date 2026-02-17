package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"

	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
)

type s3ServiceCtxKey struct{}

type S3Service struct {
	logger   log.Logger
	s3Client *s3.Client
}

func ProvideS3Service(ctx context.Context, config cfg.Config, logger log.Logger) (*S3Service, error) {
	return appctx.Provide(ctx, s3ServiceCtxKey{}, func() (*S3Service, error) {
		s3Client, err := gosoS3.ProvideClient(ctx, config, logger, "default")
		if err != nil {
			return nil, fmt.Errorf("could not create s3 client: %w", err)
		}

		return &S3Service{
			logger:   logger.WithChannel("s3_service"),
			s3Client: s3Client,
		}, nil
	})
}

type StorageEntry struct {
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	JobId        string     `json:"jobId,omitempty"`
	LastModified *time.Time `json:"lastModified,omitempty"`
	Size         *int64     `json:"size,omitempty"`
}

type MetadataInfo struct {
	Exists       bool
	LastModified *time.Time
	Size         *int64
}

// parseS3URI parses an S3 URI like "s3://bucket/prefix/path" into bucket and prefix
func parseS3URI(uri string) (bucket, prefix string, err error) {
	if !strings.HasPrefix(uri, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URI format: %s (must start with s3://)", uri)
	}

	// Remove "s3://" prefix
	uri = strings.TrimPrefix(uri, "s3://")

	// Split by first "/"
	parts := strings.SplitN(uri, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid S3 URI: missing bucket name")
	}

	bucket = parts[0]
	if len(parts) == 2 {
		prefix = parts[1]
	}

	// Ensure prefix ends with "/" for directory listing
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return bucket, prefix, nil
}

// ListStorageCheckpoints lists checkpoint/savepoint directories in S3 storage
// It uses ListObjectsV2 with delimiter "/" to get only top-level "directories" (common prefixes)
func (s *S3Service) ListStorageCheckpoints(ctx context.Context, s3URI string) ([]StorageEntry, error) {
	if s3URI == "" {
		return []StorageEntry{}, nil
	}

	bucket, prefix, err := parseS3URI(s3URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 URI: %w", err)
	}

	s.logger.Info(ctx, "listing checkpoints in s3://%s/%s", bucket, prefix)

	var entries []StorageEntry
	delimiter := "/"
	var continuationToken *string

	// Paginate through results
	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            &bucket,
			Prefix:            &prefix,
			Delimiter:         &delimiter,
			ContinuationToken: continuationToken,
		}

		result, err := s.s3Client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in S3: %w", err)
		}

		// Process common prefixes (directories)
		for _, commonPrefix := range result.CommonPrefixes {
			if commonPrefix.Prefix == nil {
				continue
			}

			fullPrefix := *commonPrefix.Prefix

			// Extract directory name (e.g., "chk-1/", "savepoint-abc123/")
			dirName := strings.TrimPrefix(fullPrefix, prefix)
			dirName = strings.TrimSuffix(dirName, "/")

			if dirName == "" {
				continue
			}

			entries = append(entries, StorageEntry{
				Name: dirName,
				Path: "s3://" + bucket + "/" + fullPrefix,
			})
		}

		// Check if there are more results
		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	s.logger.Info(ctx, "found %d checkpoint/savepoint directories", len(entries))

	return entries, nil
}

// ListJobDirectories lists all job ID directories under a given checkpoint base path
// Returns a list of dashless job IDs found as subdirectories
func (s *S3Service) ListJobDirectories(ctx context.Context, s3URI string) ([]string, error) {
	if s3URI == "" {
		return []string{}, nil
	}

	bucket, prefix, err := parseS3URI(s3URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 URI: %w", err)
	}

	s.logger.Info(ctx, "listing job directories in s3://%s/%s", bucket, prefix)

	var jobIds []string
	delimiter := "/"
	var continuationToken *string

	// Paginate through results
	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            &bucket,
			Prefix:            &prefix,
			Delimiter:         &delimiter,
			ContinuationToken: continuationToken,
		}

		result, err := s.s3Client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in S3: %w", err)
		}

		// Process common prefixes (job ID directories)
		for _, commonPrefix := range result.CommonPrefixes {
			if commonPrefix.Prefix == nil {
				continue
			}

			fullPrefix := *commonPrefix.Prefix

			// Extract directory name (job ID)
			jobId := strings.TrimPrefix(fullPrefix, prefix)
			jobId = strings.TrimSuffix(jobId, "/")

			if jobId == "" {
				continue
			}

			jobIds = append(jobIds, jobId)
		}

		// Check if there are more results
		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	s.logger.Info(ctx, "found %d job directories", len(jobIds))

	return jobIds, nil
}

// GetMetadataInfo checks if a checkpoint directory contains a _metadata file and returns its info
func (s *S3Service) GetMetadataInfo(ctx context.Context, s3URI string) (*MetadataInfo, error) {
	if s3URI == "" {
		return &MetadataInfo{Exists: false}, nil
	}

	bucket, prefix, err := parseS3URI(s3URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 URI: %w", err)
	}

	// Ensure prefix ends with _metadata
	metadataKey := prefix
	if !strings.HasSuffix(metadataKey, "/") {
		metadataKey += "/"
	}
	metadataKey += "_metadata"

	s.logger.Debug(ctx, "checking for metadata file: s3://%s/%s", bucket, metadataKey)

	// Use HeadObject to check if the _metadata file exists
	input := &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &metadataKey,
	}

	result, err := s.s3Client.HeadObject(ctx, input)
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return &MetadataInfo{Exists: false}, nil
		}

		return nil, fmt.Errorf("failed to get metadata head for s3://%s/%s: %w", bucket, metadataKey, err)
	}

	return &MetadataInfo{
		Exists:       true,
		LastModified: result.LastModified,
		Size:         result.ContentLength,
	}, nil
}

// ListValidCheckpoints lists all valid checkpoints (chk-* with _metadata) for a given job ID
func (s *S3Service) ListValidCheckpoints(ctx context.Context, checkpointBasePath string, jobId string) ([]StorageEntry, error) {
	if checkpointBasePath == "" || jobId == "" {
		return []StorageEntry{}, nil
	}

	// Construct job-specific checkpoint path
	jobPath := checkpointBasePath
	if !strings.HasSuffix(jobPath, "/") {
		jobPath += "/"
	}
	jobPath += jobId

	s.logger.Info(ctx, "listing valid checkpoints for job %s in %s", jobId, jobPath)

	// First, list all checkpoint directories (chk-*)
	allCheckpoints, err := s.ListStorageCheckpoints(ctx, jobPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints for job %s: %w", jobId, err)
	}

	// Filter checkpoints that start with "chk-" and have _metadata file
	var validCheckpoints []StorageEntry
	for _, checkpoint := range allCheckpoints {
		if !strings.HasPrefix(checkpoint.Name, "chk-") {
			continue
		}

		// Get metadata file info
		metadataInfo, err := s.GetMetadataInfo(ctx, checkpoint.Path)
		if err != nil {
			s.logger.Warn(ctx, "failed to check metadata for %s: %v", checkpoint.Path, err)

			continue
		}

		if metadataInfo.Exists {
			checkpoint.JobId = jobId
			checkpoint.LastModified = metadataInfo.LastModified
			checkpoint.Size = metadataInfo.Size
			validCheckpoints = append(validCheckpoints, checkpoint)
		}
	}

	s.logger.Info(ctx, "found %d valid checkpoints for job %s", len(validCheckpoints), jobId)

	return validCheckpoints, nil
}
