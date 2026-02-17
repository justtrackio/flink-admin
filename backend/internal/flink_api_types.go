package internal

// FlinkCheckpointStatistics represents checkpoint statistics from /jobs/:jobid/checkpoints
type FlinkCheckpointStatistics struct {
	Counts   FlinkCheckpointCounts    `json:"counts"`
	Summary  FlinkCheckpointSummary   `json:"summary"`
	Latest   *FlinkCheckpointDetail   `json:"latest"`
	History  []FlinkCheckpointDetail  `json:"history"`
	Restored *FlinkRestoredCheckpoint `json:"restored"`
}

// FlinkCheckpointCounts contains checkpoint count statistics
type FlinkCheckpointCounts struct {
	Restored   int64 `json:"restored"`
	Total      int64 `json:"total"`
	InProgress int64 `json:"in_progress"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
}

// FlinkCheckpointSummary contains summary statistics for checkpoints
type FlinkCheckpointSummary struct {
	StateSize         FlinkCheckpointSummaryStats `json:"state_size"`
	EndToEndDuration  FlinkCheckpointSummaryStats `json:"end_to_end_duration"`
	AlignmentBuffered FlinkCheckpointSummaryStats `json:"alignment_buffered"`
	ProcessedData     FlinkCheckpointSummaryStats `json:"processed_data"`
	PersistedData     FlinkCheckpointSummaryStats `json:"persisted_data"`
}

// FlinkCheckpointSummaryStats contains min/max/avg statistics
type FlinkCheckpointSummaryStats struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
	Avg int64 `json:"avg"`
}

// FlinkCheckpointDetail represents detailed information about a checkpoint
type FlinkCheckpointDetail struct {
	Id                      int64  `json:"id"`
	Status                  string `json:"status"`
	IsSavepoint             bool   `json:"is_savepoint"`
	TriggerTimestamp        int64  `json:"trigger_timestamp"`
	LatestAckTimestamp      int64  `json:"latest_ack_timestamp"`
	StateSize               int64  `json:"state_size"`
	EndToEndDuration        int64  `json:"end_to_end_duration"`
	AlignmentBuffered       int64  `json:"alignment_buffered"`
	ProcessedData           int64  `json:"processed_data"`
	PersistedData           int64  `json:"persisted_data"`
	NumSubtasks             int    `json:"num_subtasks"`
	NumAcknowledgedSubtasks int    `json:"num_acknowledged_subtasks"`
	CheckpointType          string `json:"checkpoint_type"`
	ExternalPath            string `json:"external_path"`
	Discarded               bool   `json:"discarded"`
}

// FlinkRestoredCheckpoint contains information about the restored checkpoint
type FlinkRestoredCheckpoint struct {
	Id               int64  `json:"id"`
	RestoreTimestamp int64  `json:"restore_timestamp"`
	IsSavepoint      bool   `json:"is_savepoint"`
	ExternalPath     string `json:"external_path"`
}
