// FlinkDeployment CRD types for SSE streaming

export interface FlinkDeploymentMetadata {
  name: string;
  namespace: string;
  uid: string;
  creationTimestamp: string;
  labels?: Record<string, string>;
}

export interface FlinkDeploymentResource {
  cpu: number;
  memory: string;
}

export interface FlinkDeploymentJob {
  jarURI: string;
  entryClass: string;
  args?: string[];
  parallelism: number;
  state: string;
  upgradeMode: string;
}

export interface FlinkDeploymentJobManager {
  replicas: number;
  resource: FlinkDeploymentResource;
}

export interface FlinkDeploymentTaskManager {
  resource: FlinkDeploymentResource;
}

export interface FlinkDeploymentSpec {
  image: string;
  flinkVersion: string;
  flinkConfiguration?: Record<string, string>;
  job: FlinkDeploymentJob;
  jobManager: FlinkDeploymentJobManager;
  taskManager: FlinkDeploymentTaskManager;
}

export interface FlinkClusterInfo {
  'flink-version': string;
  'flink-revision': string;
  'total-cpu': string;
  'total-memory': string;
}

export interface FlinkCheckpointInfo {
  lastPeriodicCheckpointTimestamp?: number;
}

export interface FlinkSavepointInfo {
  lastPeriodicSavepointTimestamp?: number;
  savepointHistory?: string[];
}

export interface FlinkJobStatus {
  state: string;
  startTime?: string;
  updateTime?: string;
  jobId?: string;
  checkpointInfo?: FlinkCheckpointInfo;
  savepointInfo?: FlinkSavepointInfo;
}

export interface FlinkDeploymentStatus {
  lifecycleState: string;
  jobManagerDeploymentStatus?: string;
  jobStatus?: FlinkJobStatus;
  clusterInfo?: FlinkClusterInfo;
}

export interface FlinkDeployment {
  kind: string;
  apiVersion: string;
  metadata: FlinkDeploymentMetadata;
  spec: FlinkDeploymentSpec;
  status?: FlinkDeploymentStatus;
}

export type DeploymentEventType = 'ADDED' | 'MODIFIED' | 'DELETED';

export interface DeploymentEvent {
  type: DeploymentEventType;
  deployment: FlinkDeployment;
}

// Flink REST API Checkpoint Types

export interface FlinkCheckpointCounts {
  restored: number;
  total: number;
  in_progress: number;
  completed: number;
  failed: number;
}

export interface FlinkCheckpointSummaryStats {
  min: number;
  max: number;
  avg: number;
}

export interface FlinkCheckpointSummary {
  state_size: FlinkCheckpointSummaryStats;
  end_to_end_duration: FlinkCheckpointSummaryStats;
  alignment_buffered: FlinkCheckpointSummaryStats;
  processed_data: FlinkCheckpointSummaryStats;
  persisted_data: FlinkCheckpointSummaryStats;
}

export interface FlinkCheckpointDetail {
  id: number;
  status: string;
  is_savepoint: boolean;
  trigger_timestamp: number;
  latest_ack_timestamp: number;
  state_size: number;
  end_to_end_duration: number;
  alignment_buffered: number;
  processed_data: number;
  persisted_data: number;
  num_subtasks: number;
  num_acknowledged_subtasks: number;
  checkpoint_type: string;
  external_path: string;
  discarded: boolean;
}

export interface FlinkRestoredCheckpoint {
  id: number;
  restore_timestamp: number;
  is_savepoint: boolean;
  external_path: string;
}

export interface FlinkCheckpointStatistics {
  counts: FlinkCheckpointCounts;
  summary: FlinkCheckpointSummary;
  latest: FlinkCheckpointDetail | null;
  history: FlinkCheckpointDetail[];
  restored: FlinkRestoredCheckpoint | null;
}

// S3 Storage Checkpoint Types

export interface StorageEntry {
  name: string;
  path: string;
  jobId?: string;
  lastModified?: string;
  size?: number;
}

export interface StorageCheckpointsResponse {
  jobId?: string;
  checkpointDir?: string;
  savepointDir?: string;
  checkpoints: StorageEntry[];
  savepoints: StorageEntry[];
}
