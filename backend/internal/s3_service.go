package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	s3Client *s3.Client
}

func ProvideS3Service(ctx context.Context, config cfg.Config, logger log.Logger) (*S3Service, error) {
	return appctx.Provide(ctx, s3ServiceCtxKey{}, func() (*S3Service, error) {
		var err error
		var s3Client *s3.Client

		if s3Client, err = gosoS3.ProvideClient(ctx, config, logger, "default"); err != nil {
			return nil, fmt.Errorf("could not create s3 client: %w", err)
		}

		return &S3Service{
			s3Client: s3Client,
		}, nil
	})
}

type StorageEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type StateEntry struct {
	Type         string     `json:"type"`
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	JobId        string     `json:"jobId,omitempty"`
	CheckpointId int64     `json:"checkpointId,omitempty"`
	LastModified *time.Time `json:"lastModified,omitempty"`
	Size         *int64     `json:"size,omitempty"`
}

type MetadataInfo struct {
	Exists       bool
	LastModified *time.Time
	Size         *int64
}

func buildMetadataS3URI(s3URI string) (string, error) {
	var err error
	var bucket string
	var prefix string

	if bucket, prefix, err = parseS3URI(s3URI); err != nil {
		return "", err
	}

	return "s3://" + bucket + "/" + buildMetadataKey(prefix), nil
}

func buildMetadataKey(prefix string) string {
	metadataKey := prefix
	if !strings.HasSuffix(metadataKey, "/") {
		metadataKey += "/"
	}

	return metadataKey + "_metadata"
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

// listCommonPrefixNames paginates through S3 ListObjectsV2 with a "/" delimiter and returns
// the directory names (common prefix entries with the base prefix and trailing slash stripped).
func (s *S3Service) listCommonPrefixNames(ctx context.Context, bucket, prefix string) ([]string, error) {
	var err error
	var names []string
	var result *s3.ListObjectsV2Output
	delimiter := "/"
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            &bucket,
			Prefix:            &prefix,
			Delimiter:         &delimiter,
			ContinuationToken: continuationToken,
		}

		if result, err = s.s3Client.ListObjectsV2(ctx, input); err != nil {
			return nil, fmt.Errorf("failed to list objects in S3: %w", err)
		}

		for _, commonPrefix := range result.CommonPrefixes {
			if commonPrefix.Prefix == nil {
				continue
			}

			name := strings.TrimPrefix(*commonPrefix.Prefix, prefix)
			name = strings.TrimSuffix(name, "/")

			if name == "" {
				continue
			}

			names = append(names, name)
		}

		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	return names, nil
}

// ListStorageCheckpoints lists checkpoint/savepoint directories in S3 storage.
// It uses ListObjectsV2 with delimiter "/" to get only top-level "directories" (common prefixes).
func (s *S3Service) ListStorageCheckpoints(ctx context.Context, s3URI string) ([]StorageEntry, error) {
	var err error
	var bucket string
	var prefix string
	var names []string

	if s3URI == "" {
		return []StorageEntry{}, nil
	}

	if bucket, prefix, err = parseS3URI(s3URI); err != nil {
		return nil, fmt.Errorf("failed to parse S3 URI: %w", err)
	}

	if names, err = s.listCommonPrefixNames(ctx, bucket, prefix); err != nil {
		return nil, err
	}

	entries := make([]StorageEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, StorageEntry{
			Name: name,
			Path: "s3://" + bucket + "/" + prefix + name + "/",
		})
	}

	return entries, nil
}

// ListJobDirectories lists all job ID directories under a given checkpoint base path.
// Returns a list of dashless job IDs found as subdirectories.
func (s *S3Service) ListJobDirectories(ctx context.Context, s3URI string) ([]string, error) {
	var err error
	var bucket string
	var prefix string
	var jobIds []string

	if s3URI == "" {
		return []string{}, nil
	}

	if bucket, prefix, err = parseS3URI(s3URI); err != nil {
		return nil, fmt.Errorf("failed to parse S3 URI: %w", err)
	}

	if jobIds, err = s.listCommonPrefixNames(ctx, bucket, prefix); err != nil {
		return nil, err
	}

	return jobIds, nil
}

// GetMetadataInfo checks if a checkpoint directory contains a _metadata file and returns its info
func (s *S3Service) GetMetadataInfo(ctx context.Context, s3URI string) (*MetadataInfo, error) {
	var err error
	var bucket string
	var prefix string
	var result *s3.HeadObjectOutput

	if s3URI == "" {
		return &MetadataInfo{Exists: false}, nil
	}

	if bucket, prefix, err = parseS3URI(s3URI); err != nil {
		return nil, fmt.Errorf("failed to parse S3 URI: %w", err)
	}

	metadataKey := buildMetadataKey(prefix)

	// Use HeadObject to check if the _metadata file exists
	input := &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &metadataKey,
	}

	if result, err = s.s3Client.HeadObject(ctx, input); err != nil {
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

func (s *S3Service) OpenMetadata(ctx context.Context, s3URI string) (io.ReadCloser, error) {
	var err error
	var bucket string
	var prefix string
	var result *s3.GetObjectOutput

	if s3URI == "" {
		return nil, fmt.Errorf("s3 URI is empty")
	}

	if bucket, prefix, err = parseS3URI(s3URI); err != nil {
		return nil, fmt.Errorf("failed to parse S3 URI: %w", err)
	}

	metadataKey := buildMetadataKey(prefix)
	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &metadataKey,
	}

	if result, err = s.s3Client.GetObject(ctx, input); err != nil {
		return nil, fmt.Errorf("failed to get metadata object for s3://%s/%s: %w", bucket, metadataKey, err)
	}

	return result.Body, nil
}

// ListValidCheckpoints lists all valid checkpoints (chk-* with _metadata) for a given job ID
func (s *S3Service) ListValidCheckpoints(ctx context.Context, checkpointBasePath string, jobId string) ([]StateEntry, error) {
	var err error
	var allCheckpoints []StorageEntry
	var metadataInfo *MetadataInfo

	if checkpointBasePath == "" || jobId == "" {
		return []StateEntry{}, nil
	}

	// Construct job-specific checkpoint path
	jobPath := checkpointBasePath
	if !strings.HasSuffix(jobPath, "/") {
		jobPath += "/"
	}
	jobPath += jobId

	// First, list all checkpoint directories (chk-*)
	if allCheckpoints, err = s.ListStorageCheckpoints(ctx, jobPath); err != nil {
		return nil, fmt.Errorf("failed to list checkpoints for job %s: %w", jobId, err)
	}

	// Filter checkpoints that start with "chk-" and have _metadata file
	var validCheckpoints []StateEntry
	for _, checkpoint := range allCheckpoints {
		if !strings.HasPrefix(checkpoint.Name, "chk-") {
			continue
		}

		// Get metadata file info
		if metadataInfo, err = s.GetMetadataInfo(ctx, checkpoint.Path); err != nil {
			return nil, fmt.Errorf("failed to get metadata for checkpoint %s: %w", checkpoint.Path, err)
		}

		if !metadataInfo.Exists {
			continue
		}

		validCheckpoints = append(validCheckpoints, StateEntry{
			Type:         StateTypeCheckpoint,
			Name:         checkpoint.Name,
			Path:         checkpoint.Path,
			JobId:        jobId,
			LastModified: metadataInfo.LastModified,
			Size:         metadataInfo.Size,
		})
	}

	return validCheckpoints, nil
}
