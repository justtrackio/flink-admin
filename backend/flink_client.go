package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type flinkCtxKey struct{}

func ProvideFlinkClient(ctx context.Context, config cfg.Config, logger log.Logger) (*FlinkClient, error) {
	return appctx.Provide(ctx, flinkCtxKey{}, func() (*FlinkClient, error) {
		httpClient := &http.Client{
			Timeout: 30 * time.Second,
		}

		return &FlinkClient{
			httpClient: httpClient,
			logger:     logger.WithChannel("flink_client"),
		}, nil
	})
}

type FlinkClient struct {
	httpClient *http.Client
	logger     log.Logger
}

// GetOverview fetches the cluster overview from /overview endpoint
func (c *FlinkClient) GetOverview(ctx context.Context, clusterURL string) (*FlinkOverview, error) {
	var overview FlinkOverview
	if err := c.get(ctx, clusterURL+"/overview", &overview); err != nil {
		return nil, fmt.Errorf("could not get overview: %w", err)
	}
	return &overview, nil
}

// GetConfig fetches the cluster configuration from /config endpoint
func (c *FlinkClient) GetConfig(ctx context.Context, clusterURL string) (*FlinkConfig, error) {
	var config FlinkConfig
	if err := c.get(ctx, clusterURL+"/config", &config); err != nil {
		return nil, fmt.Errorf("could not get config: %w", err)
	}
	return &config, nil
}

// ListJobs fetches all jobs from /jobs endpoint
func (c *FlinkClient) ListJobs(ctx context.Context, clusterURL string) ([]FlinkJob, error) {
	var response FlinkJobsResponse
	if err := c.get(ctx, clusterURL+"/jobs", &response); err != nil {
		return nil, fmt.Errorf("could not list jobs: %w", err)
	}
	return response.Jobs, nil
}

// GetJobDetail fetches detailed job information from /jobs/:jobid endpoint
func (c *FlinkClient) GetJobDetail(ctx context.Context, clusterURL string, jobID string) (*FlinkJobDetail, error) {
	var detail FlinkJobDetail
	if err := c.get(ctx, clusterURL+"/jobs/"+jobID, &detail); err != nil {
		return nil, fmt.Errorf("could not get job detail: %w", err)
	}
	return &detail, nil
}

// CancelJob cancels a running job via PATCH /jobs/:jobid?mode=cancel
func (c *FlinkClient) CancelJob(ctx context.Context, clusterURL string, jobID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, clusterURL+"/jobs/"+jobID+"?mode=cancel", nil)
	if err != nil {
		return fmt.Errorf("could not create cancel request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not execute cancel request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cancel job failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info(ctx, "job %s cancel request sent", jobID)
	return nil
}

// get is a helper method for GET requests with JSON response
func (c *FlinkClient) get(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("could not decode response: %w", err)
	}

	return nil
}
