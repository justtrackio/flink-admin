package internal

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

// GetCheckpoints fetches checkpoint statistics from /jobs/:jobid/checkpoints endpoint
func (c *FlinkClient) GetCheckpoints(ctx context.Context, clusterURL string, jobID string) (*FlinkCheckpointStatistics, error) {
	var stats FlinkCheckpointStatistics
	if err := c.get(ctx, clusterURL+"/jobs/"+jobID+"/checkpoints", &stats); err != nil {
		return nil, fmt.Errorf("could not get checkpoints: %w", err)
	}

	return &stats, nil
}

// get is a helper method for GET requests with JSON response
func (c *FlinkClient) get(ctx context.Context, url string, target any) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("read error response body: %w", readErr)
		}

		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("could not decode response: %w", err)
	}

	return nil
}
