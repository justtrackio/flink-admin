package main

import (
	"time"
)

// Cluster represents a Flink cluster discovered from Kubernetes FlinkDeployment CRs
type Cluster struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	URL       string `json:"url"`

	// Spec highlights from FlinkDeployment CR
	Image        string `json:"image,omitempty"`
	FlinkVersion string `json:"flinkVersion,omitempty"`

	// Status highlights from FlinkDeployment CR
	LifecycleState             string `json:"lifecycleState,omitempty"`
	JobManagerDeploymentStatus string `json:"jobManagerDeploymentStatus,omitempty"`
	JobName                    string `json:"jobName,omitempty"`
	JobState                   string `json:"jobState,omitempty"`
}

// ID returns the unique identifier for the cluster (namespace/name)
func (c *Cluster) ID() string {
	return c.Namespace + "/" + c.Name
}

// FlinkJob represents a job in the Flink cluster (from /jobs endpoint)
type FlinkJob struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"` // RUNNING, CANCELED, FAILED, FINISHED, etc.
	StartTime int64  `json:"start-time"`
	EndTime   int64  `json:"end-time"`
	Duration  int64  `json:"duration"`
}

// FlinkJobsResponse is the response from /jobs endpoint
type FlinkJobsResponse struct {
	Jobs []FlinkJob `json:"jobs"`
}

// FlinkJobDetail represents detailed job information from /jobs/:jobid endpoint
type FlinkJobDetail struct {
	Jid          string           `json:"jid"`
	Name         string           `json:"name"`
	State        string           `json:"state"`
	StartTime    int64            `json:"start-time"`
	EndTime      int64            `json:"end-time"`
	Duration     int64            `json:"duration"`
	Now          int64            `json:"now"`
	Timestamps   map[string]int64 `json:"timestamps"`
	Vertices     []JobVertex      `json:"vertices"`
	StatusCounts map[string]int   `json:"status-counts"`
	Plan         map[string]any   `json:"plan"`
}

// JobVertex represents a vertex in the job execution plan
type JobVertex struct {
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	Parallelism int              `json:"parallelism"`
	Status      string           `json:"status"`
	StartTime   int64            `json:"start-time"`
	EndTime     int64            `json:"end-time"`
	Duration    int64            `json:"duration"`
	Tasks       map[string]int   `json:"tasks"`
	Metrics     JobVertexMetrics `json:"metrics"`
}

// JobVertexMetrics contains metrics for a job vertex
type JobVertexMetrics struct {
	ReadBytes    int64 `json:"read-bytes"`
	ReadRecords  int64 `json:"read-records"`
	WriteBytes   int64 `json:"write-bytes"`
	WriteRecords int64 `json:"write-records"`
}

// FlinkOverview represents cluster overview from /overview endpoint
type FlinkOverview struct {
	Taskmanagers   int    `json:"taskmanagers"`
	SlotsTotal     int    `json:"slots-total"`
	SlotsAvailable int    `json:"slots-available"`
	JobsRunning    int    `json:"jobs-running"`
	JobsFinished   int    `json:"jobs-finished"`
	JobsCancelled  int    `json:"jobs-cancelled"`
	JobsFailed     int    `json:"jobs-failed"`
	FlinkVersion   string `json:"flink-version"`
	FlinkCommit    string `json:"flink-commit"`
}

// FlinkConfig represents cluster configuration from /config endpoint
type FlinkConfig struct {
	RefreshInterval int64  `json:"refresh-interval"`
	TimezoneName    string `json:"timezone-name"`
	TimezoneOffset  int    `json:"timezone-offset"`
	FlinkVersion    string `json:"flink-version"`
	FlinkRevision   string `json:"flink-revision"`
}

// ClusterInfo represents combined cluster information for API response
type ClusterInfo struct {
	Name     string        `json:"name"`
	URL      string        `json:"url"`
	Overview FlinkOverview `json:"overview"`
	Config   FlinkConfig   `json:"config"`
}

// CancelJobResponse is the response from canceling a job
type CancelJobResponse struct {
	Status string `json:"status"`
	JobId  string `json:"job_id"`
}

// DateTime wraps time.Time for JSON serialization/deserialization
type DateTime struct {
	time.Time
}

func (dt *DateTime) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == "null" || str == `""` {
		return nil
	}

	// Remove quotes
	str = str[1 : len(str)-1]

	// Try RFC3339 format first
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		// Try date-only format
		t, err = time.Parse(time.DateOnly, str)
		if err != nil {
			return err
		}
	}

	dt.Time = t
	return nil
}

func (dt DateTime) MarshalJSON() ([]byte, error) {
	if dt.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + dt.Format(time.RFC3339) + `"`), nil
}
